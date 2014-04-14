package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"

	"net/url"
	"regexp"
	"strings"
	"time"
)

var empty interface{}

type HandlerFunc func(w *rest.ResponseWriter, r *rest.Request)
type HandlerClientFunc func(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient)

func RestGetAppTemplates(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var unused int
	var templatesMap map[string]*dao.ServiceTemplate
	client.GetServiceTemplates(unused, &templatesMap)
	w.WriteJson(&templatesMap)
}

func RestDeployAppTemplate(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var payload dao.ServiceTemplateDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode deployment payload: ", err)
		RestBadRequest(w)
		return
	}
	var tenantId string
	err = client.DeployTemplate(payload, &tenantId)
	if err != nil {
		glog.Error("Could not deploy template: ", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Deployed template ", payload)

	// TODO: the UI needs a way to disable that automatic IP assignment (see CmdDeployTemplate)
	assignmentRequest := dao.AssignmentRequest{tenantId, "", true}
	if err := client.AssignIPs(assignmentRequest, nil); err != nil {
		glog.Error("Could not automatically assign IPs: %v", err)
		return
	}

	glog.Infof("Automatically assigned IP addresses to service: %v", tenantId)
	// end of automatic IP assignment

	w.WriteJson(&SimpleResponse{tenantId, servicesLinks()})
}

func RestGetPools(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var poolsMap map[string]*dao.ResourcePool
	err := client.GetResourcePools(&empty, &poolsMap)
	if err != nil {
		glog.Error("Could not get resource pools: ", err)
		RestServerError(w)
		return
	}
	w.WriteJson(&poolsMap)
}

func RestAddPool(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var payload dao.ResourcePool
	var poolId string
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode pool payload: ", err)
		RestBadRequest(w)
		return
	}
	err = client.AddResourcePool(payload, &poolId)
	if err != nil {
		glog.Error("Unable to add pool: ", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Added pool ", poolId)
	w.WriteJson(&SimpleResponse{"Added resource pool", poolLinks(poolId)})
}

func RestUpdatePool(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	poolId, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var payload dao.ResourcePool
	var unused int
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode pool payload: ", err)
		RestBadRequest(w)
		return
	}
	err = client.UpdateResourcePool(payload, &unused)
	if err != nil {
		glog.Error("Unable to update pool: ", err)
		RestServerError(w)
		return
	}
	glog.V(1).Info("Updated pool ", poolId)
	w.WriteJson(&SimpleResponse{"Updated resource pool", poolLinks(poolId)})
}

func RestRemovePool(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	poolId, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var unused int
	err = client.RemoveResourcePool(poolId, &unused)
	if err != nil {
		glog.Error("Could not remove resource pool: ", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Removed pool ", poolId)
	w.WriteJson(&SimpleResponse{"Removed resource pool", poolsLinks()})
}

func RestGetHosts(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var hosts map[string]*dao.Host
	err := client.GetHosts(&empty, &hosts)
	if err != nil {
		glog.Errorf("Could not get hosts: %v", err)
		RestServerError(w)
		return
	}
	glog.V(2).Infof("Returning %d hosts", len(hosts))
	w.WriteJson(&hosts)
}

func filterByNameRegex(nmregex string, services []*dao.Service) ([]*dao.Service, error) {
	r, err := regexp.Compile(nmregex)
	if err != nil {
		glog.Errorf("Bad name regexp :%s", nmregex)
		return nil, err
	}

	matches := []*dao.Service{}
	for _, service := range services {
		if r.MatchString(service.Name) {
			matches = append(matches, service)
		}
	}
	glog.V(2).Infof("Returning %d named services", len(matches))
	return matches, nil
}

func getTaggedServices(client *serviced.ControlClient, tags, nmregex string) ([]*dao.Service, error) {
	services := []*dao.Service{}
	var ts interface{}
	ts = strings.Split(tags, ",")
	if err := client.GetTaggedServices(&ts, &services); err != nil {
		glog.Errorf("Could not get tagged services: %v", err)
		return nil, err
	}

	if nmregex != "" {
		return filterByNameRegex(nmregex, services)
	}
	glog.V(2).Infof("Returning %d tagged services", len(services))
	return services, nil
}

func getNamedServices(client *serviced.ControlClient, nmregex string) ([]*dao.Service, error) {
	services := []*dao.Service{}
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get named services: %v", err)
		return nil, err
	}

	return filterByNameRegex(nmregex, services)
}

func getServices(client *serviced.ControlClient) ([]*dao.Service, error) {
	services := []*dao.Service{}
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get services: %v", err)
		return nil, err
	}

	glog.V(2).Infof("Returning %d services", len(services))
	return services, nil
}

func RestGetAllServices(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	if tags := r.URL.Query().Get("tags"); tags != "" {
		nmregex := r.URL.Query().Get("name")
		result, err := getTaggedServices(client, tags, nmregex)
		if err != nil {
			RestServerError(w)
			return
		}

		w.WriteJson(&result)
		return
	}

	if nmregex := r.URL.Query().Get("name"); nmregex != "" {
		result, err := getNamedServices(client, nmregex)
		if err != nil {
			RestServerError(w)
			return
		}

		w.WriteJson(&result)
		return
	}

	result, err := getServices(client)
	if err != nil {
		RestServerError(w)
		return
	}

	w.WriteJson(&result)
}

func RestGetRunningForHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	hostId, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var services []*dao.RunningService
	err = client.GetRunningServicesForHost(hostId, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		glog.V(3).Info("Running services was nil, returning empty list instead")
		services = []*dao.RunningService{}
	}
	glog.V(2).Infof("Returning %d running services for host %s", len(services), hostId)
	w.WriteJson(&services)
}

