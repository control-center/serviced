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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/zenoss/glog"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
)

var (
	vregistry = vhostRegistry{lookup: make(map[string]*vhostInfo), vhostWatch: make(map[string]chan<- bool)}
)

type vhostInfo struct {
	sync.RWMutex
	endpoints []vhostEndpointInfo
	counter   int
}

func newVhostInfo() *vhostInfo {
	return &vhostInfo{endpoints: make([]vhostEndpointInfo, 0)}
}

func (vi *vhostInfo) GetNext() (vhostEndpointInfo, error) {
	vi.Lock()
	defer vi.Unlock()
	if len(vi.endpoints) == 0 {
		return vhostEndpointInfo{}, errors.New("no vhost endpoints available")
	}
	vep := vi.endpoints[vi.counter%len(vi.endpoints)]
	vi.counter++
	return vep, nil
}

type vhostEndpointInfo struct {
	hostIP    string
	epPort    uint16
	privateIP string
	serviceID string
}

func createvhostEndpointInfo(vep *registry.VhostEndpoint) vhostEndpointInfo {
	return vhostEndpointInfo{
		hostIP:    vep.HostIP,
		epPort:    vep.ContainerPort,
		privateIP: vep.ContainerIP,
		serviceID: vep.ServiceID,
	}
}

//vhostRegistry keeps track of all current known vhosts and vhost endpoints.
type vhostRegistry struct {
	sync.RWMutex
	lookup     map[string]*vhostInfo  //vhost name to all availabe endpoints
	vhostWatch map[string]chan<- bool //watches to ZK vhost dir  e.g. zenoss5x. Channel is to cancel watch
}

//get returns a vhostInfo, bool is true or false if vhost is found
func (vr *vhostRegistry) get(vhost string) (*vhostInfo, bool) {
	vr.RLock()
	defer vr.RUnlock()
	vhInfo, found := vr.lookup[vhost]
	if !found {
		glog.V(4).Infof("vhost %v not found in map %v", vhost, vr.lookup)
	}
	return vhInfo, found
}

//setEndpoints sets/replaces all the endpoints available for a vhost
func (vr *vhostRegistry) setVhostInfo(vhost string, vhInfo *vhostInfo) {
	vr.Lock()
	defer vr.Unlock()
	vr.lookup[vhost] = vhInfo
	glog.Infof("setVhostInfo adding VHost %v with backend: %#v", vhost, vhInfo)
}

func areEqual(s1, s2 []string) bool {

	if s1 == nil || s2 == nil {
		return false
	}
	if len(s1) != len(s2) {
		return false
	}
	for i, v := range s1 {
		if v != s2[i] {
			return false
		}
	}
	return true
}

