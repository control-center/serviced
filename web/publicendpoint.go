// Copyright 2014 The Serpepiced Authors.
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
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
)

var (
	localpepregistry = pepRegistry{lookup: make(map[string]*pepInfo), pepWatch: make(map[string]chan<- interface{})}
)

type pepInfo struct {
	sync.RWMutex
	endpoints []pepEndpointInfo
	counter   int
}

func newPepInfo() *pepInfo {
	return &pepInfo{endpoints: make([]pepEndpointInfo, 0)}
}

func (pepi *pepInfo) GetNext() (pepEndpointInfo, error) {
	pepi.Lock()
	defer pepi.Unlock()
	if len(pepi.endpoints) == 0 {
		return pepEndpointInfo{}, errors.New("no public endpoint endpoints available")
	}
	pep := pepi.endpoints[pepi.counter%len(pepi.endpoints)]
	pepi.counter++
	return pep, nil
}

type pepEndpointInfo struct {
	hostIP    string
	epPort    uint16
	privateIP string
	serviceID string
}

func createpepEndpointInfo(pep *registry.PublicEndpoint) pepEndpointInfo {
	return pepEndpointInfo{
		hostIP:    pep.HostIP,
		epPort:    pep.ContainerPort,
		privateIP: pep.ContainerIP,
		serviceID: pep.ServiceID,
	}
}

//pepRegistry keeps track of all current known public endpoints and their corresponding target endpoints
type pepRegistry struct {
	sync.RWMutex
	lookup   map[string]*pepInfo           //maps pep key (name-type) to all availabe target endpoints
	pepWatch map[string]chan<- interface{} //watches to ZK public endpoint dir Channel is to cancel watch
}

func (pr *pepRegistry) getWatch(path string) (chan<- interface{}, bool) {
	pr.RLock()
	defer pr.RUnlock()
	channel, found := pr.pepWatch[path]
	return channel, found
}

func (pr *pepRegistry) setWatch(path string, cancel chan<- interface{}) {
	pr.RLock()
	defer pr.RUnlock()
	pr.pepWatch[path] = cancel
}

func (pr *pepRegistry) deleteWatch(path string) {
	pr.Lock()
	defer pr.Unlock()
	delete(pr.pepWatch, path)
}

//get returns a pepInfo, bool is true or false if path is found
func (pr *pepRegistry) get(path string) (*pepInfo, bool) {
	pr.RLock()
	defer pr.RUnlock()
	pepInfo, found := pr.lookup[path]
	if !found {
		glog.V(4).Infof("path %v not found in map %v", path, pr.lookup)
	}
	return pepInfo, found
}

