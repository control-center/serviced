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
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/gorilla/mux"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

// ServiceConfig is the ui/rest handler for control center
type ServiceConfig struct {
	bindPort    string
	agentPort   string
	stats       bool
	hostaliases []string
	muxTLS      bool
	muxPort     int
}

var defaultHostAlias string

// NewServiceConfig creates a new ServiceConfig
func NewServiceConfig(bindPort string, agentPort string, stats bool, hostaliases []string, muxTLS bool, muxPort int, aGroup string) *ServiceConfig {
	cfg := ServiceConfig{
		bindPort:    bindPort,
		agentPort:   agentPort,
		stats:       stats,
		hostaliases: hostaliases,
		muxTLS:      muxTLS,
		muxPort:     muxPort,
	}
	adminGroup = aGroup
	if len(cfg.agentPort) == 0 {
		cfg.agentPort = "127.0.0.1:4979"
	}
	return &cfg
}

// Serve handles control center web UI requests and virtual host requests for zenoss web based services.
// The UI server actually listens on port 7878, the uihandler defined here just reverse proxies to it.
// Virtual host routing to zenoss web based services is done by the vhosthandler function.
func (sc *ServiceConfig) Serve(shutdown <-chan (interface{})) {

	glog.V(1).Infof("starting vhost synching")
	//start getting vhost endpoints
	go sc.syncVhosts(shutdown)
	//start watching global vhosts as they are added/deleted/updated in services
	go sc.syncAllVhosts(shutdown)

	// Reverse proxy to the web UI server.
	uihandler := func(w http.ResponseWriter, r *http.Request) {
		uiURL, err := url.Parse("http://127.0.0.1:7878")
		if err != nil {
			glog.Errorf("Can't parse UI URL: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ui := httputil.NewSingleHostReverseProxy(uiURL)
		if ui == nil {
			glog.Errorf("Can't proxy UI request: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ui.ServeHTTP(w, r)
	}

	httphandler := func(w http.ResponseWriter, r *http.Request) {
		glog.V(2).Infof("httphandler handling: %+v", r)

		httphost := r.Host
		parts := strings.Split(httphost, ".")
		subdomain := parts[0]
		glog.V(2).Infof("httphost: '%s'  subdomain: '%s'", httphost, subdomain)

		allvhostsLock.Lock()
		defer allvhostsLock.Unlock()
		if _, ok := allvhosts[httphost]; ok {
			sc.vhosthandler(w, r, httphost)
		} else if _, ok := allvhosts[subdomain]; ok {
			sc.vhosthandler(w, r, subdomain)
		} else {
			uihandler(w, r)
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

	r.HandleFunc("/", httphandler)
	r.HandleFunc("/{path:.*}", httphandler)

	http.Handle("/", r)

	// FIXME: bubble up these errors to the caller
	certfile, err := proxy.TempCertFile()
	if err != nil {
		glog.Fatalf("Could not prepare cert.pem file: %s", err)
	}
	keyfile, err := proxy.TempKeyFile()
	if err != nil {
		glog.Fatalf("Could not prepare key.pem file: %s", err)
	}
	go func() {
		redirect := func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, fmt.Sprintf("https://%s:%s%s", req.Host, sc.bindPort, req.URL), http.StatusMovedPermanently)
		}
		err = http.ListenAndServe(":80", http.HandlerFunc(redirect))
		if err != nil {
			glog.Errorf("could not setup HTTP webserver: %s", err)
		}
	}()
	go func() {
		err = http.ListenAndServeTLS(sc.bindPort, certfile, keyfile, nil)
		if err != nil {
			glog.Fatalf("could not setup HTTPS webserver: %s", err)
		}
	}()
	blockerChan := make(chan bool)
	<-blockerChan
}

// ServeUI is a blocking call that runs the UI hander on port :7878
func (sc *ServiceConfig) ServeUI() {
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".woff", "application/font-woff")

	accessLogFile, err := os.OpenFile("/var/log/serviced.access.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
	if err != nil {
		glog.Errorf("Could not create access log file.")
	}

	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
		Logger: log.New(accessLogFile, "", log.LstdFlags),
	}

	routes := sc.getRoutes()
	handler.SetRoutes(routes...)

	// FIXME: bubble up these errors to the caller
	if err := http.ListenAndServe(":7878", &handler); err != nil {
		glog.Fatalf("could not setup internal web server: %s", err)
	}
}

var methods = []string{"GET", "POST", "PUT", "DELETE"}

func routeToInternalServiceProxy(path string, target string, routes []rest.Route) []rest.Route {
	targetURL, err := url.Parse(target)
	if err != nil {
		glog.Errorf("Unable to parse proxy target URL: %s", target)
		return routes
	}
	// Wrap the normal http.Handler in a rest.handlerFunc
	handlerFunc := func(w *rest.ResponseWriter, r *rest.Request) {
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
	c, err = node.NewControlClient(sc.agentPort)
	if err != nil {
		glog.Fatalf("Could not create a control center client: %v", err)
	}
	return c, err
}

func (sc *ServiceConfig) getMasterClient() (*master.Client, error) {
	glog.Infof("start getMasterClient ... sc.agentPort: %+v", sc.agentPort)
	c, err := master.NewClient(sc.agentPort)
	if err != nil {
		glog.Errorf("Could not create a control center client to %v: %v", sc.agentPort, err)
		return nil, err
	}
	glog.Info("end getMasterClient")
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
	sc     *ServiceConfig
	master *master.Client
}

func newRequestContext(sc *ServiceConfig) *requestContext {
	return &requestContext{sc: sc}
}

func (ctx *requestContext) getMasterClient() (*master.Client, error) {
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
	allvhosts     map[string]string
)

func init() {
	allvhosts = make(map[string]string)
}

func (sc *ServiceConfig) syncAllVhosts(shutdown <-chan interface{}) error {
	rootConn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Fatalf("syncAllVhosts - Error getting root zk connection: %v", err)
		return err
	}

	cancelChan := make(chan bool)
	syncVhosts := func(conn client.Connection, parentPath string, childIDs ...string) {
		glog.V(1).Infof("syncVhosts STARTING for parentPath:%s childIDs:%v", parentPath, childIDs)

		allvhostsLock.Lock()
		defer allvhostsLock.Unlock()

		allvhosts = make(map[string]string)
		for _, sv := range childIDs {
			parts := strings.SplitN(sv, "_", 2)
			allvhosts[parts[1]] = parts[0]
		}
		glog.V(0).Infof("allvhosts: %+v", allvhosts)
	}

	for {
		zkServiceVhost := "/servicevhosts" // should this use the constant from zzk/service/servicevhost?
		glog.V(0).Infof("Running registry.WatchChildren for zookeeper path: %s", zkServiceVhost)
		err := registry.WatchChildren(rootConn, zkServiceVhost, cancelChan, syncVhosts, vhostWatchError)
		if err != nil {
			glog.Warningf("will retry in 10 seconds to WatchChildren(%s) due to error: %v", zkServiceVhost, err)
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

	return nil
}
