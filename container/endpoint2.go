// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package container

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry2"
	zkservice "github.com/control-center/serviced/zzk/service2"
)

// ContainerEndpointsOptions are options for the container endpoint
type ContainerEndpointsOptions struct {
	HostID               string
	TenantID             string
	InstanceID           int
	IsShell              bool
	TCPMuxPort           uint16
	UseTLS               bool
	VirtualAddressSubnet string
}

// ContainerEndpoints manages import and export bindings for the instance.
type ContainerEndpoints struct {
	opts  ContainerEndpointsOptions
	state *zkservice.State
	cache *proxyCache
	ports map[uint16]struct{}
	vifs  *VIFRegistry
}

// NewContainerEndpoints loads the service state and manages port bindings
// for the instance.
func NewContainerEndpoints(svc *service.Service, opts ContainerEndpointsOptions) (*ContainerEndpoints, error) {

	ce := &ContainerEndpoints{
		opts:  opts,
		ports: make(map[uint16]struct{}),
		vifs:  NewVIFRegistry(),
	}

	// load the state object
	allowDirect, err := ce.loadState(svc)
	if err != nil {
		return nil, err
	}

	// set up the proxy cache
	ce.cache = newProxyCache(opts.TenantID, opts.TCPMuxPort, opts.UseTLS, allowDirect)

	// set up virtual interface registry
	if err := ce.vifs.SetSubnet(opts.VirtualAddressSubnet); err != nil {
		return nil, err
	}

	return ce, nil
}

// loadState loads state information for the container
func (ce *ContainerEndpoints) loadState(svc *service.Service) (bool, error) {
	logger := plog.WithFields(log.Fields{
		"hostid":      ce.opts.HostID,
		"serviceid":   svc.ID,
		"servicename": svc.Name,
		"instanceid":  ce.opts.InstanceID,
	})

	allowDirect := true

	if ce.opts.IsShell {
		// get the hostname
		hostname, err := os.Hostname()
		if err != nil {
			logger.WithError(err).Debug("Could not get the hostname to check the docker id")
			return false, err
		}

		// this is not a running instance so load whatever data is
		// available.

		ce.state = &zkservice.State{
			ServiceState: zkservice.ServiceState{
				ContainerID: hostname,
			},
			HostID:     ce.opts.HostID,
			ServiceID:  svc.ID,
			InstanceID: ce.opts.InstanceID,
		}

		binds := []zkservice.ImportBinding{}
		for _, ep := range svc.Endpoints {
			if ep.Purpose != "export" {
				if ep.Purpose == "import_all" {
					allowDirect = false
				}
				binds = append(binds, zkservice.ImportBinding{
					Application:    ep.Application,
					Purpose:        ep.Purpose,
					PortNumber:     ep.PortNumber,
					VirtualAddress: ep.VirtualAddress,
				})
			}
		}

		ce.state.Imports = binds

		logger.Debug("Loaded state for shell instance")
	} else {

		// connect to the coordinator
		conn, err := zzk.GetLocalConnection("/")
		if err != nil {
			logger.WithError(err).Debug("Cannot connect to the coordination server")
			return false, err
		}

		logger.Debug("Connected to the coordination server")

		// get the state
		req := zkservice.StateRequest{
			PoolID:     svc.PoolID,
			HostID:     ce.opts.HostID,
			ServiceID:  svc.ID,
			InstanceID: ce.opts.InstanceID,
		}

		// wait for the state to be ready or time out
		timer := time.NewTimer(15 * time.Second)
		cancel := make(chan struct{})
		errc := make(chan error)

		go func() {
			var err error
			ce.state, err = zkservice.MonitorState(cancel, conn, req, func(s *zkservice.State, exists bool) bool {
				return exists && s.Started.After(s.Terminated)
			})
			errc <- err
		}()

		select {
		case err := <-errc:
			if err != nil {
				logger.WithError(err).Debug("Could not load state")
				return false, err
			}
		case <-timer.C:
			close(cancel)
			<-errc
			logger.Debug("Timeout waiting for state")
			return false, errors.New("timeout waiting for state")
		}

		for _, bind := range ce.state.Imports {
			if bind.Purpose == "import_all" {
				allowDirect = false
				break
			}
		}

		logger.Debug("Loaded state for service instance")
	}

	return allowDirect, nil
}

// Run manages the container endpoints
func (ce *ContainerEndpoints) Run(cancel <-chan struct{}) {
	wg := &sync.WaitGroup{}

	// register all of the exports
	for _, bind := range ce.state.Exports {
		ce.ports[bind.PortNumber] = struct{}{}
		wg.Add(1)
		go func(bind zkservice.ExportBinding) {
			ce.AddExport(cancel, bind)
			wg.Done()
		}(bind)
	}

	// track all of the imports
	// TODO: set up another tracker for cc exports
	wg.Add(1)
	go func() {
		ce.RunImportListener(cancel, ce.opts.TenantID, ce.state.Imports...)
		wg.Done()
	}()

	wg.Wait()
}

