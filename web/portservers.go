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
	"mime"
	"net/http"
	"fmt"
	"sync"
	"time"
	"os"
//	"os/exec"

	"github.com/gorilla/mux"

	"github.com/zenoss/glog"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk/service"
)

type PublicEndpointInfo struct {
	hostIP    string
	epPort    uint16
	privateIP string
	serviceID string
}

var (
	allportsLock sync.RWMutex
	allports     map[string] chan bool // map of port number to channel that destroys the server
)

func init() {
	allports = make(map[string] chan bool)
}

func (sc *ServiceConfig) ServePublicPorts(shutdown <-chan (interface{})) {
	go sc.syncAllPublicPorts(shutdown)
}

func (sc *ServiceConfig) CreatePublicPortServer(publicEndpointKey service.PublicEndpointKey, kill <-chan bool, shutdown <-chan (interface{})) {
	print(fmt.Sprintf("***************************************STARTING PUBLIC PORT SERVER ON: %v", publicEndpointKey))
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".woff", "application/font-woff")

	httphandler := func(w http.ResponseWriter, r *http.Request) {
		glog.V(2).Infof("httphandler handling request: %+v", r)
		svcIds := make(map[string]struct{})
		var emptyStruct struct{}
		svcIds[publicEndpointKey.ServiceID()] = emptyStruct
		registryKey := registry.GetPublicEndpointKey(publicEndpointKey.Name(), publicEndpointKey.Type())
		sc.publicendpointhandler(w, r, registryKey, svcIds)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", httphandler)
	r.HandleFunc("/{path:.*}", httphandler)

	go func() {
		port := fmt.Sprintf(":%s", publicEndpointKey.Name())
		err := http.ListenAndServe(port, r)
		if err != nil {
			glog.Errorf("could not setup HTTP webserver on port %s: %s", port, err)
		}

		select {
		case <-kill:
			print(fmt.Sprintf("***************************************KILLING PUBLIC PORT SERVER ON: %v", port))
			os.Exit(0)
		default:
		}
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

		newPorts := make(map[string] chan bool)
		for _, sv := range childIDs {
			publicEndpointKey := service.PublicEndpointKey(sv)
			if publicEndpointKey.Type() == registry.EPTypePort {
				port := publicEndpointKey.Name()
				_, found := newPorts[port]
				if !found {
					newPorts[port] = make(chan bool)

					if publicEndpointKey.IsEnabled() {
						sc.CreatePublicPortServer(publicEndpointKey, newPorts[port], shutdown)
					}
				}
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
