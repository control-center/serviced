// Copyright 2014 The Serviced Authors.
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

package web

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/control-center/serviced/proxy"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	domainService "github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

var (
	allportsLock sync.RWMutex
	allports     map[string]chan bool // map of port number to channel that destroys the server
	cpDao        dao.ControlPlane
)

func init() {
	allports = make(map[string]chan bool)
}

// Removes the port from our local cache and updates the service so the UI will flip to "disabled".
//  Only needs to be called if the port is being disabled unexpectedly due to an error
func disablePort(publicEndpointKey service.PublicEndpointKey) {
	//TODO: Add control plane methods to enable/disable public endpoints so we don't have to do a GetService and then UpdateService

	// remove the port from our local cache
	delete(allports, publicEndpointKey.Name())

	// find the endpoint that matches this port number for this service (there will only be 1)
	var myService domainService.Service
	var myEndpoint domainService.ServiceEndpoint
	var unused int
	cpDao.GetService(publicEndpointKey.ServiceID(), &myService)
	for _, endpoint := range myService.Endpoints {
		for _, endpointPort := range endpoint.PortList {
			if endpointPort.PortAddr == publicEndpointKey.Name() {
				myEndpoint = endpoint
			}
		}
	}

	// disable port
	myService.EnablePort(myEndpoint.Name, publicEndpointKey.Name(), false)
	if err := cpDao.UpdateService(myService, &unused); err != nil {
		glog.Errorf("Error in disablePort(%s): %v", string(publicEndpointKey), err)
	}
}

func (sc *ServiceConfig) ServePublicPorts(shutdown <-chan (interface{}), dao dao.ControlPlane) {
	cpDao = dao
	go sc.syncAllPublicPorts(shutdown)
}

func (sc *ServiceConfig) createPublicPortServer(publicEndpointKey service.PublicEndpointKey, stopChan chan bool, shutdown <-chan (interface{})) error {
	port := publicEndpointKey.Name()
	listener, err := net.Listen("tcp", port)

	if err != nil {
		glog.Errorf("Could not setup TCP listener for port %s for public endpoint %s - %s", port, publicEndpointKey, err)
		disablePort(publicEndpointKey)
		return err
	}
	glog.Infof("Listening on port %s", port)

	go func() {
		for {
			// accept connection on public port
			localConn, err := listener.Accept()
			if err != nil {
				glog.V(1).Infof("Stopping accept on port %s", port)
				return
			}

			// lookup remote endpoint for this public port
			pepEPInfo, err := sc.getPublicEndpoint(fmt.Sprintf("%s-%d", publicEndpointKey.Name(), int(publicEndpointKey.Type())))
			if err != nil {
				// This happens if an endpoint is accessed and the containers have died or not come up yet.
				glog.Errorf("Error retrieving public endpoint %s:  %s", publicEndpointKey, err)
				// close the accepted connection and continue waiting for connections.
				if err := localConn.Close(); err != nil {
					glog.Errorf("Error closing client connection: %s", err)
				}
				continue
			}

			// setup remote connection
			var remoteAddr string
			_, isLocalContainer := sc.localAddrs[pepEPInfo.hostIP]
			if isLocalContainer {
				remoteAddr = fmt.Sprintf("%s:%d", pepEPInfo.privateIP, pepEPInfo.epPort)
			} else {
				remoteAddr = fmt.Sprintf("%s:%d", pepEPInfo.hostIP, sc.muxPort)
			}
			remoteConn, err := sc.getRemoteConnection(remoteAddr, isLocalContainer, sc.muxPort, pepEPInfo.privateIP, pepEPInfo.epPort, sc.muxTLS && (sc.muxPort > 0))
			if err != nil {
				glog.Errorf("Error getting remote connection for public endpoint %s: %v", publicEndpointKey, err)
				continue
			}

			glog.V(2).Infof("Established remote connection to %s", remoteConn.RemoteAddr())

			// Serve proxied requests/responses.  We pass our own port stop channel so that
			// all proxy loops end when our port is shutdown.
			go proxy.ProxyLoop(localConn, remoteConn, stopChan)
		}
	}()

	go func() {
		// Wait for shutdown, then kill all your connections
		select {
		case <-shutdown:
			// Received an application shutdown. Close the port channel to halt all proxy loops.
			glog.Infof("Shutting down port %s", port)
			close(stopChan)
		case <-stopChan:
		}

		listener.Close()
		glog.Infof("Closed port %s", port)
		return
	}()

	return nil
}

func (sc *ServiceConfig) syncAllPublicPorts(shutdown <-chan interface{}) error {
	rootConn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("syncAllPublicPorts - Error getting root zk connection: %v", err)
		return err
	}

	cancelChan := make(chan interface{})
	syncPorts := func(conn client.Connection, parentPath string, childIDs ...string) {
		allportsLock.Lock()
		defer allportsLock.Unlock()

		glog.V(1).Infof("syncPorts STARTING for parentPath:%s childIDs:%v", parentPath, childIDs)

		// start all servers that have been not started and enabled
		newPorts := make(map[string]chan bool)
		for _, pepID := range childIDs {
			publicEndpointKey := service.PublicEndpointKey(pepID)
			if publicEndpointKey.Type() == registry.EPTypePort && publicEndpointKey.IsEnabled() {
				port := publicEndpointKey.Name()
				stopChan, running := allports[port]

				if !running {
					// recently enabled port - port should be opened
					stopChan = make(chan bool)
					if err := sc.createPublicPortServer(publicEndpointKey, stopChan, shutdown); err != nil {
						continue
					}
				}

				newPorts[port] = stopChan
			}
		}

		// stop all servers that have been deleted or disabled
		for port, stopChan := range allports {
			_, found := newPorts[port]
			if !found {
				glog.V(2).Infof("Stopping port server for port %s", port)
				close(stopChan)
				glog.Infof("Port server shut down for port %s", port)
			}
		}

		allports = newPorts
		glog.V(2).Infof("Portserver allports: %+v", allports)
	}

	for {
		zkServicePEPService := service.ZKServicePublicEndpoints
		glog.V(1).Infof("Running registry.WatchChildren for zookeeper path: %s", zkServicePEPService)
		err := registry.WatchChildren(rootConn, zkServicePEPService, cancelChan, syncPorts, pepWatchError)
		if err != nil {
			glog.V(1).Infof("Will retry in 10 seconds to WatchChildren(%s) due to error: %v", zkServicePEPService, err)
			<-time.After(time.Second * 10)
			continue
		}
		select {
		case <-shutdown:
			close(cancelChan)
			return nil
		default:
		}
	}
}