func RestGetRunningForService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var services []*dao.RunningService
	err = client.GetRunningServicesForService(serviceId, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		glog.V(3).Info("Running services was nil, returning empty list instead")
		services = []*dao.RunningService{}
	}
	glog.V(2).Infof("Returning %d running services for service %s", len(services), serviceId)
	w.WriteJson(&services)
}

func RestGetAllRunning(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var services []*dao.RunningService
	err := client.GetRunningServices(&empty, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		glog.V(3).Info("Services was nil, returning empty list instead")
		services = []*dao.RunningService{}
	}
	glog.V(2).Infof("Return %d running services", len(services))
	w.WriteJson(&services)
}

func RestKillRunning(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceStateId, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	hostId, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	request := dao.HostServiceRequest{hostId, serviceStateId}
	glog.V(1).Info("Received request to kill ", request)

	var unused int
	err = client.StopRunningInstance(request, &unused)
	if err != nil {
		glog.Errorf("Unable to stop service: %v", err)
		RestServerError(w)
		return
	}

	w.WriteJson(&SimpleResponse{"Marked for death", servicesLinks()})
}

func RestGetTopServices(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var allServices []*dao.Service
	topServices := []*dao.Service{}

	err := client.GetServices(&empty, &allServices)
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
	glog.V(2).Infof("Returning %d services as top services", len(topServices))
	w.WriteJson(&topServices)
}

func RestGetService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var allServices []*dao.Service

	if err := client.GetServices(&empty, &allServices); err != nil {
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

func RestAddService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var payload dao.Service
	var serviceId string
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode service payload: ", err)
		RestBadRequest(w)
		return
	}
	service, err := dao.NewService()
	if err != nil {
		glog.Errorf("Could not create service: %v", err)
		RestServerError(w)
		return
	}
	now := time.Now()
	service.Name = payload.Name
	service.Description = payload.Description
	service.Context = payload.Context
	service.Tags = payload.Tags
	service.PoolId = payload.PoolId
	service.ImageId = payload.ImageId
	service.Startup = payload.Startup
	service.Instances = payload.Instances
	service.ParentServiceId = payload.ParentServiceId
	service.DesiredState = payload.DesiredState
	service.Launch = payload.Launch
	service.Endpoints = payload.Endpoints
	service.ConfigFiles = payload.ConfigFiles
	service.Volumes = payload.Volumes
	service.CreatedAt = now
	service.UpdatedAt = now

	//for each endpoint, evaluate it's Application
	if err = service.EvaluateEndpointTemplates(client); err != nil {
		glog.Errorf("Unable to evaluate service endpoints: %v", err)
		RestServerError(w)
		return
	}

	//add the service to the data store
	err = client.AddService(*service, &serviceId)
	if err != nil {
		glog.Errorf("Unable to add service: %v", err)
		RestServerError(w)
		return
	}

	//deploy the service, in other words start it
	var unused int
	sduuid, _ := dao.NewUuid()
	deployment := dao.ServiceDeployment{sduuid, "", service.Id, now}
	err = client.AddServiceDeployment(deployment, &unused)
	if err != nil {
		glog.Errorf("Unable to add service deployment: %v", err)
		RestServerError(w)
		return
	}

	glog.V(0).Info("Added service ", serviceId)
	w.WriteJson(&SimpleResponse{"Added service", serviceLinks(serviceId)})
}

