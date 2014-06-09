// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package web

import (
	"github.com/gorilla/mux"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	//	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/zzk/registry"

	"crypto/tls"
	"fmt"
	"github.com/zenoss/serviced/coordinator/client"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

var (
	vregistry  = vhostRegistry{lookup: make(map[string]*vhostInfo)}
	vregistry2 = vhostRegistry{lookup: make(map[string]*vhostInfo)}
)

type vhostInfo struct {
	endpoints []vhostEndPoint
	counter   int
}

func newVhostInfo() *vhostInfo {
	return &vhostInfo{endpoints: make([]vhostEndPoint, 0)}
}

func (vi *vhostInfo) GetNext() (vhostEndPoint, error) {
	vep := vi.endpoints[vi.counter%len(vi.endpoints)]
	return vep, nil
}

type vhostEndPoint struct {
	hostIP    string
	epPort    uint16
	privateIP string
}

func createVhostInfos(state *servicestate.ServiceState) map[string]*vhostInfo {
	infos := make(map[string]*vhostInfo)

	for _, svcep := range state.Endpoints {
		for _, vhost := range svcep.VHosts {
			if _, found := infos[vhost]; !found {
				infos[vhost] = newVhostInfo()
			}
			vi := vhostEndPoint{
				hostIP:    state.HostIP,
				epPort:    svcep.PortNumber,
				privateIP: state.PrivateIP,
			}
			info := infos[vhost]
			info.endpoints = append(infos[vhost].endpoints, vi)
		}
	}
	return infos
}

type vhostRegistry struct {
	sync.RWMutex
	lookup map[string]*vhostInfo
}

//get returns a vhostInfo, bool is true or false if vhost is found
func (vr *vhostRegistry) get(vhost string) (*vhostInfo, bool) {
	vr.RLock()
	defer vr.RUnlock()
	vhInfo, found := vr.lookup[vhost]
	return vhInfo, found
}

//get returns a list of ServiceState, bool is true or false if vhost is found
func (vr *vhostRegistry) setAll(vhosts map[string]*vhostInfo) {
	vr.Lock()
	defer vr.Unlock()
	vr.lookup = make(map[string]*vhostInfo)
	for key, infos := range vhosts {
		vr.lookup[key] = infos
		for _, ep := range infos.endpoints {
			glog.V(4).Infof("vhosthandler adding VHost %v with backend: %#v", key, ep)
		}
	}

}

func (sc *ServiceConfig) syncVhosts() {
	go sc.watchVhosts()
	sc.vhostFinder()
	for {
		select {
		case <-time.After(30 * time.Second):
			sc.vhostFinder()
		}
	}

}

func (sc *ServiceConfig) watchVhosts() error {
	glog.Info("watchVhosts starting:")
	conn, err := sc.zkClient.GetConnection()
	if err != nil {
		glog.Errorf("watchVhosts - Error getting zk connection: %v", err)
		return err
	}

	vhostRegistry, err := registry.VHostRegistry(conn)
	if err != nil {
		glog.Errorf("watchVhosts - Error getting vhost registry: %v", err)
		return err
	}

	processVhosts := func(conn client.Connection, parentPath string, childIDs ...string) {
		glog.Info("processVhosts STARTING")

		for _, id := range childIDs {
			glog.Infof("processing vhost watch: %s/%s", parentPath, id)
			vhostRegistry.WatchKey(conn, id, processVhost, vhostWatchError)
		}
	}

	vhostRegistry.WatchRegistry(conn, processVhosts, vhostWatchError)

	return nil
}

//processVhost is used to watch the children of particular vhost in the registry
func processVhost(conn client.Connection, parentPath string, childIDs ...string) {
	glog.Infof("processing vhost parent %v; children %v", parentPath, childIDs)
	vr, err := registry.VHostRegistry(conn)
	if err != nil {
		glog.Errorf("processVhost - Error getting vhost registry: %v", err)
		return
	}
	for _, child := range childIDs {
		node, err := vr.GetItem(conn, parentPath+"/"+child)
		if err != nil {
			glog.Errorf("processVhost - Error getting vhost for %v/%v: %v", parentPath, child, err)
			continue
		}
		glog.Infof("Processing vhost %s/%s: %#v", parentPath, child, node)
	}
}
func vhostWatchError(path string, err error) {
	glog.Infof("processing vhostWatchError on %s: %v", path, err)

}

func (sc *ServiceConfig) vhostFinder() error {
	glog.V(4).Infof("vhost syncing...")
	client, err := sc.getClient()
	if err != nil {
		glog.Warningf("error getting client could not lookup vhosts: %v", err)
		return err
	}

	services := []*dao.RunningService{}
	client.GetRunningServices(&empty, &services)

	vhosts := make(map[string]*vhostInfo, 0)

	for _, s := range services {
		//		var svc service.Service
		//
		//		if err := client.GetService(s.ServiceID, &svc); err != nil {
		//			glog.Errorf("Can't get service: %s (%v)", s.Id, err)
		//		}

		svcstates := []*servicestate.ServiceState{}
		if err := client.GetServiceStates(s.ServiceID, &svcstates); err != nil {
			glog.Warningf("can't retrieve service states for %s (%v)", s.ServiceID, err)
		}

		for _, state := range svcstates {
			vhostMap := createVhostInfos(state)
			for vhost, info := range vhostMap {
				if _, found := vhosts[vhost]; !found {
					vhosts[vhost] = newVhostInfo()
				}
				vhInfo := vhosts[vhost]
				vhInfo.endpoints = append(vhInfo.endpoints, info.endpoints...)

			}
		}
	}
	//
	//		for _, vhep := range svc.GetServiceVHosts() {
	//			for _, vh := range vhep.VHosts {
	//				for _, ss := range svcstates {
	//					vhosts[vh] = append(vhosts[vh], ss)
	//				}
	//			}
	//		}
	//	}
	vregistry.setAll(vhosts)
	return nil
}

// Lookup the appropriate virtual host and forward the request to it.
// TODO: when zookeeper registration is integrated we can be more event
// driven and only refresh the vhost map when service states change.
func (sc *ServiceConfig) vhosthandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	glog.V(1).Infof("vhosthandler handling: %v", r)
	muxvars := mux.Vars(r)
	subdomain := muxvars["subdomain"]

	defer func() {
		glog.V(1).Infof("Time to process %s vhost request %v: %v", subdomain, r.URL, time.Since(start))
	}()

	var vhInfo *vhostInfo
	found := false
	tries := 2
	for !found && tries > 0 {
		vhInfo, found = vregistry.get(subdomain)
		tries--
		if !found && tries > 0 {
			glog.Infof("vhost %s not found, syncing...", subdomain)
			sc.vhostFinder()
		}
	}
	if !found {
		http.Error(w, fmt.Sprintf("service associated with vhost %v is not running", subdomain), http.StatusNotFound)
		return
	}
	// TODO: implement a more intelligent strategy than "always pick the first one" when more
	// than one service state is mapped to a given virtual host
	vhEP, err := vhInfo.GetNext()
	if err != nil {
		http.Error(w, fmt.Sprintf("no available service for vhost %v ", subdomain), http.StatusNotFound)
		return
	}
	remoteAddr := fmt.Sprintf("%s:%d", vhEP.hostIP, vhEP.epPort)
	if sc.muxTLS && (sc.muxPort > 0) { // Only do TLS if connecting to a TCPMux
		remoteAddr = fmt.Sprintf("%s:%d", vhEP.hostIP, sc.muxPort)
	}
	rp := getReverseProxy(remoteAddr, sc.muxPort, vhEP.privateIP, vhEP.epPort, sc.muxTLS && (sc.muxPort > 0))
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
