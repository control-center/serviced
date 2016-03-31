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
	"time"

	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/node"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

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

	request := addressassignment.AssignmentRequest{ServiceID: serviceID, IPAddress: "", AutoAssignment: true}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Errorf("Failed to automatically assign IPs: %+v -> %v", request, err)
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

	request := addressassignment.AssignmentRequest{ServiceID: serviceID, IPAddress: ip, AutoAssignment: false}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Errorf("Failed to manually assign IP: %+v -> %v", request, err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}

// restGetServicesHealth returns health checks for all services
func restGetServicesHealth(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	stats := make(map[string]map[int]map[string]health.HealthStatus)
	if err := client.GetServicesHealth(0, &stats); err != nil {
		glog.Errorf("Could not get services health: %s", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(struct {
		Timestamp int64
		Statuses  map[string]map[int]map[string]health.HealthStatus
	}{
		Timestamp: time.Now().UTC().Unix(),
		Statuses:  stats,
	})
}
