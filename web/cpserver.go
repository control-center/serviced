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

	daoclient "github.com/control-center/serviced/dao/client"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/gorilla/mux"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
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
	vhostmgr    *VHostManager
}

var defaultHostAlias string
var uiConfig UIConfig

// NewServiceConfig creates a new ServiceConfig
func NewServiceConfig(bindPort string, agentPort string, stats bool, hostaliases []string, muxTLS bool, muxPort int,
	aGroup string, certPEMFile string, keyPEMFile string, pollFrequency int, configuredSnapshotSpacePercent int, facade facade.FacadeInterface) *ServiceConfig {

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

	snapshotSpacePercent = configuredSnapshotSpacePercent

	return &cfg
}

// Serve handles control center web UI requests and virtual host requests for zenoss web based services.
// The UI server actually listens on port 7878, the uihandler defined here just reverse proxies to it.
// Virtual host routing to zenoss web based services is done by the publicendpointhandler function.
func (sc *ServiceConfig) Serve(shutdown <-chan (interface{})) {

	glog.V(1).Infof("starting vhost synching")

	// start public port listener
	sc.startPublicPortListener(shutdown)

	// start vhost listener
	sc.startVHostListener(shutdown)

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
		if len(parts) > 1 {
			if sc.vhostmgr.Handle(subdomain, w, r) {
				return
			}
		}

		glog.V(2).Infof("httphost: calling uiHandler")
		if r.TLS == nil {
			// bindPort has already been validated, so the Split/access below won't break.
			http.Redirect(w, r, fmt.Sprintf("https://%s:%s", r.Host, strings.Split(sc.bindPort, ":")[1]), http.StatusMovedPermanently)
			return
		}
		uiHandler.ServeHTTP(w, r)
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
	certFile, keyFile := GetCertFiles(sc.certPEMFile, sc.keyPEMFile)

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
			MinVersion:               utils.MinTLS("http"),
			PreferServerCipherSuites: true,
			CipherSuites:             utils.CipherSuites("http"),
		}
		server := &http.Server{Addr: sc.bindPort, TLSConfig: config}
		glog.Infof("Creating HTTP server on port %s with CipherSuite: %v", sc.bindPort, utils.CipherSuitesByName(config))
		err := server.ListenAndServeTLS(certFile, keyFile)
		if err != nil {
			glog.Fatalf("could not setup HTTPS webserver: %s", err)
		}
	}()
	blockerChan := make(chan bool)
	<-blockerChan
}

var methods = []string{"GET", "POST", "PUT", "DELETE", "HEAD"}

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

func (sc *ServiceConfig) getClient() (c *daoclient.ControlClient, err error) {
	// setup the client
	if c, err = daoclient.NewControlClient(sc.agentPort); err != nil {
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

// startPublicPortListener starts sets up the port servers and watches for
// changes in state
func (sc *ServiceConfig) startPublicPortListener(shutdown <-chan interface{}) {
	// set up the public port manager
	pubmgr := NewPublicPortManager("", sc.certPEMFile, sc.keyPEMFile, func(portAddress string, err error) {
		logger := plog.WithField("portAddress", portAddress).WithError(err)

		// connect to zookeeper
		conn, err := zzk.GetLocalConnection("/")
		if err != nil {
			logger.WithError(err).Error("Could not connect to zookeeper")
			return
		}

		// get the public port
		key := registry.PublicPortKey{
			HostID:      "master",
			PortAddress: portAddress,
		}
		serviceID, application, err := registry.GetPublicPort(conn, key)
		if err != nil {
			logger.WithError(err).Error("Could not look up public port")
			return
		}

		// disable the public port
		if err := sc.facade.EnablePublicEndpointPort(datastore.Get(), serviceID, application, portAddress, false); err != nil {
			logger.WithError(err).Error("Could not disable public port")
			return
		}

		logger.Warn("Disabled public port due to error")
	})

	// set up the public port listener
	listener := registry.NewPublicPortListener("master", pubmgr)

	// start the listener
	go func() {
		for {
			select {
			case conn := <-zzk.Connect("/", zzk.GetLocalConnection):
				if conn != nil {
					zzk.Listen(shutdown, make(chan error, 1), conn, listener)
					select {
					case <-shutdown:
						return
					default:
					}
				}
			case <-shutdown:
				return
			}
		}
	}()
}

// startVHostListener manages proxies for all vhosts
func (sc *ServiceConfig) startVHostListener(shutdown <-chan interface{}) {
	// set up the vhost manager
	sc.vhostmgr = NewVHostManager(sc.muxTLS)

	// set up the vhost listener
	listener := registry.NewVHostListener("master", sc.vhostmgr)

	// start the listener
	go func() {
		for {
			select {
			case conn := <-zzk.Connect("/", zzk.GetLocalConnection):
				if conn != nil {
					zzk.Listen(shutdown, make(chan error, 1), conn, listener)
					select {
					case <-shutdown:
						return
					default:
					}
				}
			case <-shutdown:
				return
			}
		}
	}()
}
