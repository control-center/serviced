// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package web

import (
	"github.com/gorilla/mux"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/zzk"
	"github.com/zenoss/serviced/zzk/registry"

	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
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
}

func createvhostEndpointInfo(vep *registry.VhostEndpoint) vhostEndpointInfo {
	return vhostEndpointInfo{
		hostIP:    vep.HostIP,
		epPort:    vep.ContainerPort,
		privateIP: vep.ContainerIP,
	}
}

func createVhostInfos(state *servicestate.ServiceState) map[string]*vhostInfo {
	infos := make(map[string]*vhostInfo)

	for _, svcep := range state.Endpoints {
		for _, vhost := range svcep.VHosts {
			if _, found := infos[vhost]; !found {
				infos[vhost] = newVhostInfo()
			}
			vi := vhostEndpointInfo{
				hostIP:    state.HostIP,
				epPort:    svcep.PortNumber,
				privateIP: state.PrivateIP,
			}
			info := infos[vhost]
			info.endpoints = append(infos[vhost].endpoints, vi)
		}
	}
	glog.Infof("created vhost infos %#v", infos)
	return infos
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

//replaces all the vhost lookup information
func (vr *vhostRegistry) setAll(vhosts map[string]*vhostInfo) {
	vr.Lock()
	defer vr.Unlock()
	vr.lookup = make(map[string]*vhostInfo)
	for key, infos := range vhosts {
		vr.lookup[key] = infos
		for _, ep := range infos.endpoints {
			glog.Infof("vhosthandler adding VHost %v with backend: %#v", key, ep)
		}
	}
}

func (sc *ServiceConfig) syncVhosts() {
	go sc.watchVhosts()
}

func (sc *ServiceConfig) getProcessVhosts(vhostRegistry *registry.VhostRegistry) registry.ProcessChildrenFunc {
	return func(conn client.Connection, parentPath string, childIDs ...string) {
		glog.Infof("processVhosts STARTING for parentPath:%s childIDs:%v", parentPath, childIDs)

		currentVhosts := make(map[string]struct{})
		//watch any new vhost nodes
		for _, vhostID := range childIDs {
			vhostPath := fmt.Sprintf("%s/%s", parentPath, vhostID)
			currentVhosts[vhostPath] = struct{}{}
			if _, found := vregistry.vhostWatch[vhostPath]; !found {
				glog.Infof("processing vhost watch: %s", vhostPath)
				cancelChan := make(chan bool)
				vregistry.vhostWatch[vhostPath] = cancelChan
				go vhostRegistry.WatchKey(conn, vhostID, cancelChan, sc.processVhost(vhostID), vhostWatchError)
			} else {
				glog.Infof("vhost %s already being watched", vhostPath)
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
}

func (sc *ServiceConfig) watchVhosts() error {
	glog.Info("watchVhosts starting")

	/*mc, err := sc.getMasterClient()
	if err != nil {
		glog.Errorf("watchVhosts - Error getting master client: %v", err)
		return err
	}

	allPools, err := mc.GetResourcePools()
	if err != nil {
		glog.Errorf("watchVhosts - Error getting resource pools: %v", err)
		return err
	}*/
	// CLARK TODO
	//for _, aPool := range allPools {
	poolBasedConn, err := zzk.GetPoolBasedConnection("")
	if err != nil {
		glog.Errorf("watchVhosts - Error getting pool based zk connection: %v", err)
		return err
	}

	vhostRegistry, err := registry.VHostRegistry(poolBasedConn)
	if err != nil {
		glog.Errorf("watchVhosts - Error getting vhost registry: %v", err)
		return err
	}

	cancelChan := make(chan bool)
	go func() {
		vhostRegistry.WatchRegistry(poolBasedConn, cancelChan, sc.getProcessVhosts(vhostRegistry), vhostWatchError)
		glog.Warning("watchVhosts ended")
	}()
	//}

	return nil
}

//processVhost is used to watch the children of particular vhost in the registry
func (sc *ServiceConfig) processVhost(vhostID string) registry.ProcessChildrenFunc {

	return func(conn client.Connection, parentPath string, childIDs ...string) {
		glog.Infof("processing vhost parent %v; children %v", parentPath, childIDs)
		vr, err := registry.VHostRegistry(conn)
		if err != nil {
			glog.Errorf("processVhost - Error getting vhost registry: %v", err)
			return
		}

		vhostEndpoints := newVhostInfo()
		for _, child := range childIDs {
			vhEndpoint, err := vr.GetItem(conn, parentPath+"/"+child)
			if err != nil {
				glog.Errorf("processVhost - Error getting vhost for %v/%v: %v", parentPath, child, err)
				continue
			}
			glog.Infof("Processing vhost %s/%s: %#v", parentPath, child, vhEndpoint)
			vepInfo := createvhostEndpointInfo(vhEndpoint)
			vhostEndpoints.endpoints = append(vhostEndpoints.endpoints, vepInfo)
		}
		vregistry.setVhostInfo(vhostID, vhostEndpoints)
	}
}

func vhostWatchError(path string, err error) {
	glog.Warningf("processing vhostWatchError on %s: %v", path, err)
}

// Lookup the appropriate virtual host and forward the request to it.
// TODO: when zookeeper registration is integrated we can be more event
// driven and only refresh the vhost map when service states change.
func (sc *ServiceConfig) vhosthandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	glog.V(1).Infof("vhosthandler handling: %+v", r)
	muxvars := mux.Vars(r)
	subdomain := muxvars["subdomain"]
	glog.V(1).Infof("muxvars: %+v", muxvars)

	defer func() {
		glog.V(1).Infof("Time to process %s vhost request %v: %v", subdomain, r.URL, time.Since(start))
	}()

	vhInfo, found := vregistry.get(subdomain)
	if !found {
		http.Error(w, fmt.Sprintf("service associated with vhost %v is not running", subdomain), http.StatusNotFound)
		return
	}
	// TODO: implement a more intelligent strategy than "always pick the first one" when more
	// than one service state is mapped to a given virtual host
	vhEP, err := vhInfo.GetNext()
	if err != nil {
		glog.V(4).Infof("no endpoing found for vhost %s: %v", subdomain, err)
		http.Error(w, fmt.Sprintf("no available service for vhost %v ", subdomain), http.StatusNotFound)
		return
	}
	remoteAddr := fmt.Sprintf("%s:%d", vhEP.hostIP, vhEP.epPort)
	if sc.muxTLS && (sc.muxPort > 0) { // Only do TLS if connecting to a TCPMux
		remoteAddr = fmt.Sprintf("%s:%d", vhEP.hostIP, sc.muxPort)
	}
	rp := getReverseProxy(remoteAddr, sc.muxPort, vhEP.privateIP, vhEP.epPort, sc.muxTLS && (sc.muxPort > 0))
	glog.V(1).Infof("vhost proxy remoteAddr:%s sc.muxPort:%s vhEP.privateIP:%s vhEP.epPort:%s", remoteAddr, sc.muxPort, vhEP.privateIP, vhEP.epPort)
	glog.V(1).Infof("Time to set up %s vhost proxy for %v: %v", subdomain, r.URL, time.Since(start))
	rp.ServeHTTP(w, r)
	return
}

var reverseProxies map[string]*httputil.ReverseProxy
var reverseProxiesLock sync.Mutex

func init() {
	reverseProxies = make(map[string]*httputil.ReverseProxy)
}

func getReverseProxy(remoteAddr string, muxPort int, privateIP string, privatePort uint16, useTLS bool) *httputil.ReverseProxy {

	reverseProxiesLock.Lock()
	defer reverseProxiesLock.Unlock()

	key := fmt.Sprintf("%s,%d,%s,%s,%v", remoteAddr, muxPort, privateIP, privatePort, useTLS)
	proxy, ok := reverseProxies[key]
	if ok {
		return proxy
	}

	rpurl := url.URL{Scheme: "http", Host: remoteAddr}

	glog.V(1).Infof("vhosthandler reverse proxy to: %v", rpurl)

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	transport.Dial = func(network, addr string) (remote net.Conn, err error) {
		if useTLS { // Only do TLS if connecting to a TCPMux
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

		if muxPort > 0 {
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
	transport.DisableCompression = true
	transport.DisableKeepAlives = true
	rp := httputil.NewSingleHostReverseProxy(&rpurl)
	rp.Transport = transport
	rp.FlushInterval = time.Second

	reverseProxies[key] = rp
	return rp

}
