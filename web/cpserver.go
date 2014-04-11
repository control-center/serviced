package web

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gorilla/mux"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"

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

func NewServiceConfig(bindPort string, agentPort string, zookeepers []string, stats bool, hostaliases string) *ServiceConfig {
	cfg := ServiceConfig{bindPort, agentPort, zookeepers, stats, []string{}}
	if hostaliases != "" {
		cfg.hostaliases = strings.Split(hostaliases, ":")
	}
	if len(cfg.bindPort) == 0 {
		cfg.bindPort = ":8787"
	}
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
		var empty interface{}
		services := []*dao.RunningService{}
		client.GetRunningServices(&empty, &services)

		vhosts := make(map[string][]*dao.ServiceState, 0)

		for _, s := range services {
			var svc dao.Service

			if err := client.GetService(s.ServiceId, &svc); err != nil {
				glog.Errorf("Can't get service: %s (%v)", s.Id, err)
			}

			vheps := svc.GetServiceVHosts()

			for _, vhep := range vheps {
				for _, vh := range vhep.VHosts {
					svcstates := []*dao.ServiceState{}
					if err := client.GetServiceStates(s.ServiceId, &svcstates); err != nil {
						http.Error(w, fmt.Sprintf("can't retrieve service states for %s (%v)", s.ServiceId, err), http.StatusInternalServerError)
						return
					}

					for _, ss := range svcstates {
						vhosts[vh] = append(vhosts[vh], ss)
					}
				}
			}

		}

		muxvars := mux.Vars(r)
		svcstates, ok := vhosts[muxvars["subdomain"]]
		if !ok {
			http.Error(w, fmt.Sprintf("unknown vhost: %v", muxvars["subdomain"]), http.StatusNotFound)
			return
		}

		// TODO: implement a more intelligent strategy than "always pick the first one" when more
		// than one service state is mapped to a given virtual host
		for _, svcep := range svcstates[0].Endpoints {
			for _, vh := range svcep.VHosts {
				if vh == muxvars["subdomain"] {
					rp := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: fmt.Sprintf("%s:%d", svcstates[0].PrivateIp, svcep.PortNumber)})
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
		r.HandleFunc("/", vhosthandler).Host(fmt.Sprintf("{subdomain}.%s", ha))
		r.HandleFunc("/{path:.*}", vhosthandler).Host(fmt.Sprintf("{subdomain}.%s", ha))
	}

	r.HandleFunc("/{path:.*}", uihandler)

	http.Handle("/", r)

	certfile, err := serviced.TempCertFile()
	if err != nil {
		glog.Error("Could not prepare cert.pem file.")
	}
	keyfile, err := serviced.TempKeyFile()
	if err != nil {
		glog.Error("Could not prepare key.pem file.")
	}
	http.ListenAndServeTLS(sc.bindPort, certfile, keyfile, nil)
}

func (this *ServiceConfig) ServeUI() {
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".woff", "application/font-woff")

	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
	}
	routes := []rest.Route{
		rest.Route{"GET", "/", MainPage},
		rest.Route{"GET", "/test", TestPage},
		rest.Route{"GET", "/stats", this.IsCollectingStats()},
		// Hosts
		rest.Route{"GET", "/hosts", this.AuthorizedClient(RestGetHosts)},
		rest.Route{"POST", "/hosts/add", this.AuthorizedClient(RestAddHost)},
		rest.Route{"DELETE", "/hosts/:hostId", this.AuthorizedClient(RestRemoveHost)},
		rest.Route{"PUT", "/hosts/:hostId", this.AuthorizedClient(RestUpdateHost)},
		rest.Route{"GET", "/hosts/:hostId/running", this.AuthorizedClient(RestGetRunningForHost)},
		rest.Route{"DELETE", "/hosts/:hostId/:serviceStateId", this.AuthorizedClient(RestKillRunning)},
		// Pools
		rest.Route{"POST", "/pools/add", this.AuthorizedClient(RestAddPool)},
		rest.Route{"GET", "/pools/:poolId/hosts", this.AuthorizedClient(RestGetHostsForResourcePool)},
		rest.Route{"DELETE", "/pools/:poolId", this.AuthorizedClient(RestRemovePool)},
		rest.Route{"PUT", "/pools/:poolId", this.AuthorizedClient(RestUpdatePool)},
		rest.Route{"GET", "/pools", this.AuthorizedClient(RestGetPools)},
		// Services (Apps)
		rest.Route{"GET", "/services", this.AuthorizedClient(RestGetAllServices)},
		rest.Route{"GET", "/services/:serviceId", this.AuthorizedClient(RestGetService)},
		rest.Route{"GET", "/services/:serviceId/running", this.AuthorizedClient(RestGetRunningForService)},
		rest.Route{"GET", "/services/:serviceId/running/:serviceStateId", this.AuthorizedClient(RestGetRunningService)},
		rest.Route{"GET", "/services/:serviceId/:serviceStateId/logs", this.AuthorizedClient(RestGetServiceStateLogs)},
		rest.Route{"POST", "/services/add", this.AuthorizedClient(RestAddService)},
		rest.Route{"DELETE", "/services/:serviceId", this.AuthorizedClient(RestRemoveService)},
		rest.Route{"GET", "/services/:serviceId/logs", this.AuthorizedClient(RestGetServiceLogs)},
		rest.Route{"PUT", "/services/:serviceId", this.AuthorizedClient(RestUpdateService)},
		rest.Route{"GET", "/services/:serviceId/snapshot", this.AuthorizedClient(RestSnapshotService)},
		rest.Route{"PUT", "/services/:serviceId/startService", this.AuthorizedClient(RestStartService)},
		rest.Route{"PUT", "/services/:serviceId/stopService", this.AuthorizedClient(RestStopService)},

		// Services (Virtual Host)
		rest.Route{"GET", "/vhosts", this.AuthorizedClient(RestGetVirtualHosts)},
		rest.Route{"POST", "/vhosts/:serviceId/:application/:vhostName", this.AuthorizedClient(RestAddVirtualHost)},
		rest.Route{"DELETE", "/vhosts/:serviceId/:application/:vhostName", this.AuthorizedClient(RestRemoveVirtualHost)},

		// Service templates (App templates)
		rest.Route{"GET", "/templates", this.AuthorizedClient(RestGetAppTemplates)},
		rest.Route{"POST", "/templates/deploy", this.AuthorizedClient(RestDeployAppTemplate)},
		// Login
		rest.Route{"POST", "/login", this.UnAuthorizedClient(RestLogin)},
		rest.Route{"DELETE", "/login", RestLogout},
		// "Misc" stuff
		rest.Route{"GET", "/top/services", this.AuthorizedClient(RestGetTopServices)},
		rest.Route{"GET", "/running", this.AuthorizedClient(RestGetAllRunning)},
		// Generic static data
		rest.Route{"GET", "/favicon.ico", FavIcon},
		rest.Route{"GET", "/static*resource", StaticData},
	}

	// Hardcoding these target URLs for now.
	// TODO: When internal services are allowed to run on other hosts, look that up.
	routes = routeToInternalServiceProxy("/elastic", "http://127.0.0.1:9200/", routes)
	routes = routeToInternalServiceProxy("/metrics", "http://127.0.0.1:8888/", routes)

	handler.SetRoutes(routes...)

	http.ListenAndServe(":7878", &handler)
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
