package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/domain/service"

	"net/url"
	"strings"
)

// json object for adding/removing a virtual host with a service
type virtualHostRequest struct {
	ServiceID       string
	Application     string
	VirtualHostName string
}

// restAddVirtualHost parses payload, adds the vhost to the service, then updates the service
func restAddVirtualHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var request virtualHostRequest
	err := r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w)
		return
	}

	var services []*service.Service
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w)
		return
	}

	var service *service.Service
	for _, _service := range services {
		if _service.Id == request.ServiceID {
			service = _service
		}
	}

	if service == nil {
		glog.Errorf("Could not find service: %s", services)
		restServerError(w)
		return
	}

	//checkout other virtual hosts for redundancy
	_vhost := strings.ToLower(request.VirtualHostName)
	for _, service := range services {
		if service.Endpoints == nil {
			continue
		}

		for _, endpoint := range service.Endpoints {
			for _, host := range endpoint.VHosts {
				if host == _vhost {
					glog.Errorf("vhost %s already defined for service: %s", request.VirtualHostName, service.Id)
					restServerError(w)
					return
				}
			}
		}
	}

	err = service.AddVirtualHost(request.Application, request.VirtualHostName)
	if err != nil {
		glog.Errorf("Unexpected error adding vhost to service (%s): %v", service.Name, err)
		restServerError(w)
		return
	}

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Unexpected error adding vhost to service (%s): %v", service.Name, err)
		restServerError(w)
		return
	}

	restSuccess(w)
}

// restRemoveVirtualHost removes a vhost name from provided service and endpoint. Parameters are defined in path.
func restRemoveVirtualHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Failed getting serviceId: %v", err)
		restBadRequest(w)
		return
	}
	application, err := url.QueryUnescape(r.PathParam("application"))
	if err != nil {
		glog.Errorf("Failed getting application: %v", err)
		restBadRequest(w)
		return
	}

	hostname, err := url.QueryUnescape(r.PathParam("name"))
	if err != nil {
		glog.Errorf("Failed getting hostname: %v", err)
		restBadRequest(w)
		return
	}

	var service service.Service
	err = client.GetService(serviceID, &service)
	if err != nil {
		glog.Errorf("Unexpected error getting service (%s): %v", serviceID, err)
		restServerError(w)
		return
	}

	err = service.RemoveVirtualHost(application, hostname)
	if err != nil {
		glog.Errorf("Unexpected error removing vhost, %s, from service (%s): %v", hostname, serviceID, err)
		restServerError(w)
		return
	}

	var unused int
	err = client.UpdateService(service, &unused)
	if err != nil {
		glog.Errorf("Unexpected error removing vhost, %s, from service (%s): %v", hostname, serviceID, err)
		restServerError(w)
		return
	}

	restSuccess(w)
}

// Get all virtual hosts
type virtualHost struct {
	Name            string
	Application     string
	ServiceName     string
	ServiceEndpoint string
}

// restGetVirtualHosts gets all services, then extracts all vhost information and returns it.
func restGetVirtualHosts(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var services []*service.Service
	err := client.GetServices(&empty, &services)
	if err != nil {
		glog.Errorf("Unexpected error retrieving virtual hosts: %v", err)
		restServerError(w)
		return
	}

	serviceTree := make(map[string]*service.Service)
	for _, service := range services {
		serviceTree[service.Id] = service
	}

	vhosts := make([]virtualHost, 0)
	for _, service := range services {
		if service.Endpoints == nil {
			continue
		}

		for _, endpoint := range service.Endpoints {
			if len(endpoint.VHosts) > 0 {
				parent, _ := serviceTree[service.ParentServiceID]
				for ; len(parent.ParentServiceID) != 0; parent, _ = serviceTree[parent.ParentServiceID] {
				}

				for _, vhost := range endpoint.VHosts {
					vh := virtualHost{
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
