// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package web

import (
	"github.com/gorilla/mux"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"

	"crypto/tls"
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
	registry = vhostRegistry{lookup: make(map[string][]*servicestate.ServiceState, 0)}
)

type vhostRegistry struct {
	sync.RWMutex
	lookup map[string][]*servicestate.ServiceState
}

//get returns a list of ServiceState, bool is true or false if vhost is found
func (vr *vhostRegistry) get(vhost string) ([]*servicestate.ServiceState, bool) {
	vr.RLock()
	defer vr.RUnlock()
	states, found := vr.lookup[vhost]
	return states, found
}

//get returns a list of ServiceState, bool is true or false if vhost is found
func (vr *vhostRegistry) setAll(vhosts map[string][]*servicestate.ServiceState) {
	vr.Lock()
	defer vr.Unlock()
	vr.lookup = make(map[string][]*servicestate.ServiceState, 0)
	for key, states := range vhosts {
		vr.lookup[key] = states
		for _, state := range states {
			glog.V(4).Infof("vhosthandler adding VHost %v with backend: %#v", key, *state)
		}
	}

}

func (sc *ServiceConfig) syncVhosts() {
	sc.vhostFinder()
	for {
		select {
		case <-time.After(30 * time.Second):
			sc.vhostFinder()
		}
	}

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

	vhosts := make(map[string][]*servicestate.ServiceState, 0)

	for _, s := range services {
		var svc service.Service

		if err := client.GetService(s.ServiceID, &svc); err != nil {
			glog.Errorf("Can't get service: %s (%v)", s.Id, err)
		}

		svcstates := []*servicestate.ServiceState{}
		if err := client.GetServiceStates(s.ServiceID, &svcstates); err != nil {
			glog.Warningf("can't retrieve service states for %s (%v)", s.ServiceID, err)
		}

		for _, vhep := range svc.GetServiceVHosts() {
			for _, vh := range vhep.VHosts {
				for _, ss := range svcstates {
					vhosts[vh] = append(vhosts[vh], ss)
				}
			}
		}
	}
	registry.setAll(vhosts)
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

	defer func() { glog.V(1).Infof("Time to process %s vhost request %v: %v", subdomain, r.URL, time.Since(start)) }()

	var svcstates []*servicestate.ServiceState
	found := false
	tries := 2
	for !found && tries > 0 {
		svcstates, found = registry.get(subdomain)
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
	for _, svcep := range svcstates[0].Endpoints {
		for _, vh := range svcep.VHosts {
			if vh == subdomain {
				remoteAddr := fmt.Sprintf("%s:%d", svcstates[0].HostIP, svcep.PortNumber)
				if sc.muxTLS && (sc.muxPort > 0) { // Only do TLS if connecting to a TCPMux
					remoteAddr = fmt.Sprintf("%s:%d", svcstates[0].HostIP, sc.muxPort)
				}
				rp := getReverseProxy(remoteAddr, sc.muxPort, svcstates[0].PrivateIP, svcep.PortNumber, sc.muxTLS && (sc.muxPort > 0))
				glog.V(1).Infof("Time to set up %s vhost proxy for %v: %v", subdomain, r.URL, time.Since(start))
				rp.ServeHTTP(w, r)
				return
			}
		}
	}
	http.Error(w, fmt.Sprintf("unrecognized endpoint: %s", subdomain), http.StatusNotImplemented)
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
