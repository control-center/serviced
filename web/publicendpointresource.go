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
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type errInvalidVhostUsedCCHostname struct{}

func (e errInvalidVhostUsedCCHostname) Error() string {
	return "cannot add a vhost using the Control Center host name"
}

// json payload object for adding/removing/enabling a public endpoint
// with a service. other properties are retrieved from the url
type endpointRequest struct {
	ServiceName string
	UseTLS      bool
	Protocol    string
	IsEnabled   bool
}

// restAddVirtualHost parses payload, adds the vhost to the service, then updates the service
func restAddVirtualHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	glog.V(1).Infof("Add VHost with %s %#v", r.URL.Path, r)

	serviceid, application, vhostname, err := getVHostContext(r)
	if err != nil {
		restServerError(w, err)
		return
	}

	// Disallow adding a vhost with the same name as the hostname being used
	// to view CC (do not lock them out of the CC ui from this hostname)
	if strings.Compare(strings.ToLower(r.Host), strings.ToLower(vhostname)) == 0 {
		restServerError(w, errInvalidVhostUsedCCHostname{})
		return
	}

	var request endpointRequest
	err = r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	_, err = facade.AddPublicEndpointVHost(dataCtx, serviceid, application, vhostname, true, true)
	if err != nil {
		glog.Errorf("Error adding vhost to service (%s): %v", request.ServiceName, err)
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("VHost (%s) added to service (%s)", vhostname, request.ServiceName)
	restSuccess(w)
}

// restRemoveVirtualHost removes a vhost name from provided service and endpoint. Parameters are defined in path.
func restRemoveVirtualHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	serviceid, application, vhost, err := getVHostContext(r)
	if err != nil {
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("Removing vhost %s from service (%s)", vhost, serviceid)

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	err = facade.RemovePublicEndpointVHost(dataCtx, serviceid, application, vhost)
	if err != nil {
		glog.Error(err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}

// return serviceID, application and vhostname from the URL path
func getVHostContext(r *rest.Request) (string, string, string, error) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Failed getting serviceID: %v", err)
		return "", "", "", err
	}

	application, err := url.QueryUnescape(r.PathParam("application"))
	if err != nil {
		glog.Errorf("Failed getting application: %v", err)
		return "", "", "", err
	}

	vhostname, err := url.QueryUnescape(r.PathParam("name"))
	if err != nil {
		glog.Errorf("Failed getting hostname: %v", err)
		return "", "", "", err
	}
	return serviceID, application, vhostname, nil

}

// restVirtualHostEnable enables or disables a virtual host endpoint
func restVirtualHostEnable(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	glog.V(1).Infof("Enable/Disable VHOST with %s %#v", r.URL.Path, r)

	serviceid, application, vhostname, err := getVHostContext(r)
	if err != nil {
		restServerError(w, err)
		return
	}

	var request endpointRequest
	err = r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	err = facade.EnablePublicEndpointVHost(dataCtx, serviceid, application, vhostname, request.IsEnabled)
	if err != nil {
		glog.Errorf("Error enabling/disabling vhost %s on service (%s): %v", vhostname, request.ServiceName, err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}

// Returns the service, application, and portnumber from the request
func getPortContext(r *rest.Request) (string, string, string, error) {
	glog.V(1).Infof("in getPortContext()")

	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Failed getting serviceID: %v", err)
		return "", "", "", err
	}

	application, err := url.QueryUnescape(r.PathParam("application"))
	if err != nil {
		glog.Errorf("Failed getting application: %v", err)
		return "", "", "", err
	}

	// Validate the port number
	portName, err := url.QueryUnescape(r.PathParam("portname"))
	if err != nil {
		err := fmt.Errorf("Failed getting port name for service (%s): %v", serviceID, err)
		return "", "", "", err
	}

	// Validate the port number
	_, err = strconv.Atoi(strings.Split(portName, ":")[1])
	if err != nil {
		err := fmt.Errorf("Port must be a number between 1 and 65536")
		return "", "", "", err
	}

	return serviceID, application, portName, nil
}

// restAddPort parses payload, adds the port to the service, then updates the service
func restAddPort(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	glog.V(1).Infof("Add PORT with %s %#v", r.URL.Path, r)

	serviceid, application, port, err := getPortContext(r)
	if err != nil {
		err := fmt.Errorf("Error removing port from service (%s): %v", serviceid, err)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	var request endpointRequest
	err = r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	glog.V(0).Infof("Add PORT with %s %#v", r.URL.Path, r)

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	_, err = facade.AddPublicEndpointPort(dataCtx, serviceid, application,
		port, request.UseTLS, request.Protocol, true, true)
	if err != nil {
		glog.Errorf("Error adding port to service (%s): %v", request.ServiceName, err)
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("Port (%s) added to service (%s), UseTLS=%t, protocol=%s",
		port, request.ServiceName, request.UseTLS, request.Protocol)

	restSuccess(w)
}

func restRemovePort(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	glog.V(1).Infof("Remove PORT with %s %#v", r.URL.Path, r)

	serviceid, application, port, err := getPortContext(r)
	if err != nil {
		err := fmt.Errorf("Error removing port from service (%s): %v", serviceid, err)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("Removing port %s from service (%s)", port, serviceid)

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	err = facade.RemovePublicEndpointPort(dataCtx, serviceid, application, port)
	if err != nil {
		glog.Error(err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}

func restPortEnable(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	glog.V(1).Infof("Enable/Disable PORT with %s %#v", r.URL.Path, r)

	serviceid, application, port, err := getPortContext(r)
	if err != nil {
		err := fmt.Errorf("Error removing port from service (%s): %v", serviceid, err)
		glog.Error(err)
		restServerError(w, err)
		return
	}

	var request endpointRequest
	err = r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	err = facade.EnablePublicEndpointPort(dataCtx, serviceid, application,
		port, request.IsEnabled)
	if err != nil {
		glog.Errorf("Error setting enabled=%t for service (%s) port %s: %v", request.IsEnabled,
			request.ServiceName, port, err)
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("Port (%s) enabled=%t set for service (%s)", port, request.IsEnabled,
		request.ServiceName)

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