//setPublicEndpointInfo sets/replaces all the endpoints available for a public endpoint
func (pr *pepRegistry) setPublicEndpointInfo(path string, pepInfo *pepInfo) {
	pr.Lock()
	defer pr.Unlock()
	pr.lookup[path] = pepInfo
	glog.Infof("setPublicEndpointInfo adding Public Endpoint %v with backend: %#v", path, pepInfo)
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

func (sc *ServiceConfig) syncPublicEndpoints(shutdown <-chan interface{}) error {
	glog.Info("syncPublicEndpoints starting")

	glog.V(2).Infof("getting pool based connection")
	// public endpoints are at the root level (not pool aware)
	poolBasedConn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("syncPublicEndpoints - Error getting pool based zk connection: %v", err)
		return err
	}

	glog.V(2).Infof("creating zkPepRegistry")
	zkPepRegistry, err := registry.PublicEndpointRegistry(poolBasedConn)
	if err != nil {
		glog.Errorf("syncPublicEndpoints - Error getting public endpoint registry: %v", err)
		return err
	}

	processPublicEndpoints := func(conn client.Connection, parentPath string, childIDs ...string) {
		glog.V(1).Infof("processPublicEndpoints STARTING for parentPath:%s childIDs:%v", parentPath, childIDs)

		currentPEPs := make(map[string]struct{})
		//watch any new public endpoint nodes
		for _, pepID := range childIDs {
			pepPath := fmt.Sprintf("%s/%s", parentPath, pepID)
			currentPEPs[pepPath] = struct{}{}
			if _, found := localpepregistry.getWatch(pepPath); !found {
				glog.Infof("processing public endpoint watch: %s", pepPath)
				cancelChan := make(chan interface{})
				localpepregistry.setWatch(pepPath, cancelChan)
				go func(pepID string) {
					defer localpepregistry.deleteWatch(pepPath)
					glog.Infof("starting public endpoint watch: %s", pepPath)
					var lastChildIDs []string
					processPublicEndpoint := func(conn client.Connection, parentPath string, childIDs ...string) {

						glog.V(1).Infof("watching:%s %+v", parentPath, childIDs)
						if !sort.StringsAreSorted(childIDs) {
							sort.Strings(childIDs)
						}
						if areEqual(lastChildIDs, childIDs) {
							glog.V(1).Infof("not processing children because they are the same as last ones: %v = %v ", lastChildIDs, childIDs)
							return
						}
						glog.V(1).Infof("processing public endpoint parent %v; children %v", parentPath, childIDs)
						pr, err := registry.PublicEndpointRegistry(conn)
						if err != nil {
							glog.Errorf("processPublicEndpoint - Error getting public endpoint registry: %v", err)
							return
						}

						errors := false
						pepEndpoints := newPepInfo()
						for _, child := range childIDs {
							pepEndpoint, err := pr.GetItem(conn, parentPath+"/"+child)
							if err != nil {
								errors = true
								glog.Errorf("processPublicEndpoint - Error getting public endpoint for %v/%v: %v", parentPath, child, err)
								continue
							}
							glog.V(1).Infof("Processing public endpoint %s/%s: %#v", parentPath, child, pepEndpoint)
							pepInfo := createpepEndpointInfo(pepEndpoint)
							pepEndpoints.endpoints = append(pepEndpoints.endpoints, pepInfo)
						}
						localpepregistry.setPublicEndpointInfo(pepID, pepEndpoints)
						if !errors {
							lastChildIDs = childIDs
						}
					}
					// loop if error. If watch is cancelled will not return error. Blocking call
					for {
						err := zkPepRegistry.WatchKey(conn, pepID, cancelChan, processPublicEndpoint, pepWatchError)
						if err == nil {
							glog.Infof("Public Endpoint Registry Watch %s Stopped", pepID)
							return
						}
						glog.Infof("Public Endpoint Registry Watch %s Restarting due to %v", pepID, err)
						time.Sleep(500 * time.Millisecond)
					}

				}(pepID)
			} else {
				glog.V(2).Infof("public endpoint %s already being watched", pepPath)
			}
		}

		//cancel watching any public endpoint nodes that are no longer
		for previousPEP, cancel := range localpepregistry.pepWatch {
			if _, found := currentPEPs[previousPEP]; !found {
				glog.Infof("Cancelling public endpoint watch for %s}", previousPEP)
				delete(localpepregistry.pepWatch, previousPEP)
				cancel <- true
				close(cancel)
			}
		}
	}
	cancelChan := make(chan interface{})
	for {
		glog.Info("Running zkPepRegistry.WatchRegistry")

		watchStopped := make(chan error)

		go func() {
			watchStopped <- zkPepRegistry.WatchRegistry(poolBasedConn, cancelChan, processPublicEndpoints, pepWatchError)
		}()
		select {
		case <-shutdown:
			close(cancelChan)
			for pep, ch := range localpepregistry.pepWatch {
				glog.V(1).Infof("Shutdown closing watch for %v", pep)
				close(ch)
			}
			return nil
		case err := <-watchStopped:
			if err != nil {
				glog.Infof("Public Endpoint Registry Watch Restarting due to %v", err)
				time.Sleep(500 * time.Millisecond)
			}

		}
	}
}

func pepWatchError(path string, err error) {
	glog.Warningf("processing pepWatchError on %s: %v", path, err)
}

