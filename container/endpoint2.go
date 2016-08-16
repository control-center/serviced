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
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
)

// ContainerEndpointOptions are options for the container endpoint
type ContainerEndpointOptions struct {
	HostID               string
	TenantID             string
	InstanceID           int
	IsShell              bool
	TCPMuxPort           uint16
	UseTLS               bool
	VirtualAddressSubnet string
}

// ContainerEndpoint manages import and export bindings for the instance.
type ContainerEndpoint struct {
	opts  ContainerEndpointOptions
	state *zkservice.State
	cache *proxyCache
	ports map[uint16]struct{}
	vifs  *VIFRegistry
}

// NewContainerEndpoint loads the service state and manages port bindings
// for the instance.
func NewContainerEndpoint(svc *service.Service, opts ContainerEndpointOptions) (*ContainerEndpoint, error) {

	ce := &ContainerEndpoint{
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
func (ce *ContainerEndpoint) loadState(svc *service.Service) (bool, error) {
	logger := log.WithFields(log.Fields{
		"HostID":     ce.opts.HostID,
		"ServiceID":  svc.ID,
		"InstanceID": ce.opts.InstanceID,
	})

	// get the hostname
	hostname, err := os.Hostname()
	if err != nil {
		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not get the hostname to check the docker id")
		return false, err
	}

	allowDirect := true

	if ce.opts.IsShell {

		// this is not a running instance so load whatever data is
		// available.

		ce.state = &zkservice.State{
			ServiceState: zkservice.ServiceState{
				DockerID: hostname,
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
			logger.WithFields(log.Fields{
				"Error": err,
			}).Debug("Cannot connect to the coordinator")
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
			ce.state, err = zkservice.MonitorState(cancel, conn, req, func(s *zkservice.State) bool {
				return strings.HasPrefix(s.DockerID, hostname)
			})
			errc <- err
		}()

		select {
		case err := <-errc:
			if err != nil {
				logger.WithFields(log.Fields{
					"Error": err,
				}).Debug("Could not load state")
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
func (ce *ContainerEndpoint) Run(cancel <-chan struct{}) {
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
	for _, bind := range ce.state.Imports {
		wg.Add(1)
		go func(bind zkservice.ImportBinding) {
			ce.AddImport(cancel, bind)
			wg.Done()
		}(bind)
	}

	wg.Wait()
}

// AddExport ensures that an export is registered for other services to bind
func (ce *ContainerEndpoint) AddExport(cancel <-chan struct{}, bind zkservice.ExportBinding) {
	logger := log.WithFields(log.Fields{
		"Application": bind.Application,
		"PortNumber":  bind.PortNumber,
		"Protocol":    bind.Protocol,
	})

	exp := zkservice.ExportDetails{
		ExportBinding: bind,
		PrivateIP:     ce.state.PrivateIP,
		InstanceID:    ce.state.InstanceID,
	}

	logger.Debug("Registering export")

	defer logger.Debug("Unregistered export")

	for {
		select {
		case conn := <-zzk.Connect("/", zzk.GetLocalConnection):
			if conn != nil {

				logger.Debug("Received coordinator connection")

				zkservice.RegisterExport(cancel, conn, ce.opts.TenantID, exp)
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
func (ce *ContainerEndpoint) AddImport(cancel <-chan struct{}, bind zkservice.ImportBinding) {
	logger := log.WithFields(log.Fields{
		"Application": bind.Application,
		"Purpose":     bind.Purpose,
	})

	logger.Debug("Tracking exports for endpoint")

	defer logger.Debug("Exited export tracking for endpoint")

	for {
		select {
		case conn := <-zzk.Connect("/", zzk.GetLocalConnection):
			if conn != nil {

				logger.Debug("Received coordinator connection")

				ch := zkservice.TrackExports(cancel, conn, ce.opts.TenantID, bind.Application)
				for exports := range ch {
					ce.SetExports(bind, exports)
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

// SetExports updates the proxy connections for an import port binding
func (ce *ContainerEndpoint) SetExports(bind zkservice.ImportBinding, exports []zkservice.ExportDetails) {
	logger := log.WithFields(log.Fields{
		"Application": bind.Application,
		"Purpose":     bind.Purpose,
	})

	logger.Debug("Updating exports for endpoint")

	if bind.Purpose == "import_all" {

		// set up a proxy for each endpoint

		for _, export := range exports {

			exLogger := logger.WithFields(log.Fields{
				"InstanceID": export.InstanceID,
			})

			// calculate the inbound port number
			port, err := bind.GetPortNumber(export.InstanceID)
			if err != nil {
				exLogger.WithFields(log.Fields{
					"Error": err,
				}).Error("Could not get port for instance")
				return
			}

			exLogger = exLogger.WithFields(log.Fields{
				"PortNumber": port,
			})

			// check if the port is in use by an export
			if _, ok := ce.ports[port]; ok {
				exLogger.Warn("Port is in use")
				continue
			}

			// update the proxy
			isNew, err := ce.cache.Set(bind.Application, port, export)
			if err != nil {
				exLogger.WithFields(log.Fields{
					"Error": err,
				}).Error("Could not update proxy")
				return
			}

			exLogger.Debug("Updated proxy")

			// set up the virtual address if this this a new export
			if isNew {
				virtualAddress, err := bind.GetVirtualAddress(export.InstanceID)
				if err != nil {
					exLogger.WithFields(log.Fields{
						"Error": err,
					}).Warn("Could not get virtual address")
					continue
				}

				if virtualAddress != "" {
					exLogger = exLogger.WithFields(log.Fields{
						"VirtualAddress": virtualAddress,
					})

					if err := ce.vifs.RegisterVirtualAddress(virtualAddress, fmt.Sprintf(":%d", port), export.Protocol); err != nil {
						exLogger.WithFields(log.Fields{
							"Error": err,
						}).Warn("Could not register virtual address")
						continue
					}

					exLogger.Debug("Registered virtual address")
				}
			}
		}
	} else {
		exLogger := logger

		// calculate the inbound port number
		port, err := bind.GetPortNumber(0)
		if err != nil {
			exLogger.WithFields(log.Fields{
				"Error": err,
			}).Error("Could not get port for application")
			return
		}

		exLogger = exLogger.WithFields(log.Fields{
			"PortNumber": port,
		})

		// check if the port is used by an export
		if _, ok := ce.ports[port]; ok {
			exLogger.Warn("Port is in use")
			return
		}

		// update the proxy
		if _, err := ce.cache.Set(bind.Application, port, exports...); err != nil {
			exLogger.WithFields(log.Fields{
				"Error": err,
			}).Error("Could not update proxy")
			return
		}

		exLogger.Debug("Updated proxy")
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
		allowDirect: allowDirect,
	}
}

// Set returns true if the key was created and an error
func (c *proxyCache) Set(application string, portNumber uint16, exports ...zkservice.ExportDetails) (bool, error) {
	logger := log.WithFields(log.Fields{
		"Application": application,
		"PortNumber":  portNumber,
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
			logger.WithFields(log.Fields{
				"Error": err,
			}).Debug("Could not open port")
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
			logger.WithFields(log.Fields{
				"Error": err,
			}).Debug("Could not start proxy")
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
