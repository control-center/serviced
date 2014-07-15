// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced/node"
	"github.com/zenoss/serviced/dao"

	"net/url"
)

// restServiceAutomaticAssignIP rest resource for automatic assigning ips to a service
func restServiceAutomaticAssignIP(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Could not get serviceId: %v", err)
		restBadRequest(w)
		return
	}

	request := dao.AssignmentRequest{ServiceID: serviceID, IPAddress: "", AutoAssignment: true}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Error("Failed to automatically assign IPs: %+v -> %v", request, err)
		restServerError(w)
		return
	}

	restSuccess(w)
}

// restServiceManualAssignIP rest resource for manual assigning ips to a service
func restServiceManualAssignIP(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Could not get serviceId: %v", err)
		restBadRequest(w)
		return
	}

	ip, err := url.QueryUnescape(r.PathParam("ip"))
	if err != nil {
		glog.Errorf("Could not get serviceId: %v", err)
		restBadRequest(w)
		return
	}

	request := dao.AssignmentRequest{ServiceID: serviceID, IPAddress: ip, AutoAssignment: false}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Error("Failed to manually assign IP: %+v -> %v", request, err)
		restServerError(w)
		return
	}

	restSuccess(w)
}
