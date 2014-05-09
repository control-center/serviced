// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/rpc/agent"

	"net/url"
	"strings"
)

//RestGetHosts gets all hosts. Response is map[host-id]host.Host
func RestGetHosts(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	response := make(map[string]*host.Host)
	client, err := ctx.getMasterClient()
	if err != nil {
		RestServerError(w)
		return
	}

	hosts, err := client.GetHosts()
	if err != nil {
		glog.Errorf("Could not get hosts: %v", err)
		RestServerError(w)
		return
	}
	glog.V(2).Infof("Returning %d hosts", len(hosts))
	for _, host := range hosts {
		response[host.ID] = host
	}

	w.WriteJson(&response)
}

//RestGetHost retrieves a host. Response is Host
func RestGetHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	client, err := ctx.getMasterClient()
	if err != nil {
		RestServerError(w)
		return
	}

	host, err := client.GetHost(hostID)
	if err != nil {
		glog.Error("Could not get host: ", err)
		RestServerError(w)
		return
	}

	glog.V(4).Infof("RestGetHost: id %s, host %#v", hostID, host)
	w.WriteJson(&host)
}

//RestAddHost adds a Host. Request input is host.Host
func RestAddHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	var payload host.Host
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Infof("Could not decode host payload: %v", err)
		RestBadRequest(w)
		return
	}
	// Save the pool ID and IP address for later. GetInfo wipes these
	ipAddr := payload.IPAddr
	parts := strings.Split(ipAddr, ":")
	hostIP := parts[0]

	agentClient, err := agent.NewClient(payload.IPAddr)
	//	remoteClient, err := serviced.NewAgentClient(payload.IPAddr)
	if err != nil {
		glog.Errorf("Could not create connection to host %s: %v", payload.IPAddr, err)
		RestServerError(w)
		return
	}

	IPs := []string{}
	for _, ip := range payload.IPs {
		IPs = append(IPs, ip.IPAddress)
	}
	buildRequest := agent.BuildHostRequest{
		IP:     hostIP,
		PoolID: payload.PoolID,
	}
	host, err := agentClient.BuildHost(buildRequest)
	if err != nil {
		glog.Errorf("Unable to get remote host info: %v", err)
		RestBadRequest(w)
		return
	}
	masterClient, err := ctx.getMasterClient()
	if err != nil {
		glog.Errorf("Unable to add host: %v", err)
		RestServerError(w)
		return
	}
	err = masterClient.AddHost(*host)
	if err != nil {
		glog.Errorf("Unable to add host: %v", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Added host ", host.ID)
	w.WriteJson(&SimpleResponse{"Added host", hostLinks(host.ID)})
}

//RestUpdateHost updates a host. Request input is host.Host
func RestUpdateHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	glog.V(3).Infof("Received update request for %s", hostID)
	var payload host.Host
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Infof("Could not decode host payload: %v", err)
		RestBadRequest(w)
		return
	}

	masterClient, err := ctx.getMasterClient()
	if err != nil {
		glog.Errorf("Unable to add host: %v", err)
		RestServerError(w)
		return
	}
	err = masterClient.UpdateHost(payload)
	if err != nil {
		glog.Errorf("Unable to update host: %v", err)
		RestServerError(w)
		return
	}
	glog.V(1).Info("Updated host ", hostID)
	w.WriteJson(&SimpleResponse{"Updated host", hostLinks(hostID)})
}

//RestRemoveHost removes a host using host-id
func RestRemoveHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	masterClient, err := ctx.getMasterClient()
	if err != nil {
		glog.Errorf("Unable to add host: %v", err)
		RestServerError(w)
		return
	}
	err = masterClient.RemoveHost(hostID)
	if err != nil {
		glog.Errorf("Could not remove host: %v", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Removed host ", hostID)
	w.WriteJson(&SimpleResponse{"Removed host", hostsLinks()})
}
