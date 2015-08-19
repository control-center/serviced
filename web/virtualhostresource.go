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
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/node"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

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
func restAddVirtualHost(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var request virtualHostRequest
	err := r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	var services []service.Service
	var serviceRequest dao.ServiceRequest
	if err := client.GetServices(serviceRequest, &services); err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
		return
	}

	var service *service.Service
	for _, _service := range services {
		if _service.ID == request.ServiceID {
			service = &_service
			break
		}
	}

	if service == nil {
		glog.Errorf("Could not find service: %s", request.ServiceID)
		restServerError(w, err)
		return
	}

	//checkout other virtual hosts for redundancy
	_vhost := strings.ToLower(request.VirtualHostName)
	for _, service := range services {
		if service.Endpoints == nil {
			continue
		}

		for _, endpoint := range service.Endpoints {
			for _, host := range endpoint.VHostList {
				if host.Name == _vhost {
					glog.Errorf("vhost %s already defined for service: %s", request.VirtualHostName, service.ID)
					restServerError(w, err)
					return
				}
			}
		}
	}

	err = service.AddVirtualHost(request.Application, request.VirtualHostName)
	if err != nil {
		glog.Errorf("Unexpected error adding vhost to service (%s): %v", service.Name, err)
		restServerError(w, err)
		return
	}

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Unexpected error adding vhost to service (%s): %v", service.Name, err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}

// restRemoveVirtualHost removes a vhost name from provided service and endpoint. Parameters are defined in path.
func restRemoveVirtualHost(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {

	service, application, vhostname, err := getVHostContext(r, client)
	if err != nil {
		restServerError(w, err)
		return
	}

	err = service.RemoveVirtualHost(application, vhostname)
	if err != nil {
		glog.Errorf("Unexpected error removing vhost, %s, from service (%s): %v", vhostname, service.ID, err)
		restServerError(w, err)
		return
	}

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Unexpected error removing vhost, %s, from service (%s): %v", vhostname, service.ID, err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}


// return serviceID, application and vhostname from the URL path
func getVHostContext(r *rest.Request,  client *node.ControlClient) (*service.Service, string, string, error){
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Failed getting serviceID: %v", err)
		return nil, "", "", err
	}

	var service service.Service
	err = client.GetService(serviceID, &service)
	if err != nil {
		glog.Errorf("Unexpected error getting service (%s): %v", serviceID, err)
		return nil, "", "", err
	}

	application, err := url.QueryUnescape(r.PathParam("application"))
	if err != nil {
		glog.Errorf("Failed getting application: %v", err)
		return nil, "", "", err
	}

	vhostname, err := url.QueryUnescape(r.PathParam("name"))
	if err != nil {
		glog.Errorf("Failed getting hostname: %v", err)
		return nil, "", "", err
	}
	return &service, application, vhostname, nil

}

// json object for enabling/disabling a virtual host
type virtualHostEnable struct {
	Enable bool
}

// restVirtualHostEnable enables or disables a virtual host endpoint
func restVirtualHostEnable(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
   	glog.V(0).Infof("vhost enable %s", r.URL.Path)
	var request virtualHostEnable
	err := r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}
	glog.Infof("Enable VHOST with %s %#v", r.URL.Path, request)

	service, application, vhostname, err := getVHostContext(r, client)
	if err != nil {
		restServerError(w, err)
		return
	}
	glog.Infof("Enable VHOST request : %s, %s, %s", service.ID, application, vhostname)
	err = service.EnableVirtualHost(application, vhostname, request.Enable)
	if err != nil {
		glog.Errorf("Unexpected error enabling/disabling vhost %s on service (%s): %v", vhostname, service.Name, err)
		restServerError(w, err)
		return
	}

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Unexpected error updating  vhost %s on service (%s): %v", vhostname, service.Name, err)
		restServerError(w, err)
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
	Enabled         bool
}

// restGetVirtualHosts gets all services, then extracts all vhost information and returns it.
func restGetVirtualHosts(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var services []service.Service
	var serviceRequest dao.ServiceRequest
	err := client.GetServices(serviceRequest, &services)
	if err != nil {
		glog.Errorf("Unexpected error retrieving virtual hosts: %v", err)
		restServerError(w, err)
		return
	}

	serviceTree := make(map[string]service.Service)
	for _, service := range services {
		serviceTree[service.ID] = service
	}

	vhosts := make([]virtualHost, 0)
	for _, service := range services {
		if service.Endpoints == nil {
			continue
		}

		for _, endpoint := range service.Endpoints {
			if len(endpoint.VHostList) > 0 {
				parent, _ := serviceTree[service.ParentServiceID]
				for ; len(parent.ParentServiceID) != 0; parent, _ = serviceTree[parent.ParentServiceID] {
				}

				for _, vhost := range endpoint.VHostList {
					vh := virtualHost{
						Name:            vhost.Name,
						Application:     parent.Name,
						ServiceName:     service.Name,
						ServiceEndpoint: endpoint.Application,
						Enabled:         vhost.Enabled,
					}
					vhosts = append(vhosts, vh)
				}
			}
		}
	}

	w.WriteJson(&vhosts)
}