func RestUpdateService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	glog.V(3).Infof("Received update request for %s", serviceId)
	if err != nil {
		RestBadRequest(w)
		return
	}
	var payload dao.Service
	var unused int
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode service payload: ", err)
		RestBadRequest(w)
		return
	}
	err = client.UpdateService(payload, &unused)
	if err != nil {
		glog.Errorf("Unable to update service %s: %v", serviceId, err)
		RestServerError(w)
		return
	}
	glog.V(1).Info("Updated service ", serviceId)
	w.WriteJson(&SimpleResponse{"Updated service", serviceLinks(serviceId)})
}

func RestRemoveService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
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
	glog.V(0).Info("Removed service ", serviceId)
	w.WriteJson(&SimpleResponse{"Removed service", servicesLinks()})
}

func RestGetHostsForResourcePool(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var poolHosts []*dao.PoolHost
	poolId, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		glog.V(1).Infof("Unable to acquire pool ID: %v", err)
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
		glog.V(3).Info("Pool hosts was nil, returning empty list instead")
		poolHosts = []*dao.PoolHost{}
	}
	glog.V(2).Infof("Returning %d hosts for pool %s", len(poolHosts), poolId)
	w.WriteJson(&poolHosts)
}

func RestAddHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var payload dao.Host
	var hostId string
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Infof("Could not decode host payload: %v", err)
		RestBadRequest(w)
		return
	}
	// Save the pool ID and IP address for later. GetInfo wipes these
	pool := payload.PoolId
	ipAddr := payload.IpAddr
	remoteClient, err := serviced.NewAgentClient(payload.IpAddr)
	if err != nil {
		glog.Errorf("Could not create connection to host %s: %v", payload.IpAddr, err)
		RestServerError(w)
		return
	}

	//TODO: get user supplied IPs from UI
	err = remoteClient.GetInfo([]string{}, &payload)
	if err != nil {
		glog.Errorf("Unable to get remote host info: %v", err)
		RestBadRequest(w)
		return
	}
	// Reset the pool ID and IP address
	payload.PoolId = pool
	parts := strings.Split(ipAddr, ":")
	payload.IpAddr = parts[0]

	err = client.AddHost(payload, &hostId)
	if err != nil {
		glog.Errorf("Unable to add host: %v", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Added host ", hostId)
	w.WriteJson(&SimpleResponse{"Added host", hostLinks(hostId)})
}

func RestUpdateHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	hostId, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	glog.V(3).Infof("Received update request for %s", hostId)
	var payload dao.Host
	var unused int
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Infof("Could not decode host payload: %v", err)
		RestBadRequest(w)
		return
	}
	err = client.UpdateHost(payload, &unused)
	if err != nil {
		glog.Errorf("Unable to update host: %v", err)
		RestServerError(w)
		return
	}
	glog.V(1).Info("Updated host ", hostId)
	w.WriteJson(&SimpleResponse{"Updated host", hostLinks(hostId)})
}

func RestRemoveHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
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
	glog.V(0).Info("Removed host ", hostId)
	w.WriteJson(&SimpleResponse{"Removed host", hostsLinks()})
}

