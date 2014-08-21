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
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/dao"

	"net/url"
)

// restServiceAutomaticAssignIP rest resource for automatic assigning ips to a service
func restServiceAutomaticAssignIP(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Could not get serviceId: %v", err)
		restBadRequest(w, err)
		return
	}

	request := dao.AssignmentRequest{ServiceID: serviceID, IPAddress: "", AutoAssignment: true}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Error("Failed to automatically assign IPs: %+v -> %v", request, err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}

// restServiceManualAssignIP rest resource for manual assigning ips to a service
func restServiceManualAssignIP(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Could not get serviceId: %v", err)
		restBadRequest(w, err)
		return
	}

	ip, err := url.QueryUnescape(r.PathParam("ip"))
	if err != nil {
		glog.Errorf("Could not get serviceId: %v", err)
		restBadRequest(w, err)
		return
	}

	request := dao.AssignmentRequest{ServiceID: serviceID, IPAddress: ip, AutoAssignment: false}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Error("Failed to manually assign IP: %+v -> %v", request, err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}
