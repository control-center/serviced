package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"

	"net/url"
)

// FIXME json object for putting or deleting virtual ips (to be replaced with actual dao.VirtualIpRequest)
type virtualIPRequest struct {
	PoolID        string
	IP            string
	Netmask       string
	BindInterface string
}

// RestAddPoolVirtualIP takes a poolID, IP, netmask, and bindinterface and adds it
func RestAddPoolVirtualIP(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	//TODO replace virtualiprequest with model object
	var request virtualIPRequest
	err := r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w)
		return
	}

	glog.V(0).Infof("Add virtual ip: %+v", request)
	//TODO make call to dao service
	restSuccess(w)
}

// RestRemovePoolVirtualIP deletes virtualip
func RestRemovePoolVirtualIP(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	id, err := url.QueryUnescape(r.PathParam("id"))
	if err != nil {
		glog.Errorf("Could not get virtual ip - id: %v", err)
		restBadRequest(w)
		return
	}

	glog.V(0).Infof("Remove virtual ip - id=%s", id)
	//TODO make call to dao service
	restSuccess(w)
}