func (sc *ServiceConfig) syncVhosts(shutdown <-chan interface{}) error {
	glog.Info("watchVhosts starting")

	glog.V(2).Infof("getting pool based connection")
	// vhosts are at the root level (not pool aware)
	poolBasedConn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("watchVhosts - Error getting pool based zk connection: %v", err)
		return err
	}

	glog.V(2).Infof("creating vhostRegistry")
	vhostRegistry, err := registry.VHostRegistry(poolBasedConn)
	if err != nil {
		glog.Errorf("watchVhosts - Error getting vhost registry: %v", err)
		return err
	}

	processVhosts := func(conn client.Connection, parentPath string, childIDs ...string) {
		glog.V(1).Infof("processVhosts STARTING for parentPath:%s childIDs:%v", parentPath, childIDs)

		currentVhosts := make(map[string]struct{})
		//watch any new vhost nodes
		for _, vhostID := range childIDs {
			vhostPath := fmt.Sprintf("%s/%s", parentPath, vhostID)
			currentVhosts[vhostPath] = struct{}{}
			if _, found := vregistry.vhostWatch[vhostPath]; !found {
				glog.Infof("processing vhost watch: %s", vhostPath)
				cancelChan := make(chan bool)
				vregistry.vhostWatch[vhostPath] = cancelChan
				go func(vhostID string) {
					defer delete(vregistry.vhostWatch, vhostPath)
					glog.Infof("starting vhost watch: %s", vhostPath)
					var lastChildIDs []string
					processVhost := func(conn client.Connection, parentPath string, childIDs ...string) {

						glog.V(1).Infof("watching:%s %+v", parentPath, childIDs)
						if !sort.StringsAreSorted(childIDs) {
							sort.Strings(childIDs)
						}
						if areEqual(lastChildIDs, childIDs) {
							glog.V(1).Infof("not processing children because they are the same as last ones: %v = %v ", lastChildIDs, childIDs)
							return
						}
						glog.V(1).Infof("processing vhost parent %v; children %v", parentPath, childIDs)
						vr, err := registry.VHostRegistry(conn)
						if err != nil {
							glog.Errorf("processVhost - Error getting vhost registry: %v", err)
							return
						}

						errors := false
						vhostEndpoints := newVhostInfo()
						for _, child := range childIDs {
							vhEndpoint, err := vr.GetItem(conn, parentPath+"/"+child)
							if err != nil {
								errors = true
								glog.Errorf("processVhost - Error getting vhost for %v/%v: %v", parentPath, child, err)
								continue
							}
							glog.V(1).Infof("Processing vhost %s/%s: %#v", parentPath, child, vhEndpoint)
							vepInfo := createvhostEndpointInfo(vhEndpoint)
							vhostEndpoints.endpoints = append(vhostEndpoints.endpoints, vepInfo)
						}
						vregistry.setVhostInfo(vhostID, vhostEndpoints)
						if !errors {
							lastChildIDs = childIDs
						}
					}
					// loop if error. If watch is cancelled will not return error. Blocking call
					for {
						err := vhostRegistry.WatchKey(conn, vhostID, cancelChan, processVhost, vhostWatchError)
						if err == nil {
							glog.Infof("VHostRegisty Watch %s Stopped", vhostID)
							return
						}
						glog.Infof("VHostRegisty Watch %s Restarting due to %v", vhostID, err)
						time.Sleep(500 * time.Millisecond)
					}

				}(vhostID)
			} else {
				glog.V(2).Infof("vhost %s already being watched", vhostPath)
			}
		}

		//cancel watching any vhosts nodes that are no longer
		for previousVhost, cancel := range vregistry.vhostWatch {
			if _, found := currentVhosts[previousVhost]; !found {
				glog.Infof("Cancelling vhost watch for %s}", previousVhost)
				delete(vregistry.vhostWatch, previousVhost)
				cancel <- true
				close(cancel)
			}
		}
	}
	cancelChan := make(chan bool)
	for {
		glog.Info("Running vhostRegistry.WatchRegistry")

		watchStopped := make(chan error)

		go func() {
			watchStopped <- vhostRegistry.WatchRegistry(poolBasedConn, cancelChan, processVhosts, vhostWatchError)
		}()
		select {
		case <-shutdown:
			close(cancelChan)
			for vhost, ch := range vregistry.vhostWatch {
				glog.V(1).Infof("Shutdown closing watch for %v", vhost)
				close(ch)
			}
			return nil
		case err := <-watchStopped:
			if err != nil {
				glog.Infof("VHostRegisty Watch Restarting due to %v", err)
				time.Sleep(500 * time.Millisecond)
			}

		}
	}
}

func vhostWatchError(path string, err error) {
	glog.Warningf("processing vhostWatchError on %s: %v", path, err)
}

