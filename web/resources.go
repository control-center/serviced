package web

import (
	"github.com/ant0ine/go-json-rest"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	clientlib "github.com/zenoss/serviced/client"

	"net/url"
	"os"
	"regexp"
	"strings"
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

func RestGetAppTemplates(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var unused int
	var templatesMap map[string]*serviced.ServiceTemplate
	client.GetServiceTemplates(unused, &templatesMap)
	w.WriteJson(&templatesMap)
}

func RestDeployAppTemplate(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var payload serviced.ServiceTemplateDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.Infof("Could not decode deployment payload: %v", err)
		RestBadRequest(w)
		return
	}
	var unused int
	err = client.DeployTemplate(payload, &unused)
	if err != nil {
		glog.Errorf("Could not deploy template: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&SimpleResponse{"Removed resource pool", servicesLink()})
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

func RestGetAllServices(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
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

	nmregex := r.URL.Query().Get("name")

	if nmregex == "" {
		w.WriteJson(&services)
	} else {
		r, err := regexp.Compile(nmregex)
		if err != nil {
			glog.Errorf("Bad name regexp :%s", nmregex)
			RestServerError(w)
			return
		}
		matches := []*serviced.Service{}
		for _, service := range services {
			if r.MatchString(service.Name) {
				matches = append(matches, service)
			}
		}
		w.WriteJson(&matches)
	}
}

func RestGetRunningForHost(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	hostId, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var services []*serviced.RunningService
	err = client.GetRunningServicesForHost(hostId, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		services = []*serviced.RunningService{}
	}
	w.WriteJson(&services)
}

func RestGetRunningForService(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var services []*serviced.RunningService
	err = client.GetRunningServicesForService(serviceId, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		services = []*serviced.RunningService{}
	}
	w.WriteJson(&services)
}


func RestGetAllRunning(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var services []*serviced.RunningService
	request := serviced.EntityRequest{}
	err := client.GetRunningServices(request, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		services = []*serviced.RunningService{}
	}
	w.WriteJson(&services)
}

func RestKillRunning(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	serviceStateId, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	glog.Infof("Received request to kill %s", serviceStateId)
	if err != nil {
		RestBadRequest(w)
		return
	}
	var unused int
	err = client.StopRunningInstance(serviceStateId, &unused)
	if err != nil {
		glog.Errorf("Unable to stop service: %v", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&SimpleResponse{"Marked for death", servicesLink()})
}

func RestGetTopServices(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var allServices []*serviced.Service
	topServices := []*serviced.Service{}

	request := serviced.EntityRequest{}
	err := client.GetServices(request, &allServices)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	for _, service := range allServices {
		if len(service.ParentServiceId) == 0 {
			topServices = append(topServices, service)
		}
	}
	w.WriteJson(&topServices)
}

func RestGetService(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	var allServices []*serviced.Service

	request := serviced.EntityRequest{}
	if err := client.GetServices(request, &allServices); err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}

	sid, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	for _, service := range allServices {
		if service.Id == sid {
			w.WriteJson(&service)
			return
		}
	}

	glog.Errorf("No such service [%v]", sid)
	RestServerError(w)
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
	service.Instances = payload.Instances
	service.ParentServiceId = payload.ParentServiceId
	service.DesiredState = payload.DesiredState
	service.Launch = payload.Launch

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
	glog.Infof("Received update request for %s", serviceId)
	if err != nil {
		RestBadRequest(w)
		return
	}
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
		RestBadRequest(w)
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

func RestGetServiceLogs(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var logs string
	err = client.GetServiceLogs(serviceId, &logs)
	if err != nil {
		glog.Errorf("Unexpected error getting logs: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{logs, servicesLink()})
}

func RestGetServiceStateLogs(w *rest.ResponseWriter, r *rest.Request, client *clientlib.ControlClient) {
	serviceStateId, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var logs string
	err = client.GetServiceStateLogs(serviceStateId, &logs)
	if err != nil {
		glog.Errorf("Unexpected error getting logs: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{logs, servicesLink()})
}

func init() {
	configuration = ServiceConfig{}
	configDefaults(&configuration)
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
