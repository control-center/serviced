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
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/control-center/serviced/zzk/service"
	"github.com/gorilla/mux"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/control-center/serviced/facade"
)

// UIConfig contains configuration values
// that the UI may care about
type UIConfig struct {
	PollFrequency int
}

// ServiceConfig is the ui/rest handler for control center
type ServiceConfig struct {
	bindPort    string
	agentPort   string
	stats       bool
	hostaliases []string
	muxTLS      bool
	muxPort     int
	certPEMFile string
	keyPEMFile  string
	localAddrs  map[string]struct{}
	uiConfig    UIConfig
	facade      facade.FacadeInterface
}

var defaultHostAlias string
var uiConfig UIConfig

// NewServiceConfig creates a new ServiceConfig
func NewServiceConfig(bindPort string, agentPort string, stats bool, hostaliases []string, muxTLS bool, muxPort int,
	aGroup string, certPEMFile string, keyPEMFile string, pollFrequency int, facade facade.FacadeInterface) *ServiceConfig {

	uiCfg := UIConfig{
		PollFrequency: pollFrequency,
	}

	cfg := ServiceConfig{
		bindPort:    bindPort,
		agentPort:   agentPort,
		stats:       stats,
		hostaliases: hostaliases,
		muxTLS:      muxTLS,
		muxPort:     muxPort,
		certPEMFile: certPEMFile,
		keyPEMFile:  keyPEMFile,
		uiConfig:    uiCfg,
		facade:      facade,
	}

	hostAddrs, err := utils.GetIPv4Addresses()
	if err != nil {
		glog.Fatal(err)
	}
	cfg.localAddrs = make(map[string]struct{})
	for _, host := range hostAddrs {
		cfg.localAddrs[host] = struct{}{}
	}

	adminGroup = aGroup
	return &cfg
}

// getPublicEndpointServices returns the list of services for a given
// public endpoint/type.
func getPublicEndpointServices(name string, Type registry.PublicEndpointType) (map[string]struct{}, bool, registry.PublicEndpointKey) {
	allvhostsLock.RLock()
	defer allvhostsLock.RUnlock()
	key := registry.GetPublicEndpointKey(name, Type)
	svcs, found := allvhosts[key]
	return svcs, found, key
}

// getVHostServices returns the list of services for a given vhost (public endpoint)
func getVHostServices(vhostname string) (map[string]struct{}, bool, registry.PublicEndpointKey) {
	return getPublicEndpointServices(vhostname, registry.EPTypeVHost)
}

// getPortServices returns the list of services for a given port (public endpoint)
func getPortServices(port uint8) (map[string]struct{}, bool, registry.PublicEndpointKey) {
	return getPublicEndpointServices(fmt.Sprintf("%d", port), registry.EPTypePort)
}

