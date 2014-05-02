package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"

	"net/url"
	"strings"
)

// json object for adding/removing a virtual host with a service
type virtualHostRequest struct {
	ServiceID       string
	Application     string
	VirtualHostName string
}

// RestAddVirtualHost parses payload, adds the vhost to the service, then updates the service
func RestAddVirtualHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var request virtualHostRequest
	err := r.DecodeJsonPayload(&request)
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
		if _service.Id == request.ServiceID {
			service = _service
		}
	}

	if service == nil {
		glog.Errorf("Could not find service: %s", services)
		RestServerError(w)
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
					RestServerError(w)
					return
				}
			}
		}
	}

	err = service.AddVirtualHost(request.Application, request.VirtualHostName)
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

	RestSuccess(w)
}

// RestRemoveVirtualHost removes a vhost name from provided service and endpoint. Parameters are defined in path.
func RestRemoveVirtualHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Failed getting serviceId: %v", err)
		RestBadRequest(w)
		return
	}
	application, err := url.QueryUnescape(r.PathParam("application"))
	if err != nil {
		glog.Errorf("Failed getting application: %v", err)
		RestBadRequest(w)
		return
	}

	hostname, err := url.QueryUnescape(r.PathParam("name"))
	if err != nil {
		glog.Errorf("Failed getting hostname: %v", err)
		RestBadRequest(w)
		return
	}

	var service dao.Service
	err = client.GetService(serviceID, &service)
	if err != nil {
		glog.Errorf("Unexpected error getting service (%s): %v", serviceID, err)
		RestServerError(w)
		return
	}

	err = service.RemoveVirtualHost(application, hostname)
	if err != nil {
		glog.Errorf("Unexpected error removing vhost, %s, from service (%s): %v", hostname, serviceID, err)
		RestServerError(w)
		return
	}

	var unused int
	err = client.UpdateService(service, &unused)
	if err != nil {
		glog.Errorf("Unexpected error removing vhost, %s, from service (%s): %v", hostname, serviceID, err)
		RestServerError(w)
		return
	}

	RestSuccess(w)
}

// Get all virtual hosts
type virtualHost struct {
	Name            string
	Application     string
	ServiceName     string
	ServiceEndpoint string
}

// RestGetVirtualHosts gets all services, then extracts all vhost information and returns it.
func RestGetVirtualHosts(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var services []*dao.Service
	err := client.GetServices(&empty, &services)
	if err != nil {
		glog.Errorf("Unexpected error retrieving virtual hosts: %v", err)
		RestServerError(w)
		return
	}

	serviceTree := make(map[string]*dao.Service)
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
				parent, _ := serviceTree[service.ParentServiceId]
				for ; len(parent.ParentServiceId) != 0; parent, _ = serviceTree[parent.ParentServiceId] {
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
