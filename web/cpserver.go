package web

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/gorilla/mux"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/proxy"
	"github.com/zenoss/serviced/rpc/master"

	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type ServiceConfig struct {
	bindPort    string
	agentPort   string
	zookeepers  []string
	stats       bool
	hostaliases []string
}

func NewServiceConfig(bindPort string, agentPort string, zookeepers []string, stats bool, hostaliases []string) *ServiceConfig {
	cfg := ServiceConfig{bindPort, agentPort, zookeepers, stats, []string{}}
	if len(cfg.agentPort) == 0 {
		cfg.agentPort = "127.0.0.1:4979"
	}
	if len(cfg.zookeepers) == 0 {
		cfg.zookeepers = []string{"127.0.0.1:2181"}
	}
	return &cfg
}

// Serve handles control plane web UI requests and virtual host requests for zenoss web based services.
// The UI server actually listens on port 7878, the uihandler defined here just reverse proxies to it.
// Virutal host routing to zenoss web based services is done by the vhosthandler function.
func (sc *ServiceConfig) Serve() {
	client, err := sc.getClient()
	if err != nil {
		glog.Errorf("Unable to get control plane client: %v", err)
		return
	}

	// Reverse proxy to the web UI server.
	uihandler := func(w http.ResponseWriter, r *http.Request) {
		uiUrl, err := url.Parse("http://127.0.0.1:7878")
		if err != nil {
			glog.Errorf("Can't parse UI URL: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ui := httputil.NewSingleHostReverseProxy(uiUrl)
		if ui == nil {
			glog.Errorf("Can't proxy UI request: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ui.ServeHTTP(w, r)
	}

	// Lookup the appropriate virtual host and forward the request to it.
	// TODO: when zookeeper registration is integrated we can be more event
	// driven and only refresh the vhost map when service states change.
	vhosthandler := func(w http.ResponseWriter, r *http.Request) {
		glog.V(1).Infof("vhosthandler handling: %v", r)

		var empty interface{}
		services := []*dao.RunningService{}
		client.GetRunningServices(&empty, &services)

		vhosts := make(map[string][]*servicestate.ServiceState, 0)

		for _, s := range services {
			var svc service.Service

			if err := client.GetService(s.ServiceID, &svc); err != nil {
				glog.Errorf("Can't get service: %s (%v)", s.Id, err)
			}

			vheps := svc.GetServiceVHosts()

			for _, vhep := range vheps {
				for _, vh := range vhep.VHosts {
					svcstates := []*servicestate.ServiceState{}
					if err := client.GetServiceStates(s.ServiceID, &svcstates); err != nil {
						http.Error(w, fmt.Sprintf("can't retrieve service states for %s (%v)", s.ServiceID, err), http.StatusInternalServerError)
						return
					}

					for _, ss := range svcstates {
						vhosts[vh] = append(vhosts[vh], ss)
					}
				}
			}

		}

		glog.V(1).Infof("vhosthandler VHost map: %v", vhosts)

		muxvars := mux.Vars(r)
		svcstates, ok := vhosts[muxvars["subdomain"]]
		if !ok {
			http.Error(w, fmt.Sprintf("service associated with vhost %v is not running", muxvars["subdomain"]), http.StatusNotFound)
			return
		}

		// TODO: implement a more intelligent strategy than "always pick the first one" when more
		// than one service state is mapped to a given virtual host
		for _, svcep := range svcstates[0].Endpoints {
			for _, vh := range svcep.VHosts {
				if vh == muxvars["subdomain"] {
					rpurl := url.URL{Scheme: "http", Host: fmt.Sprintf("%s:%d", svcstates[0].PrivateIp, svcep.PortNumber)}

					glog.V(1).Infof("vhosthandler reverse proxy to: %v", rpurl)

					rp := httputil.NewSingleHostReverseProxy(&rpurl)
					rp.ServeHTTP(w, r)
					return
				}
			}
		}

		http.Error(w, fmt.Sprintf("unrecognized endpoint: %s", muxvars["subdomain"]), http.StatusNotImplemented)
	}

	r := mux.NewRouter()

	if hnm, err := os.Hostname(); err == nil {
		sc.hostaliases = append(sc.hostaliases, hnm)
	}

	cmd := exec.Command("hostname", "--fqdn")
	if hnm, err := cmd.CombinedOutput(); err == nil {
		sc.hostaliases = append(sc.hostaliases, string(hnm[:len(hnm)-1]))
	}

	for _, ha := range sc.hostaliases {
		glog.V(1).Infof("Use vhosthandler for: %s", fmt.Sprintf("{subdomain}.%s", ha))
		r.HandleFunc("/{path:.*}", vhosthandler).Host(fmt.Sprintf("{subdomain}.%s", ha))
		r.HandleFunc("/", vhosthandler).Host(fmt.Sprintf("{subdomain}.%s", ha))
	}

	r.HandleFunc("/{path:.*}", uihandler)

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
	err = http.ListenAndServeTLS(sc.bindPort, certfile, keyfile, nil)
	if err != nil {
		glog.Fatalf("could not setup webserver: %s", err)
	}
}

func (this *ServiceConfig) ServeUI() {
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".woff", "application/font-woff")

	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
	}

	routes := this.getRoutes()
	handler.SetRoutes(routes...)

	// FIXME: bubble up these errors to the caller
	if err := http.ListenAndServe(":7878", &handler); err != nil {
		glog.Fatalf("could not setup internal web server: %s", err)
	}
}

var methods []string = []string{"GET", "POST", "PUT", "DELETE"}

func routeToInternalServiceProxy(path string, target string, routes []rest.Route) []rest.Route {
	targetUrl, err := url.Parse(target)
	if err != nil {
		glog.Errorf("Unable to parse proxy target URL: %s", target)
		return routes
	}
	// Wrap the normal http.Handler in a rest.HandlerFunc
	handlerFunc := func(w *rest.ResponseWriter, r *rest.Request) {
		proxy := serviced.NewReverseProxy(path, targetUrl)
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

func (this *ServiceConfig) UnAuthorizedClient(realfunc HandlerClientFunc) HandlerFunc {
	return func(w *rest.ResponseWriter, r *rest.Request) {
		client, err := this.getClient()
		if err != nil {
			glog.Errorf("Unable to acquire client: %v", err)
			RestServerError(w)
			return
		}
		defer client.Close()
		realfunc(w, r, client)
	}
}

func (this *ServiceConfig) AuthorizedClient(realfunc HandlerClientFunc) HandlerFunc {
	return func(w *rest.ResponseWriter, r *rest.Request) {
		if !LoginOk(r) {
			RestUnauthorized(w)
			return
		}
		client, err := this.getClient()
		if err != nil {
			glog.Errorf("Unable to acquire client: %v", err)
			RestServerError(w)
			return
		}
		defer client.Close()
		realfunc(w, r, client)
	}
}

func (this *ServiceConfig) IsCollectingStats() HandlerFunc {
	if this.stats {
		return func(w *rest.ResponseWriter, r *rest.Request) {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		return func(w *rest.ResponseWriter, r *rest.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		}
	}
}

func (this *ServiceConfig) getClient() (c *serviced.ControlClient, err error) {
	// setup the client
	c, err = serviced.NewControlClient(this.agentPort)
	if err != nil {
		glog.Fatalf("Could not create a control plane client: %v", err)
	}
	return c, err
}

func (sc *ServiceConfig) newRequestHandler(check CheckFunc, realfunc CtxHandlerFunc) HandlerFunc {
	return func(w *rest.ResponseWriter, r *rest.Request) {
		if !check(w, r) {
			return
		}
		reqCtx := newRequestContext(sc)
		defer reqCtx.end()
		realfunc(w, r, reqCtx)
	}
}

func (sc *ServiceConfig) CheckAuth(realfunc CtxHandlerFunc) HandlerFunc {
	check := func(w *rest.ResponseWriter, r *rest.Request) bool {
		if !LoginOk(r) {
			RestUnauthorized(w)
			return false
		}
		return true
	}
	return sc.newRequestHandler(check, realfunc)
}

func (sc *ServiceConfig) NoAuth(realfunc CtxHandlerFunc) HandlerFunc {
	check := func(w *rest.ResponseWriter, r *rest.Request) bool {
		return true
	}
	return sc.newRequestHandler(check, realfunc)
}

type Close interface {
	Close() error
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
		if c, err := master.NewClient(ctx.sc.agentPort); err != nil {
			glog.Errorf("Could not create a control plane client to %v: %v", ctx.sc.agentPort, err)
			return nil, err
		} else {
			ctx.master = c
		}
	}
	return ctx.master, nil
}

func (ctx *requestContext) end() error {
	if ctx.master != nil {
		return ctx.master.Close()
	}
	return nil
}

type CtxHandlerFunc func(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext)
type CheckFunc func(w *rest.ResponseWriter, r *rest.Request) bool

type getRoutes func(sc *ServiceConfig) []rest.Route
