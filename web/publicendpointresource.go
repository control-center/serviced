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
//
// Rest methods for VHost and Port public endpoints
//

package web

import (
	"github.com/control-center/serviced/dao"
	svc "github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/node"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

	"fmt"
	"net"
	"net/url"
	"time"
	"strconv"
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

	var services []svc.Service
	var serviceRequest dao.ServiceRequest
	if err := client.GetServices(serviceRequest, &services); err != nil {
		err := fmt.Errorf("Could not get services: %v", err)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	var service *svc.Service
	for _, _service := range services {
		if _service.ID == request.ServiceID {
			service = &_service
			break
		}
	}

	if service == nil {
		err := fmt.Errorf("Could not find service: %s", request.ServiceID)
		glog.Error(err)
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
					err := fmt.Errorf("vhost %s already defined for service: %s", request.VirtualHostName, service.ID)
					glog.Error(err)
					restServerError(w, err)
					return
				}
			}
		}
	}

	err = service.AddVirtualHost(request.Application, request.VirtualHostName)
	if err != nil {
		err := fmt.Errorf("Error adding vhost to service (%s): %v", service.Name, err)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		err := fmt.Errorf("Error adding vhost to service (%s): %v", service.Name, err)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	// Restart the service if it is running
	if service.DesiredState == int(svc.SVCRun) || service.DesiredState == int(svc.SVCRestart) {
		if err = client.RestartService(dao.ScheduleServiceRequest{ServiceID: service.ID}, &unused); err != nil {
			glog.Errorf("Error restarting service %s: %s. Trying again in 10 seconds.", service.Name, err)
			time.Sleep(10 * time.Second)
			if err = client.RestartService(dao.ScheduleServiceRequest{ServiceID: service.ID}, &unused); err != nil {
				glog.Errorf("Error restarting service %s: %s. Aborting.", service.Name, err)
				err = fmt.Errorf("Error restarting service %s.  Service will need to be restarted manually.", service.Name)
				restServerError(w, err)
				return
			}
		}
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
		glog.Errorf("Error removing vhost, %s, from service (%s): %v", vhostname, service.Name, err)
		restServerError(w, err)
		return
	}

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Error removing vhost, %s, from service (%s): %v", vhostname, service.Name, err)
		restServerError(w, err)
		return
	}

	// Restart the service if it is running
	if service.DesiredState == int(svc.SVCRun) || service.DesiredState == int(svc.SVCRestart) {
		if err = client.RestartService(dao.ScheduleServiceRequest{ServiceID: service.ID}, &unused); err != nil {
			glog.Errorf("Error Starting service %s: %s", service.Name, err)
			err = fmt.Errorf("Error restarting service %s.  Service will need to be started manually.", service.Name)
			restServerError(w, err)
			return
		}
	}

	restSuccess(w)
}

// return serviceID, application and vhostname from the URL path
func getVHostContext(r *rest.Request, client *node.ControlClient) (*svc.Service, string, string, error) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Failed getting serviceID: %v", err)
		return nil, "", "", err
	}

	var service svc.Service
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
	var request virtualHostEnable
	err := r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}
	glog.V(1).Infof("Enable VHOST with %s %#v", r.URL.Path, request)

	service, application, vhostname, err := getVHostContext(r, client)
	if err != nil {
		restServerError(w, err)
		return
	}
	glog.V(1).Infof("Enable VHOST request : %s, %s, %s", service.ID, application, vhostname)
	err = service.EnableVirtualHost(application, vhostname, request.Enable)
	if err != nil {
		glog.Errorf("Error enabling/disabling vhost %s on service (%s): %v", vhostname, service.Name, err)
		restServerError(w, err)
		return
	}

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Error updating  vhost %s on service (%s): %v", vhostname, service.Name, err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}

// json object for adding/removing a port with a service
type portRequest struct {
	ServiceID   string
	Application string
	PortName    string
}

// Returns the service, application, and portnumber from the request
func getPortContext(r *rest.Request, client *node.ControlClient) (*svc.Service, string, string, error) {
	glog.V(1).Infof("in getPortContext()")

	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Failed getting serviceID: %v", err)
		return nil, "", "", err
	}

	var service svc.Service
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

	// Validate the port number
	portName, err := url.QueryUnescape(r.PathParam("portname"))
	if err != nil {
		err := fmt.Errorf("Failed getting port name for service (%s): %v", serviceID, err)
		return nil, "", "", err
	}

	// Validate the port number
	_, err = strconv.Atoi(strings.Split(portName, ":")[1])
	if err != nil {
		err := fmt.Errorf("Port must be a number greater than 1024 and less then 65536")
		return nil, "", "", err
	}

	return &service, application, portName, nil
}