// Serve handles control center web UI requests and virtual host requests for zenoss web based services.
// The UI server actually listens on port 7878, the uihandler defined here just reverse proxies to it.
// Virtual host routing to zenoss web based services is done by the publicendpointhandler function.
func (sc *ServiceConfig) Serve(shutdown <-chan (interface{})) {

	glog.V(1).Infof("starting vhost synching")
	//start getting vhost endpoints
	go sc.syncPublicEndpoints(shutdown)
	//start watching global vhosts as they are added/deleted/updated in services
	go sc.syncAllVhosts(shutdown)

	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".woff", "application/font-woff")

	accessLogFile, err := os.OpenFile("/var/log/serviced.access.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
	if err != nil {
		glog.Errorf("Could not create access log file.")
	}

	uiHandler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
		Logger: log.New(accessLogFile, "", log.LstdFlags),
	}

	routes := sc.getRoutes()
	uiHandler.SetRoutes(routes...)

	httphandler := func(w http.ResponseWriter, r *http.Request) {
		glog.V(2).Infof("httphandler handling request: %+v", r)

		httphost := strings.Split(r.Host, ":")[0]
		parts := strings.Split(httphost, ".")
		subdomain := parts[0]
		glog.V(2).Infof("httphost: '%s'  subdomain: '%s'", httphost, subdomain)
		isVHost := len(parts) > 1 //Don't look for a vhost if httphost didn't contain a '.'
		if isVHost { 
			if svcIDs, found, registrykey := getVHostServices(httphost); found {
				glog.V(2).Infof("httphost: calling sc.publicendpointhandler")
				sc.publicendpointhandler(w, r, registrykey, svcIDs)
			} else if svcIDs, found, registrykey := getVHostServices(subdomain); found {
				glog.V(2).Infof("httphost: calling sc.publicendpointhandler")
				sc.publicendpointhandler(w, r, registrykey, svcIDs)
			} else {
				isVHost = false
			}	
		}
		 
		if !isVHost {
			glog.V(2).Infof("httphost: calling uiHandler")
			if r.TLS == nil {
				// bindPort has already been validated, so the Split/access below won't break.
				http.Redirect(w, r, fmt.Sprintf("https://%s:%s", r.Host, strings.Split(sc.bindPort, ":")[1]), http.StatusMovedPermanently)
				return
			}
			uiHandler.ServeHTTP(w, r)
		}
	}

	r := mux.NewRouter()

	if hnm, err := os.Hostname(); err == nil {
		sc.hostaliases = append(sc.hostaliases, hnm)
	}

	cmd := exec.Command("hostname", "--fqdn")
	if hnm, err := cmd.CombinedOutput(); err == nil {
		sc.hostaliases = append(sc.hostaliases, string(hnm[:len(hnm)-1]))
	}

	defaultHostAlias = sc.hostaliases[0]
	uiConfig = sc.uiConfig

	r.HandleFunc("/", httphandler)
	r.HandleFunc("/{path:.*}", httphandler)

	http.Handle("/", r)

	// FIXME: bubble up these errors to the caller
	certFile := sc.certPEMFile
	if len(certFile) == 0 {
		tempCertFile, err := proxy.TempCertFile()
		if err != nil {
			glog.Fatalf("Could not prepare cert.pem file: %s", err)
		}
		certFile = tempCertFile
	}
	keyFile := sc.keyPEMFile
	if len(keyFile) == 0 {
		tempKeyFile, err := proxy.TempKeyFile()
		if err != nil {
			glog.Fatalf("Could not prepare key.pem file: %s", err)
		}
		keyFile = tempKeyFile
	}

	go func() {
		redirect := func(w http.ResponseWriter, req *http.Request) {
			// bindPort has already been validated, so the Split/access below won't break.
			http.Redirect(w, req, fmt.Sprintf("https://%s:%s%s", req.Host, strings.Split(sc.bindPort, ":")[1], req.URL), http.StatusMovedPermanently)
		}
		err := http.ListenAndServe(":80", http.HandlerFunc(redirect))
		if err != nil {
			glog.Errorf("could not setup HTTP webserver: %s", err)
		}
	}()
	go func() {
		// This cipher suites and tls min version change may not be needed with golang 1.5
		// https://github.com/golang/go/issues/10094
		// https://github.com/golang/go/issues/9364
		config := &tls.Config{
			MinVersion:               utils.MinTLS(),
			PreferServerCipherSuites: true,
			CipherSuites:             utils.CipherSuites(),
		}
		server := &http.Server{Addr: sc.bindPort, TLSConfig: config}
		err := server.ListenAndServeTLS(certFile, keyFile)
		if err != nil {
			glog.Fatalf("could not setup HTTPS webserver: %s", err)
		}
	}()
	blockerChan := make(chan bool)
	<-blockerChan
}

var methods = []string{"GET", "POST", "PUT", "DELETE"}

func routeToInternalServiceProxy(path string, target string, requiresAuth bool, routes []rest.Route) []rest.Route {
	targetURL, err := url.Parse(target)
	if err != nil {
		glog.Errorf("Unable to parse proxy target URL: %s", target)
		return routes
	}
	// Wrap the normal http.Handler in a rest.handlerFunc
	handlerFunc := func(w *rest.ResponseWriter, r *rest.Request) {
		// All proxied requests should be authenticated first
		if requiresAuth && !loginOK(r) {
			restUnauthorized(w)
			return
		}
		proxy := node.NewReverseProxy(path, targetURL)
		proxy.ServeHTTP(w.ResponseWriter, r.Request)
	}
	// Add on a glob to match subpaths
	andsubpath := path + "*x"
	for _, method := range methods {
		routes = append(routes, rest.Route{method, path, handlerFunc})
		routes = append(routes, rest.Route{method, andsubpath, handlerFunc})
	}
	return routes
}

func (sc *ServiceConfig) unAuthorizedClient(realfunc handlerClientFunc) handlerFunc {
	return func(w *rest.ResponseWriter, r *rest.Request) {
		client, err := sc.getClient()
		if err != nil {
			glog.Errorf("Unable to acquire client: %v", err)
			restServerError(w, err)
			return
		}
		defer client.Close()
		realfunc(w, r, client)
	}
}

func (sc *ServiceConfig) authorizedClient(realfunc handlerClientFunc) handlerFunc {
	return func(w *rest.ResponseWriter, r *rest.Request) {
		if !loginOK(r) {
			restUnauthorized(w)
			return
		}
		client, err := sc.getClient()
		if err != nil {
			glog.Errorf("Unable to acquire client: %v", err)
			restServerError(w, err)
			return
		}
		defer client.Close()
		realfunc(w, r, client)
	}
}

