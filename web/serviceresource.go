package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"

	"net/url"
)

// RestServiceAutomaticAssignIP rest resource for automatic assigning ips to a service
func RestServiceAutomaticAssignIP(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Could not get serviceId: %v", err)
		RestBadRequest(w)
		return
	}

	request := dao.AssignmentRequest{serviceID, "", true}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Error("Failed to automatically assign IPs: %+v -> %v", request, err)
		RestServerError(w)
		return
	}

	RestSuccess(w)
}

// RestServiceManualAssignIP rest resource for manual assigning ips to a service
func RestServiceManualAssignIP(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		glog.Errorf("Could not get serviceId: %v", err)
		RestBadRequest(w)
		return
	}

	ip, err := url.QueryUnescape(r.PathParam("ip"))
	if err != nil {
		glog.Errorf("Could not get serviceId: %v", err)
		RestBadRequest(w)
		return
	}

	request := dao.AssignmentRequest{serviceID, ip, false}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Error("Failed to manually assign IP: %+v -> %v", request, err)
		RestServerError(w)
		return
	}

	RestSuccess(w)
}