// restAddPort parses payload, adds the port to the service, then updates the service
func restAddPort(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	glog.V(1).Infof("Add PORT with %s %#v", r.URL.Path, r)

	var request portRequest
	err := r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	// Validate the port number
	scrubbedPort := svc.ScrubPortString(request.PortName)
	portParts := strings.Split(scrubbedPort, ":")
	if len(portParts) <= 1 {
		err := fmt.Errorf("Invalid port address. Port address be \":[PORT NUMBER]\" or \"[IP ADDRESS]:[PORT NUMBER]\"")
		glog.Error(err)
		restServerError(w, err)
		return
	}
	port, err := strconv.Atoi(portParts[1])
	if err != nil {
		err := fmt.Errorf("Port must be a number greater than 1024 and less then 65536")
		glog.Error(err)
		restServerError(w, err)
		return
	}

	if port < 1024 || port > 65535 {
		err := fmt.Errorf("Port must be greater than 1024 and less then 65536")
		glog.Error(err)
		restServerError(w, err)
		return
	}

	switch port {
	case 5000:
		fallthrough
	case 8080:
		err := fmt.Errorf("Port %d is reserved", port)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	var services []svc.Service
	var serviceRequest dao.ServiceRequest
	if err := client.GetServices(serviceRequest, &services); err != nil {
		err := fmt.Errorf("Could not get services: %v", err)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	var service *svc.Service
	for _, _service := range services {
		if _service.ID == request.ServiceID {
			service = &_service
			break
		}
	}

	if service == nil {
		err := fmt.Errorf("Could not find service: %s", request.ServiceID)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	//checkout other ports for redundancy
	for _, service := range services {
		if service.Endpoints == nil {
			continue
		}

		for _, endpoint := range service.Endpoints {
			for _, epPort := range endpoint.PortList {
				if scrubbedPort == epPort.PortAddr {
					err := fmt.Errorf("Port %s already defined for service: %s", epPort.PortAddr, service.Name)
					glog.Error(err)
					restServerError(w, err)
					return
				}
			}
		}
	}

	// Check to make sure the port is available.  Don't allow adding a port if it's already being used.
	err = checkPort("tcp", fmt.Sprintf("%s", scrubbedPort))
	if err != nil {
		glog.Error(err)
		restServerError(w, err)
		return
	}

	err = service.AddPort(request.Application, scrubbedPort)
	if err != nil {
		glog.Errorf("Error adding port to service (%s): %v", service.Name, err)
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("Port (%s) added to service (%s)", request.PortName, service.Name)

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Unexpected error adding port to service (%s): %v", service.Name, err)
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("Service (%s) updated", service.Name)

	// Restart the service if it is running
	if service.DesiredState == int(svc.SVCRun) || service.DesiredState == int(svc.SVCRestart) {
		if err = client.RestartService(dao.ScheduleServiceRequest{ServiceID: service.ID}, &unused); err != nil {
			glog.Errorf("Error restarting service %s: %s. Trying again in 10 seconds.", service.Name, err)
			time.Sleep(10 * time.Second)
			if err = client.RestartService(dao.ScheduleServiceRequest{ServiceID: service.ID}, &unused); err != nil {
				glog.Errorf("Error restarting service %s: %s. Aborting.", service.Name, err)
				err = fmt.Errorf("Error restarting service %s.  Service will need to be restarted manually.", service.Name)
				restServerError(w, err)
				return
			}
		}
	}

	restSuccess(w)
}

func restRemovePort(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	glog.V(1).Infof("Remove PORT with %s %#v", r.URL.Path, r)

	service, application, port, err := getPortContext(r, client)
	if err != nil {
		err := fmt.Errorf("Error removing port from service (%s): %v", service.Name, err)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	glog.V(2).Info("Removing port %d from service (%s)", port, service.Name)

	err = service.RemovePort(application, port)
	if err != nil {
		glog.Errorf("Error removing port, %s, from service (%s): %v", port, service.Name, err)
		restServerError(w, err)
		return
	}

	glog.V(2).Info("Updating service (%s)", port, service.Name)

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Error removing port, %s, from service (%s): %v", port, service.Name, err)
		restServerError(w, err)
		return
	}

	glog.V(2).Info("Successfully removed port %s from service (%s)", port, service.Name)

	// Restart the service if it is running
	if service.DesiredState == int(svc.SVCRun) || service.DesiredState == int(svc.SVCRestart) {
		if err = client.RestartService(dao.ScheduleServiceRequest{ServiceID: service.ID}, &unused); err != nil {
			glog.Errorf("Error Restarting service %s: %s", service.Name, err)
			err = fmt.Errorf("Error restarting service %s.  Service will need to be started manually.", service.Name)
			restServerError(w, err)
			return
		}
	}

	restSuccess(w)
}

// json object for enabling/disabling a port
type portEnable struct {
	Enable bool
}

// Try to open the port.  If the port opens, we're good. Otherwise return error.
func checkPort(network string, laddr string) error {
	glog.V(2).Infof("Checking %s port %s", network, laddr)
	listener, err := net.Listen(network, laddr)
	if err != nil {
		// Port isn't available.
		glog.V(2).Infof("Port Listen failed; something else is using this port.")
		return err
	} else {
		// Port was opened. Make sure we close it.
		glog.V(2).Infof("Port Listen succeeded. Closing the listener.")
		listener.Close()
	}
	return nil
}

func restPortEnable(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	glog.V(1).Infof("Enable/Disable PORT with %s %#v", r.URL.Path, r)

	var request portEnable
	err := r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	service, application, port, err := getPortContext(r, client)
	if err != nil {
		restServerError(w, err)
		return
	}

	// If they're trying to enable the port, check to make sure it's available.
	if request.Enable {
		err = checkPort("tcp", fmt.Sprintf("%s", port))
		if err != nil {
			restServerError(w, err)
			return
		}
	}

	err = service.EnablePort(application, port, request.Enable)
	if err != nil {
		glog.Errorf("Error enabling/disabling port %s on service (%s): %v", port, service.Name, err)
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("Port %d for service (%s) enabled", port, service.Name)

	var unused int
	err = client.UpdateService(*service, &unused)
	if err != nil {
		glog.Errorf("Error updating port %s on service (%s): %v", port, service.Name, err)
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("Service (%s) updated", service.Name)
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
	var services []svc.Service
	var serviceRequest dao.ServiceRequest
	err := client.GetServices(serviceRequest, &services)
	if err != nil {
		glog.Errorf("Unexpected error retrieving virtual hosts: %v", err)
		restServerError(w, err)
		return
	}

	serviceTree := make(map[string]svc.Service)
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