func (sc *ServiceConfig) isCollectingStats() handlerFunc {
	if sc.stats {
		return func(w *rest.ResponseWriter, r *rest.Request) {
			w.WriteHeader(http.StatusOK)
		}
	}
	return func(w *rest.ResponseWriter, r *rest.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func (sc *ServiceConfig) getClient() (c *node.ControlClient, err error) {
	// setup the client
	if c, err = node.NewControlClient(sc.agentPort); err != nil {
		glog.Errorf("Could not create a control center client: %s", err)
	}
	return
}

func (sc *ServiceConfig) getMasterClient() (master.ClientInterface, error) {
	glog.V(2).Infof("start getMasterClient ... sc.agentPort: %+v", sc.agentPort)
	c, err := master.NewClient(sc.agentPort)
	if err != nil {
		glog.Errorf("Could not create a control center client to %v: %v", sc.agentPort, err)
		return nil, err
	}
	glog.V(2).Info("end getMasterClient")
	return c, nil
}

func (sc *ServiceConfig) newRequestHandler(check checkFunc, realfunc ctxhandlerFunc) handlerFunc {
	return func(w *rest.ResponseWriter, r *rest.Request) {
		if !check(w, r) {
			return
		}
		reqCtx := newRequestContext(sc)
		defer reqCtx.end()
		realfunc(w, r, reqCtx)
	}
}

func (sc *ServiceConfig) checkAuth(realfunc ctxhandlerFunc) handlerFunc {
	check := func(w *rest.ResponseWriter, r *rest.Request) bool {
		if !loginOK(r) {
			restUnauthorized(w)
			return false
		}
		return true
	}
	return sc.newRequestHandler(check, realfunc)
}

func (sc *ServiceConfig) noAuth(realfunc ctxhandlerFunc) handlerFunc {
	check := func(w *rest.ResponseWriter, r *rest.Request) bool {
		return true
	}
	return sc.newRequestHandler(check, realfunc)
}

type requestContext struct {
	sc      *ServiceConfig
	master  master.ClientInterface
	dataCtx datastore.Context
}

func newRequestContext(sc *ServiceConfig) *requestContext {
	return &requestContext{sc: sc}
}

func (ctx *requestContext) getMasterClient() (master.ClientInterface, error) {
	if ctx.master == nil {
		c, err := ctx.sc.getMasterClient()
		if err != nil {
			glog.Errorf("Could not create a control center client: %v", err)
			return nil, err
		}
		ctx.master = c
	}
	return ctx.master, nil
}

func (ctx *requestContext) getFacade() facade.FacadeInterface {
	return ctx.sc.facade
}

func (ctx *requestContext) getDatastoreContext() datastore.Context {
	//here in case we ever need to create a per request datastore context
	if ctx.dataCtx == nil {
		ctx.dataCtx = datastore.Get()
	}
	return ctx.dataCtx
}

func (ctx *requestContext) end() error {
	if ctx.master != nil {
		return ctx.master.Close()
	}
	return nil
}

type ctxhandlerFunc func(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext)
type checkFunc func(w *rest.ResponseWriter, r *rest.Request) bool

type getRoutes func(sc *ServiceConfig) []rest.Route

var (
	allvhostsLock sync.RWMutex
	allvhosts     map[registry.PublicEndpointKey]map[string]struct{} // map of PublicEndpointKey to service IDs that have the vhost enabled
)

func init() {
	allvhosts = make(map[registry.PublicEndpointKey]map[string]struct{})
}

func (sc *ServiceConfig) syncAllVhosts(shutdown <-chan interface{}) error {
	rootConn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("syncAllVhosts - Error getting root zk connection: %v", err)
		return err
	}

	cancelChan := make(chan interface{})
	syncVhosts := func(conn client.Connection, parentPath string, childIDs ...string) {
		glog.V(1).Infof("syncVhosts STARTING for parentPath:%s childIDs:%v", parentPath, childIDs)

		newVhosts := make(map[registry.PublicEndpointKey]map[string]struct{})
		for _, sv := range childIDs {
			//cast to a VHostKey so we don't have to care about the format of the key string
			pep := service.PublicEndpointKey(sv)
			if pep.Type() == registry.EPTypeVHost {
				registryKey := registry.GetPublicEndpointKey(pep.Name(), pep.Type())
				vhostServices, found := newVhosts[registryKey]
				if !found {
					vhostServices = make(map[string]struct{})
					newVhosts[registryKey] = vhostServices
				}
				if pep.IsEnabled() {
					vhostServices[pep.ServiceID()] = struct{}{}
				}
			}
		}

		//lock for as short a time as possible
		allvhostsLock.Lock()
		defer allvhostsLock.Unlock()
		allvhosts = newVhosts
		glog.V(1).Infof("allvhosts: %+v", allvhosts)
	}

	for {
		zkServiceVhost := service.ZKServicePublicEndpoints
		select {
		case <-shutdown:
			close(cancelChan)
			return nil
		default:
		}
		glog.V(1).Infof("Running registry.WatchChildren for zookeeper path: %s", zkServiceVhost)
		err := registry.WatchChildren(rootConn, zkServiceVhost, cancelChan, syncVhosts, pepWatchError)
		if err != nil {
			glog.V(1).Infof("Will retry in 10 seconds to WatchChildren(%s) due to error: %v", zkServiceVhost, err)
			<-time.After(time.Second * 10)
			continue
		}
	}
}
