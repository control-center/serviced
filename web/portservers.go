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
	allports     map[string]chan int // map of port number to channel that destroys the server
	cpDao        dao.ControlPlane
)

func init() {
	allports = make(map[string]chan int)
}

func disablePort(publicEndpointKey service.PublicEndpointKey) {
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
	cpDao.UpdateService(myService, &unused)
}

func (sc *ServiceConfig) ServePublicPorts(shutdown <-chan (interface{}), dao dao.ControlPlane) {
	cpDao = dao
	go sc.syncAllPublicPorts(shutdown)
}

func (sc *ServiceConfig) CreatePublicPortServer(publicEndpointKey service.PublicEndpointKey, stopChan <-chan int, shutdown <-chan (interface{})) error {
	port := publicEndpointKey.Name()
	listener, err := net.Listen("tcp", port)
	stopChans := []chan bool{}
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
				glog.Errorf("%s", err)
			}

			// setup remote connection
			remotePort := fmt.Sprintf("%s:%d", pepEPInfo.privateIP, pepEPInfo.epPort)
			remoteAddr, err := net.ResolveTCPAddr("tcp", remotePort)
			if err != nil {
				glog.Errorf("Cannot resolve remote address - %s: %s", remotePort, err)
				continue
			} else {
				glog.Infof("Resolved remote address - %s", remotePort)
			}

			remoteConn, err := net.DialTCP("tcp", nil, remoteAddr)
			if err != nil {
				glog.Errorf("%s", err)
				continue
			}

			connStopChan := make(chan bool)
			stopChans = append(stopChans, connStopChan)
			if err != nil {
				for _, c := range stopChans {
					c <- true
				}
				disablePort(publicEndpointKey)
				listener.Close()
			}

			// serve proxied requests/responses
			go proxy.ProxyLoop(localConn, remoteConn, connStopChan)
		}
	}()

	go func() {
		// Wait for shutdown, then kill all your connections
		<-stopChan
		for _, c := range stopChans {
			c <- true
		}

		// disablePort modifies allports, so we need a lock
		allportsLock.Lock()
		defer allportsLock.Unlock()
		disablePort(publicEndpointKey)
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
		newPorts := make(map[string]chan int)
		for _, pepID := range childIDs {
			publicEndpointKey := service.PublicEndpointKey(pepID)
			if publicEndpointKey.Type() == registry.EPTypePort && publicEndpointKey.IsEnabled() {
				port := publicEndpointKey.Name()
				stopChan, running := allports[port]

				if !running {
					// recently enabled port - port should be opened
					stopChan = make(chan int)
					if err := sc.CreatePublicPortServer(publicEndpointKey, stopChan, shutdown); err != nil {
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
				stopChan <- 0
			}
		}

		allports = newPorts
		glog.V(1).Infof("allports: %+v", allports)
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