// Lookup the appropriate public endpoint and forward the request to it.
// serviceIDs is the list of services on which the public endpoint is enabled
func (sc *ServiceConfig) publicendpointhandler(w http.ResponseWriter, r *http.Request, pepKey registry.PublicEndpointKey, serviceIDs map[string]struct{}) {
	start := time.Now()
	glog.V(1).Infof("publicendpointhandler handling: %+v", r)

	defer func() {
		glog.V(1).Infof("Time to process %s public endpoint request %v: %v", pepKey, r.URL, time.Since(start))
	}()

	pepEP, err := sc.getPublicEndpoint(string(pepKey))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// check that the endpoint's service id is in the list of public endpoints that are enabled.
	// This happens if more than one tenant has the same public endpoint. One tenant is off and the one that is running has
	// has disabled this public endpoint.
	if _, found := serviceIDs[pepEP.serviceID]; !found {
		http.Error(w, fmt.Sprintf("public endpoint %s not available", pepKey), http.StatusNotFound)
		return
	}

	rp := sc.getReverseProxy(pepEP.hostIP, sc.muxPort, pepEP.privateIP, pepEP.epPort, sc.muxTLS && (sc.muxPort > 0))
	glog.V(1).Infof("Time to set up %s public endpoint proxy for %v: %v", pepKey, r.URL, time.Since(start))

	// Set up the X-Forwarded-Proto header so that downstream servers know
	// the request originated as HTTPS.
	if _, ok := r.Header["X-Forwarded-Proto"]; !ok {
		r.Header.Set("X-Forwarded-Proto", "https")
	}

	rp.ServeHTTP(w, r)
	return
}

func (sc *ServiceConfig) getPublicEndpoint(pepKey string) (pepEndpointInfo, error) {
	pepInfo, found := localpepregistry.get(pepKey)
	if !found {
		glog.V(4).Infof("public endpoint not enabled %s: %v", pepKey)
		return pepEndpointInfo{}, fmt.Errorf("service associated with public endpoint %v is not running", pepKey)
	}

	// round robin through available endpoints
	pepEP, err := pepInfo.GetNext()
	if err != nil {
		glog.V(4).Infof("no endpoint found for public endpoint %s: %v", pepKey, err)
		return pepEndpointInfo{}, err
	}

	return pepEP, nil
}

var reverseProxies map[string]*httputil.ReverseProxy
var reverseProxiesLock sync.Mutex

func init() {
	reverseProxies = make(map[string]*httputil.ReverseProxy)
}

func (sc *ServiceConfig) getReverseProxy(hostIP string, muxPort int, privateIP string, privatePort uint16, useTLS bool) *httputil.ReverseProxy {

	var remoteAddr string

	reverseProxiesLock.Lock()
	defer reverseProxiesLock.Unlock()

	_, isLocalContainer := sc.localAddrs[hostIP]
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

	glog.V(1).Infof("publicendpointhandler reverse proxy to: %v", rpurl)

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	transport.Dial = func(network, addr string) (remote net.Conn, err error) {
		return sc.getRemoteConnection(remoteAddr, isLocalContainer, muxPort, privateIP, privatePort, useTLS)
	}
	rp := httputil.NewSingleHostReverseProxy(&rpurl)
	rp.Transport = transport
	rp.FlushInterval = time.Millisecond * 10

	reverseProxies[key] = rp
	return rp

}

func (sc *ServiceConfig) getRemoteConnection(remoteAddr string, isLocalContainer bool, muxPort int, privateIP string, privatePort uint16, useTLS bool) (net.Conn, error) {
	var (
		remote net.Conn
		err    error
	)

	if useTLS && !isLocalContainer { // Only do TLS if connecting to a TCPMux
		config := tls.Config{InsecureSkipVerify: true}
		glog.V(1).Infof("public endpoint about to dial %s", remoteAddr)
		remote, err = tls.Dial("tcp4", remoteAddr, &config)
	} else {
		glog.V(1).Info("public endpoint about to dial %s", remoteAddr)
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
		glog.V(1).Infof("public endpoint muxing to %s", muxAddr)
		io.WriteString(remote, muxAddr)

	}
	return remote, nil
}