func RestGetServiceLogs(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var logs string
	err = client.GetServiceLogs(serviceId, &logs)
	if err != nil {
		glog.Errorf("Unexpected error getting service logs: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{logs, serviceLinks(serviceId)})
}

// RestStartService starts the service with the given id and all of its children
func RestStartService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var i string
	err = client.StartService(serviceId, &i)
	if err != nil {
		glog.Errorf("Unexpected error starting service: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{"Started service", serviceLinks(serviceId)})
}

// RestStopService stop the service with the given id and all of its children
func RestStopService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var i int
	err = client.StopService(serviceId, &i)
	if err != nil {
		glog.Errorf("Unexpected error stopping service: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{"Stopped service", serviceLinks(serviceId)})
}

func RestSnapshotService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var label string
	err = client.Snapshot(serviceId, &label)
	if err != nil {
		glog.Errorf("Unexpected error snapshotting service: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{label, serviceLinks(serviceId)})
}

func RestGetRunningService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceStateId, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	request := dao.ServiceStateRequest{serviceId, serviceStateId}

	var running dao.RunningService
	err = client.GetRunningService(request, &running)
	if err != nil {
		glog.Errorf("Unexpected error retrieving services: %v", err)
		RestServerError(w)
	}
	w.WriteJson(running)
}

func RestGetServiceStateLogs(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceStateId, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	request := dao.ServiceStateRequest{serviceId, serviceStateId}

	var logs string
	err = client.GetServiceStateLogs(request, &logs)
	if err != nil {
		glog.Errorf("Unexpected error getting service state logs: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{logs, servicesLinks()})
}

type VirtualHost struct {
	Name            string
	Application     string
	ServiceName     string
	ServiceEndpoint string
}

func RestAddVirtualHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	application, err := url.QueryUnescape(r.PathParam("application"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	vhostName, err := url.QueryUnescape(r.PathParam("vhostName"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	var services []*dao.Service
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}

	var service *dao.Service
	for _, _service := range services {
		if _service.Id == serviceId {
			service = _service
		}
	}

	if service == nil {
		glog.Errorf("Could not find service: %s", services)
		RestServerError(w)
		return
	}

	//checkout other virtual hosts for redundancy
	_vhost := strings.ToLower(vhostName)
	for _, service := range services {
		if service.Endpoints == nil {
			continue
		}

		for _, endpoint := range service.Endpoints {
			for _, host := range endpoint.VHosts {
				if host == _vhost {
					glog.Errorf("vhost %s already defined for service: %s", vhostName, service.Id)
					RestServerError(w)
					return
				}
			}
		}
	}

	err = service.AddVirtualHost(application, vhostName)
	if err != nil {
		glog.Errorf("Unexpected error adding vhost to service (%s): %v", service.Name, err)
		RestServerError(w)
		return
	}

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Unexpected error adding vhost to service (%s): %v", service.Name, err)
		RestServerError(w)
		return
	}
}

// Remove a virtual hosts for provided service, endpoint, and vhost name, parameters are defined in path
func RestRemoveVirtualHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	application, err := url.QueryUnescape(r.PathParam("application"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	vhostName, err := url.QueryUnescape(r.PathParam("vhostName"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	var service dao.Service
	err = client.GetService(serviceId, &service)
	if err != nil {
		glog.Errorf("Unexpected error getting service (%s): %v", serviceId, err)
		RestServerError(w)
		return
	}

	err = service.RemoveVirtualHost(application, vhostName)
	if err != nil {
		glog.Errorf("Unexpected error removing vhost, %s, from service (%s): %v", vhostName, serviceId, err)
		RestServerError(w)
		return
	}

	var unused int
	err = client.UpdateService(service, &unused)
	if err != nil {
		glog.Errorf("Unexpected error removing vhost, %s, from service (%s): %v", vhostName, serviceId, err)
		RestServerError(w)
		return
	}
}

// Get all virtual hosts
func RestGetVirtualHosts(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var services []*dao.Service
	err := client.GetServices(&empty, &services)
	if err != nil {
		glog.Errorf("Unexpected error retrieving virtual hosts: %v", err)
		RestServerError(w)
		return
	}

	service_tree := make(map[string]*dao.Service)
	for _, service := range services {
		service_tree[service.Id] = service
	}

	var vhosts []VirtualHost = make([]VirtualHost, 0)
	for _, service := range services {
		if service.Endpoints == nil {
			continue
		}

		for _, endpoint := range service.Endpoints {
			if len(endpoint.VHosts) > 0 {
				parent, _ := service_tree[service.ParentServiceId]
				for ; len(parent.ParentServiceId) != 0; parent, _ = service_tree[parent.ParentServiceId] {
				}

				for _, vhost := range endpoint.VHosts {
					vh := VirtualHost{
						Name:            vhost,
						Application:     parent.Name,
						ServiceName:     service.Name,
						ServiceEndpoint: endpoint.Application,
					}
					vhosts = append(vhosts, vh)
				}
			}
		}
	}

	w.WriteJson(&vhosts)
}