// AddExport ensures that an export is registered for other services to bind
func (ce *ContainerEndpoints) AddExport(cancel <-chan struct{}, bind zkservice.ExportBinding) {
	logger := plog.WithFields(log.Fields{
		"application": bind.Application,
		"portnumber":  bind.PortNumber,
		"protocol":    bind.Protocol,
	})

	exp := registry.ExportDetails{
		ExportBinding: bind,
		PrivateIP:     ce.state.PrivateIP,
		HostIP:        ce.state.HostIP,
		MuxPort:       ce.opts.TCPMuxPort,
		InstanceID:    ce.state.InstanceID,
	}

	logger.Debug("Registering export")
	defer logger.Debug("Unregistered export")

	for {
		select {
		case conn := <-zzk.Connect("/", zzk.GetLocalConnection):
			if conn != nil {

				logger.Debug("Received coordinator connection")

				registry.RegisterExport(cancel, conn, ce.opts.TenantID, exp)
				select {
				case <-cancel:
					return
				default:
				}
			}
		case <-cancel:
			return
		}
	}
}

// RunImportListener keeps track of the state of all matching imports
// TODO: isvcs imports should be stored under tenantID /net/export/cc
func (ce *ContainerEndpoints) RunImportListener(cancel <-chan struct{}, tenantID string, binds ...zkservice.ImportBinding) {
	plog.Debug("Running import listener")
	defer plog.Debug("Exited import listener")

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	// set up the import listener and add the import bindings
	listener := registry.NewImportListener(tenantID)
	for _, bind := range binds {
		rgx, err := regexp.Compile(fmt.Sprintf("^%s$", bind.Application))
		if err != nil {
			plog.WithField("regex", bind.Application).WithError(err).Warn("Could not compile regex; skipping")
			continue
		}
		ch := listener.AddTerm(rgx)
		wg.Add(1)
		go func(bind zkservice.ImportBinding) {
			for {
				select {
				case app := <-ch:
					wg.Add(1)
					go func() {
						// add an import listener to each matching application
						ce.AddImport(cancel, app, bind)
						wg.Done()
					}()
				case <-cancel:
					wg.Done()
					return
				}
			}
		}(bind)
	}

	// start the import listener
	for {
		select {
		case conn := <-zzk.Connect("/", zzk.GetLocalConnection):
			if conn != nil {

				plog.Debug("Received coordinator connection")
				listener.Run(cancel, conn)

				select {
				case <-cancel:
					return
				default:
				}
			}
		case <-cancel:
			return
		}
	}
}

// AddImport tracks exports for a given import binding
func (ce *ContainerEndpoints) AddImport(cancel <-chan struct{}, application string, bind zkservice.ImportBinding) {
	logger := plog.WithFields(log.Fields{
		"application":     application,
		"applicationglob": bind.Application,
		"purpose":         bind.Purpose,
	})
	logger.Debug("Tracking exports for endpoint")
	defer logger.Debug("Exited export tracking for endpoint")
	bind.Application = application

	for {
		select {
		case conn := <-zzk.Connect("/", zzk.GetLocalConnection):
			if conn != nil {

				logger.Debug("Received coordinator connection")

				ch := registry.TrackExports(cancel, conn, ce.opts.TenantID, application)
				for exports := range ch {
					ce.UpdateRemoteExports(bind, exports)
				}

				select {
				case <-cancel:
					return
				default:
				}
			}
		case <-cancel:
			return
		}
	}
}

