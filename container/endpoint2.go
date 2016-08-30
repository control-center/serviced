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
			logger.WithError(err).Debug("Cannot connect to the coordinator")
			return false, err
		}

		logger.Debug("Connected to the coordinator")

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
	listener := registry.NewImportListener(ce.opts.TenantID)
	for _, bind := range ce.state.Imports {

		// compile the term as a regex
		rgx, err := regexp.Compile(bind.Application)
		if err != nil {
			plog.WithField("searchterm", bind.Application).WithError(err).Warn("Could not compile term, skipping import")
			continue
		}

		ch := listener.AddTerm(rgx)
		wg.Add(1)
		go func(bind zkservice.ImportBinding) {

			// set up a listener for each matching application
			for {
				select {
				case app := <-ch:
					wg.Add(1)
					go func() {
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
	wg.Add(1)
	go func() {
		ce.RunImportListener(cancel, listener)
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

// RunImportListener starts and persists the import listener
func (ce *ContainerEndpoints) RunImportListener(cancel <-chan struct{}, listener *registry.ImportListener) {
	plog.Debug("Running import listener")
	defer plog.Debug("Exited import listener")

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
		"applciation":     application,
		"applicationglob": bind.Application,
		"purpose":         bind.Purpose,
	})
	logger.Debug("Tracking exports for endpoint")
	defer logger.Debug("Exited export tracking for endpoint")

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
		collisionCount := 0

		// set up a proxy for each endpoint

		for _, export := range exports {

			exLogger := logger.WithField("instanceid", export.InstanceID)

			// calculate the inbound port number
			var port uint16
			var err error

			if bind.PortTemplate != "" {
				port, err = bind.GetPortNumber(export.InstanceID)
				if err != nil {
					exLogger.WithError(err).Error("Could not get port for instance")
					return
				}
			} else {
				port = bind.PortNumber + uint16(export.InstanceID)
			}

			exLogger = exLogger.WithField("portnumber", port)

			// check if the port is in use by an export
			if _, ok := ce.ports[port]; ok {
				if collisionCount > 1 {
					exLogger.Error("Port is in use")
				} else {
					exLogger.Debug("Port is in use")
				}
				collisionCount++
				continue
			}

			// update the proxy
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
	} else if len(exports) > 0 {
		exLogger := logger

		// calculate the inbound port number
		port, err := bind.GetPortNumber(0)
		if err != nil {
			exLogger.WithError(err).Error("Could not get port for application")
			return
		}
		if port == 0 {
			port = exports[0].PortNumber
		}

		exLogger = exLogger.WithField("portnumber", port)

		// check if the port is used by an export
		if _, ok := ce.ports[port]; ok {
			exLogger.Error("Port is in use")
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
