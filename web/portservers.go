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
	"net"
	"sync"
	"time"
	"fmt"

	"github.com/control-center/serviced/proxy"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/dao"
	domainService "github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

var (
	allportsLock sync.RWMutex
	allports     map[string]chan int // map of port number to channel that destroys the server
	cpDao	dao.ControlPlane
)

func init() {
	allports = make(map[string]chan int)
}

func disablePort(publicEndpointKey service.PublicEndpointKey){
	// remove the port from our local cache
	delete(allports, publicEndpointKey.Name())

	// find the endpoint that matches this port number for this service (there will only be 1)
	var myService domainService.Service
	var myEndpoint domainService.ServiceEndpoint
	var unused int
	cpDao.GetService(publicEndpointKey.ServiceID(), &myService)
	for _, endpoint := range(myService.Endpoints) {
		for _, endpointPort := range(endpoint.PortList) {
			if endpointPort.PortAddr == publicEndpointKey.Name(){
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

func (sc *ServiceConfig) CreatePublicPortServer(publicEndpointKey service.PublicEndpointKey, stopChan <-chan int, shutdown <-chan (interface{})) {
	port := publicEndpointKey.Name()
	listener, err := net.Listen("tcp", port)
	stopChans := []chan bool{}
	if err != nil {
		glog.Errorf("Could not setup TCP listener - %s", err)
		disablePort(publicEndpointKey)
		return
	}
	glog.Infof("Listening on port %s", port)

	go func() {
		for {
			pepEPInfo, err := sc.getPublicEndpoint(string(publicEndpointKey))
			if err != nil {
				glog.Errorf(fmt.Sprintf("%s", err))
				return
			}
			remotePort := fmt.Sprintf("%s:%d", pepEPInfo.privateIP, pepEPInfo.epPort)
			remoteAddr, err := net.ResolveTCPAddr("tcp", remotePort)
			if err != nil {
				glog.Errorf("Cannot resolve remote address - %s: %s", remotePort, err)
				return
			}else{
				glog.Infof("Resolved remote address - %s", remotePort)
			}

			localConn, err := listener.Accept()
			remoteConn, err := net.DialTCP("tcp", nil, remoteAddr)

			connStopChan := make(chan bool)
			stopChans = append(stopChans, connStopChan)
			if err != nil {
				for _, c := range stopChans {
					c <- true
				}
				disablePort(publicEndpointKey)
				listener.Close()
			}

			go proxy.ProxyLoop(localConn, remoteConn, connStopChan)
		}
	}()

	go func() {
		// Wait for shutdown, then kill all your connections
		<-stopChan
		for _, c := range stopChans {
			c <- true
		}
		listener.Close()
		glog.Infof("Closed port %s", port)
		return
	}()
}

func (sc *ServiceConfig) syncAllPublicPorts(shutdown <-chan interface{}) error {
	rootConn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("syncAllPublicPorts - Error getting root zk connection: %v", err)
		return err
	}

	cancelChan := make(chan interface{})
	syncPorts := func(conn client.Connection, parentPath string, childIDs ...string) {
		glog.V(1).Infof("syncPorts STARTING for parentPath:%s childIDs:%v", parentPath, childIDs)

		// start all servers that have been not started and enabled
		newPorts := make(map[string]chan int)
		for _, sv := range childIDs {
			publicEndpointKey := service.PublicEndpointKey(sv)
			if publicEndpointKey.Type() == registry.EPTypePort && publicEndpointKey.IsEnabled() {
				port := publicEndpointKey.Name()
				stopChan, running := allports[port]

				if !running {
					// recently enabled port - port should be opened
					stopChan = make(chan int)
					sc.CreatePublicPortServer(publicEndpointKey, stopChan, shutdown)
				}

				newPorts[port] = stopChan
			}
		}

		// stop all servers that have been deleted or disabled
		for port, stopChan := range allports {
			_, found := newPorts[port]
			if !found {
				stopChan <- 0
				close(stopChan)
			}
		}

		//lock for as short a time as possible
		allportsLock.Lock()
		defer allportsLock.Unlock()
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