// UpdateRemoteExports updates the proxy connections for an import port binding
func (ce *ContainerEndpoints) UpdateRemoteExports(bind zkservice.ImportBinding, exports []registry.ExportDetails) {
	logger := plog.WithFields(log.Fields{
		"application": bind.Application,
		"purpose":     bind.Purpose,
	})
	logger.Debug("Updating exports for endpoint")

	if bind.Purpose == "import_all" {

		// keep track of the number of port collisions
		// 1 means it is probably importing itself (within a cluster)
		// >1 means there is a port collision
		collisionCount := 0

		// set up a proxy for each endpoint

		for _, export := range exports {

			exLogger := logger.WithField("instanceid", export.InstanceID)

			// calculate the inbound port number
			// Parse the port template with the provided instance id.
			//  - If that is not set, then increment the default port number by
			//    the instance id.
			//  - Otherwise, just use the port number described by the export.
			var port uint16
			if bind.PortTemplate != "" {
				var err error
				port, err = bind.GetPortNumber(export.InstanceID)
				if err != nil {
					exLogger.WithError(err).Error("Could not calculate inbound port number; not importing endpoint")
					return
				}
				exLogger.Debug("Setting the port to the value defined on the template")
			} else if bind.PortNumber > 0 {
				exLogger.Debug("Port template not defined, setting port number to base + instance id")
				port = bind.PortNumber + uint16(export.InstanceID)
			} else {
				exLogger.Debug("Port number not set, using the export's port number")
				port = export.PortNumber
			}

			exLogger = exLogger.WithField("portnumber", port)

			// check if the port is in use by a previously registered export.
			// (see ce.state.Exports)
			if _, ok := ce.ports[port]; ok {
				if collisionCount > 1 {
					exLogger.Error("Port is already being exposed by the service instance")
				} else {
					exLogger.Debug("Port is already being exposed by the service instance")
				}
				collisionCount++
				continue
			}

			// update the proxy; returns a boolean if a new proxy was created.
			isNew, err := ce.cache.Set(bind.Application, port, export)
			if err != nil {
				exLogger.WithError(err).Error("Could not update proxy")
				return
			}

			exLogger.Debug("Updated proxy")

			// set up the virtual address if this this a new export
			if isNew {
				virtualAddress, err := bind.GetVirtualAddress(export.InstanceID)
				if err != nil {
					exLogger.WithError(err).Warn("Could not get virtual address")
					continue
				}

				if virtualAddress != "" {

					exLogger = exLogger.WithField("virtualaddress", virtualAddress)
					if err := ce.vifs.RegisterVirtualAddress(virtualAddress, fmt.Sprintf("%d", port), export.Protocol); err != nil {
						exLogger.WithError(err).Warn("Could not register virtual address")
						continue
					}

					exLogger.Debug("Registered virtual address")
				}
			}
		}
	} else {
		exLogger := logger

		// calculate the inbound port number, this is based on the port
		// template being set.
		port, err := bind.GetPortNumber(0)
		if port == 0 {
			if bind.PortNumber > 0 {
				exLogger.WithError(err).Debug("Could not calculate import port to map export endpoint, falling back to default import")
				port = bind.PortNumber
			} else if len(exports) > 0 {
				exLogger.WithError(err).Debug("Could not calculate import port to map export endpoint, using export port")
				port = exports[0].PortNumber
			} else {
				exLogger.WithError(err).Debug("Cannot update proxy to an empty list")
				return
			}
		}

		exLogger = exLogger.WithField("portnumber", port)

		// check if the port is used by an export
		if _, ok := ce.ports[port]; ok {
			exLogger.Error("Port is already being exposed by the service instance")
			return
		}

		// update the proxy
		isNew, err := ce.cache.Set(bind.Application, port, exports...)
		if err != nil {
			exLogger.WithError(err).Error("Could not update proxy")
			return
		}

		exLogger.Debug("Updated proxy")

		// set up virtual address if this is a new export
		if isNew {
			virtualAddress, err := bind.GetVirtualAddress(0)
			if err != nil {
				exLogger.WithError(err).Warn("Could not get virtual address")
				return
			}

			if virtualAddress != "" {

				exLogger = exLogger.WithField("virtualaddress", virtualAddress)
				if err := ce.vifs.RegisterVirtualAddress(virtualAddress, fmt.Sprintf(":%d", port), "tcp"); err != nil {
					exLogger.WithError(err).Warn("Could not register virtual address")
					return
				}
			}

			exLogger.Debug("Registered virtual address")
		}
	}
}

type proxyKey struct {
	Application string
	PortNumber  uint16
}

type proxyCache struct {
	mu    *sync.Mutex
	cache map[proxyKey]*proxy

	tenantID    string
	tcpMuxPort  uint16
	useTLS      bool
	allowDirect bool
}

func newProxyCache(tenantID string, tcpMuxPort uint16, useTLS, allowDirect bool) *proxyCache {
	return &proxyCache{
		mu:          &sync.Mutex{},
		cache:       make(map[proxyKey]*proxy),
		tenantID:    tenantID,
		tcpMuxPort:  tcpMuxPort,
		useTLS:      useTLS,
		allowDirect: allowDirect,
	}
}

// Set returns true if the key was created and an error
func (c *proxyCache) Set(application string, portNumber uint16, exports ...registry.ExportDetails) (bool, error) {
	logger := plog.WithFields(log.Fields{
		"application": application,
		"portnumber":  portNumber,
	})

	c.mu.Lock()
	defer c.mu.Unlock()

	key := proxyKey{
		Application: application,
		PortNumber:  portNumber,
	}

	// check if the key exists
	prxy, ok := c.cache[key]
	if !ok {

		logger.Debug("Setting up new proxy")

		// start the listener on the provided port
		listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", portNumber))
		if err != nil {
			logger.WithError(err).Debug("Could not open port")
			return false, err
		}

		logger.Debug("Started port listener")

		// create the proxy
		prxy, err = newProxy(
			fmt.Sprintf("%s-%d", application, portNumber),
			fmt.Sprintf("%s-%s-%d", c.tenantID, application, portNumber),
			c.tcpMuxPort,
			c.useTLS,
			listener,
			c.allowDirect,
		)
		if err != nil {
			logger.WithError(err).Debug("Could not start proxy")
			return false, err
		}

		logger.Debug("Created new proxy for port")
		c.cache[key] = prxy

	}

	// update the proxy addresses
	addresses := make([]addressTuple, len(exports))
	for i, export := range exports {
		addresses[i] = addressTuple{
			host:          export.HostIP,
			containerAddr: fmt.Sprintf("%s:%d", export.PrivateIP, export.PortNumber),
		}
	}
	prxy.SetNewAddresses(addresses)

	logger.Debug("Set exports for proxy")

	return !ok, nil
}