// Lookup the appropriate virtual host and forward the request to it.
// serviceIDs is the list of services on which the vhost is enabled
func (sc *ServiceConfig) vhosthandler(w http.ResponseWriter, r *http.Request, vhostname string, serviceIDs map[string]struct{}) {
	start := time.Now()
	glog.V(1).Infof("vhosthandler handling: %+v", r)

	defer func() {
		glog.V(1).Infof("Time to process %s vhost request %v: %v", vhostname, r.URL, time.Since(start))
	}()

	vhInfo, found := vregistry.get(vhostname)
	if !found {
		http.Error(w, fmt.Sprintf("service associated with vhost %v is not running", vhostname), http.StatusNotFound)
		return
	}
	// round robin through available endpoints
	vhEP, err := vhInfo.GetNext()
	if err != nil {
		glog.V(4).Infof("no endpoint found for vhost %s: %v", vhostname, err)
		http.Error(w, fmt.Sprintf("no available service for vhost %v ", vhostname), http.StatusNotFound)
		return
	}
	// check that the endpoint's service id is in the list of vhosts that are enabled.
	// This happens if more than one tenant has the same vhost. One tenant is off and the one that is running has
	// has disabled this vhost.
	if _, found := serviceIDs[vhEP.serviceID]; !found {
		glog.V(4).Infof("vhost not enabled %s: %v", vhostname, err)
		http.Error(w, fmt.Sprintf("vhost %s not available", vhostname), http.StatusNotFound)
		return
	}

	rp := getReverseProxy(vhEP.hostIP, sc.muxPort, vhEP.privateIP, vhEP.epPort, sc.muxTLS && (sc.muxPort > 0))
	glog.V(1).Infof("Time to set up %s vhost proxy for %v: %v", vhostname, r.URL, time.Since(start))

	// Set up the X-Forwarded-Proto header so that downstream servers know
	// the request originated as HTTPS.
	if _, ok := r.Header["X-Forwarded-Proto"]; !ok {
		r.Header.Set("X-Forwarded-Proto", "https")
	}

	rp.ServeHTTP(w, r)
	return
}

var reverseProxies map[string]*httputil.ReverseProxy
var reverseProxiesLock sync.Mutex
var localAddrs map[string]struct{}

func init() {
	var err error
	reverseProxies = make(map[string]*httputil.ReverseProxy)
	hostAddrs, err := utils.GetIPv4Addresses()
	if err != nil {
		panic(err)
	}
	localAddrs = make(map[string]struct{})
	for _, host := range hostAddrs {
		localAddrs[host] = struct{}{}
	}
}

func getReverseProxy(hostIP string, muxPort int, privateIP string, privatePort uint16, useTLS bool) *httputil.ReverseProxy {

	var remoteAddr string

	reverseProxiesLock.Lock()
	defer reverseProxiesLock.Unlock()

	_, isLocalContainer := localAddrs[hostIP]
	if isLocalContainer {
		remoteAddr = fmt.Sprintf("%s:%d", privateIP, privatePort)
	} else {
		remoteAddr = fmt.Sprintf("%s:%d", hostIP, muxPort)
	}

	key := fmt.Sprintf("%s,%d,%s,%d,%v", remoteAddr, muxPort, privateIP, privatePort, useTLS)
	proxy, ok := reverseProxies[key]
	if ok {
		return proxy
	}

	rpurl := url.URL{Scheme: "http", Host: remoteAddr}

	glog.V(1).Infof("vhosthandler reverse proxy to: %v", rpurl)

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	transport.Dial = func(network, addr string) (remote net.Conn, err error) {
		if useTLS && !isLocalContainer { // Only do TLS if connecting to a TCPMux
			config := tls.Config{InsecureSkipVerify: true}
			glog.V(1).Infof("vhost about to dial %s", remoteAddr)
			remote, err = tls.Dial("tcp4", remoteAddr, &config)
		} else {
			glog.V(1).Info("vhost about to dial %s", remoteAddr)
			remote, err = net.Dial("tcp4", remoteAddr)
		}
		if err != nil {
			return nil, err
		}

		if muxPort > 0 && !isLocalContainer {
			//TODO: move this check to happen sooner
			if len(privateIP) == 0 {
				return nil, fmt.Errorf("missing endpoint")
			}
			muxAddr := fmt.Sprintf("%s:%d\n", privateIP, privatePort)
			glog.V(1).Infof("vhost muxing to %s", muxAddr)
			io.WriteString(remote, muxAddr)

		}
		return remote, nil
	}
	rp := httputil.NewSingleHostReverseProxy(&rpurl)
	rp.Transport = transport
	rp.FlushInterval = time.Millisecond * 10

	reverseProxies[key] = rp
	return rp

}
