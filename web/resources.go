package web

import (
	"github.com/ant0ine/go-json-rest"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/svc"
	"github.com/zenoss/serviced"
	agent "github.com/zenoss/serviced/agent"
	clientlib "github.com/zenoss/serviced/client"
	"github.com/zenoss/serviced/proxy"

	"os"
	"net"
	"strings"
	"net/http"
	"net/rpc"
	"net/url"
)

type ServiceConfig struct {
	AgentPort   string
	MasterPort  string
	DbString    string
	MuxPort     int
	Tls         bool
	KeyPEMFile  string
	CertPEMFile string
	Zookeepers  []string
}

type HandlerFunc func(w *rest.ResponseWriter, r *rest.Request)
type HandlerClientFunc func(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient)

var started bool
var masterService *svc.ControlSvc
var configuration ServiceConfig

func AuthorizedClient(realfunc HandlerClientFunc) HandlerFunc {
	return func(w *rest.ResponseWriter, r *rest.Request) {
		if !LoginOk(r) {
			RestUnauthorized(w)
			return 
		}
		client, err := getClient()
		if err != nil {
			glog.Errorf("Unable to acquire client: %v", err)
			RestServerError(w)
			return
		}
		defer client.Close()
		realfunc(w, r, client)
	}
}

func RestGetPools(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	request := serviced.EntityRequest{}
	var poolsMap map[string]*serviced.ResourcePool
	err := client.GetResourcePools(request, &poolsMap)
	if err != nil {
		glog.Errorf("Could not get resource pools: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&poolsMap)
}

func RestAddPool(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var payload serviced.ResourcePool
	var unused int
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.Infof("Could not decode pool payload: %v", err)
		RestBadRequest(w)
		return
	}
	err = client.AddResourcePool(payload, &unused)
	if err != nil {
		glog.Errorf("Unable to add pool: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&SimpleResponse{"Added resource pool", poolsLink()})
}

func RestUpdatePool(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	poolId, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	glog.Infof("Received update request for %s", poolId)
	var payload serviced.ResourcePool
	var unused int
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.Infof("Could not decode pool payload: %v", err)
		RestBadRequest(w)
		return
	}
	err = client.UpdateResourcePool(payload, &unused)
	if err != nil {
		glog.Errorf("Unable to update pool: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&SimpleResponse{"Updated resource pool", poolsLink()})
}

func RestRemovePool(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	poolId, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var unused int
	err = client.RemoveResourcePool(poolId, &unused)
	if err != nil {
		glog.Errorf("Could not remove resource pool: %v", err)
		RestServerError(w)
		return
	}
	glog.Infof("Removed pool %s", poolId)
	w.WriteJson(&SimpleResponse{"Removed resource pool", poolsLink()})
}

func RestGetHosts(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var hosts map[string]*serviced.Host
	request := serviced.EntityRequest{}
	err := client.GetHosts(request, &hosts)
	if err != nil {
		glog.Errorf("Could not get hosts: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&hosts)
}

func RestGetServices(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var services []*serviced.Service
	request := serviced.EntityRequest{}
	err := client.GetServices(request, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		services = []*serviced.Service{}
	}
	w.WriteJson(&services)
}

func RestAddService(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var payload serviced.Service
	var unused int
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.Infof("Could not decode service payload: %v", err)
		RestBadRequest(w)
		return
	}
	service, err := serviced.NewService()
	if err != nil {
		glog.Errorf("Could not create service: %v", err)
		RestServerError(w)
		return
	}
	service.Name = payload.Name
	service.Description = payload.Description
	service.PoolId = payload.PoolId
	service.ImageId = payload.ImageId
	service.Startup = payload.Startup

	err = client.AddService(*service, &unused)
	if err != nil {
		glog.Errorf("Unable to add service: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&SimpleResponse{"Added service", servicesLink()})
}

func RestUpdateService(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	glog.Infof("Received update request for %s", serviceId)
	var payload serviced.Service
	var unused int
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.Infof("Could not decode service payload: %v", err)
		RestBadRequest(w)
		return
	}
	err = client.UpdateService(payload, &unused)
	if err != nil {
		glog.Errorf("Unable to update service: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&SimpleResponse{"Updated service", servicesLink()})
}


func RestRemoveService(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var unused int
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	err = client.RemoveService(serviceId, &unused)
	if err != nil {
		glog.Errorf("Could not remove service: %v", err)
		RestServerError(w)
		return
	}
	glog.Infof("Removed service %s", serviceId)
	w.WriteJson(&SimpleResponse{"Removed service", servicesLink()})
}

func RestGetHostsForResourcePool(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var poolHosts []*serviced.PoolHost
	poolId, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		glog.Infof("Unable to acquire pool ID: %v", err)
		RestBadRequest(w)
		return
	}
	err = client.GetHostsForResourcePool(poolId, &poolHosts)
	if err != nil {
		glog.Errorf("Could not get hosts: %v", err)
		RestServerError(w)
		return
	}
	if poolHosts == nil {
		poolHosts = []*serviced.PoolHost{}
	}
	w.WriteJson(&poolHosts)
}

func RestAddHost(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var payload serviced.Host
	var unused int
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.Infof("Could not decode host payload: %v", err)
		RestBadRequest(w)
		return
	}
	// Save the pool ID and IP address for later. GetInfo wipes these
	pool := payload.PoolId 
	ipAddr := payload.IpAddr
	remoteClient, err := clientlib.NewAgentClient(payload.IpAddr)
	if err != nil {
		glog.Errorf("Could not create connection to host %s: %v", payload.IpAddr, err)
		RestServerError(w)
		return
	}

	err = remoteClient.GetInfo(0, &payload)
	if err != nil {
		glog.Errorf("Unable to get remote host info: %v", err)
		RestBadRequest(w);
		return
	}
	// Reset the pool ID and IP address
	payload.PoolId = pool
	parts := strings.Split(ipAddr, ":")
	payload.IpAddr = parts[0]
	
	err = client.AddHost(payload, &unused)
	if err != nil {
		glog.Errorf("Unable to add host: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&SimpleResponse{"Added host", hostsLink()})
}

func RestUpdateHost(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	hostId, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	glog.Infof("Received update request for %s", hostId)
	var payload serviced.Host
	var unused int
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.Infof("Could not decode host payload: %v", err)
		RestBadRequest(w)
		return
	}
	err = client.UpdateHost(payload, &unused)
	if err != nil {
		glog.Errorf("Unable to update host: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&SimpleResponse{"Updated host", hostsLink()})
}

func RestRemoveHost(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var unused int
	hostId, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	err = client.RemoveHost(hostId, &unused)
	if err != nil {
		glog.Errorf("Could not remove host: %v", err)
		RestServerError(w)
		return
	}
	glog.Infof("Removed host %s", hostId)
	w.WriteJson(&SimpleResponse{"Removed host", hostsLink()})
}


func startServer(options *ServiceConfig) {
	master, err := svc.NewControlSvc(options.DbString, options.Zookeepers)
	if err != nil {
		glog.Fatalf("Could not start ControlPlane service: %v", err)
	}
	// register the API
	glog.Infoln("registering ControlPlane service")
	rpc.RegisterName("LoadBalancer", master)
	rpc.RegisterName("ControlPlane", master)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", options.MasterPort)
	if err != nil {
		glog.Fatalf("Could not bind to port %v", err)
	}
	glog.Infof("Listening on %s", l.Addr().String())
	started = true
	masterService = master
	glog.Flush()
	go startAgent(options)
	http.Serve(l, nil) // start the server
}

func startAgent(options *ServiceConfig) {
	mux := proxy.TCPMux{}
	mux.CertPEMFile = options.CertPEMFile
	mux.KeyPEMFile = options.KeyPEMFile
	mux.Enabled = true
	mux.Port = options.MuxPort
	mux.UseTLS = options.Tls
	agent, err := agent.NewHostAgent(options.AgentPort, mux)
	if err != nil {
		glog.Fatalf("Could not start ControlPlane agent: %v", err)
	}
	// register the API
	glog.Infoln("registering ControlPlaneAgent service")
	rpc.RegisterName("ControlPlaneAgent", agent)
}

func init() {
	configuration = ServiceConfig{}
	configDefaults(&configuration)
	go startServer(&configuration)
}

func configDefaults(cfg *ServiceConfig) {
	if len(cfg.AgentPort) == 0 {
		cfg.AgentPort = "127.0.0.1:4979"
	}
	if len(cfg.MasterPort) == 0 {
		cfg.MasterPort = ":4979"
	}
	if cfg.MuxPort == 0 {
		cfg.MuxPort = 22250
	}
	conStr := os.Getenv("CP_PROD_DB")
	if len(conStr) == 0 {
		conStr = "mysql://root@127.0.0.1:3306/cp"
	} else {
		glog.Infoln("Using connection string from env var CP_PROD_DB")
	}
	if len(cfg.DbString) == 0 {
		cfg.DbString = conStr
	}
}

func getClient() (c *clientlib.ControlClient, err error) {
	// setup the client
	c, err = clientlib.NewControlClient(configuration.AgentPort)
	if err != nil {
		glog.Fatalf("Could not create a control plane client: %v", err)
	}
	return c, err
}


