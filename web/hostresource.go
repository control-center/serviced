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
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

//restGetHosts gets all hosts. Response is map[host-id]host.Host
func restGetHosts(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	hosts, err := facade.GetHosts(dataCtx)
	if err != nil {
		glog.Errorf("Could not get hosts: %v", err)
		restServerError(w, err)
		return
	}

	glog.V(2).Infof("Returning %d hosts", len(hosts))
	response := make(map[string]*host.Host)
	for i, host := range hosts {
		response[host.ID] = &hosts[i]
		if err := buildHostMonitoringProfile(&hosts[i]); err != nil {
			restServerError(w, err)
			return
		}
	}

	w.WriteJson(&response)
}

func restGetActiveHostIDs(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	hostids, err := facade.GetActiveHostIDs(dataCtx)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(&hostids)

}

//restGetHost retrieves a host. Response is Host
func restGetHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		restBadRequest(w, err)
		return
	} else if len(hostID) == 0 {
		restBadRequest(w, fmt.Errorf("hostID must be specified for GET"))
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	host, err := facade.GetHost(dataCtx, hostID)
	if err != nil {
		glog.Error("Could not get host: ", err)
		restServerError(w, err)
		return
	}

	if err := buildHostMonitoringProfile(host); err != nil {
		restServerError(w, err)
		return
	}

	glog.V(4).Infof("restGetHost: id %s, host %#v", hostID, host)
	w.WriteJson(&host)
}

func restGetDefaultHostAlias(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	w.WriteJson(&map[string]string{"hostalias": defaultHostAlias})
}

type addHostRequest struct {
	IPAddr   string
	PoolID   string
	RAMLimit string
}

//restAddHost adds a Host. Request input is host.Host
func restAddHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	var payload addHostRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Infof("Could not decode host payload: %v", err)
		restBadRequest(w, err)
		return
	}
	ipAddr := payload.IPAddr
	parts := strings.Split(ipAddr, ":")
	hostIPAddr, err := net.ResolveIPAddr("ip", parts[0])
	if err != nil {
		glog.Errorf("%s could not be resolved", parts[0])
		restBadRequest(w, err)
		return
	}
	hostIP := hostIPAddr.IP.String()

	if len(parts) < 2 {
		glog.Errorf("rpcport needs to be specified")
		restBadRequest(w, err)
		return
	}

	rpcPort, err := strconv.Atoi(parts[1])
	if err != nil {
		glog.Errorf("could not convert rpcport %s to int", parts[1])
		restBadRequest(w, err)
		return
	}

	agentClient, err := agent.NewClient(payload.IPAddr)
	if err != nil {
		glog.Errorf("Could not create connection to host %s: %v", payload.IPAddr, err)
		restServerError(w, err)
		return
	}

	buildRequest := agent.BuildHostRequest{
		IP:     hostIP,
		Port:   rpcPort,
		PoolID: payload.PoolID,
		Memory: payload.RAMLimit,
	}
	host, err := agentClient.BuildHost(buildRequest)
	if err != nil {
		glog.Errorf("Unable to get remote host info: %v", err)
		restBadRequest(w, err)
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	err = facade.AddHost(dataCtx, host)
	if err != nil {
		glog.Errorf("Unable to add host: %v", err)
		restServerError(w, err)
		return
	}
	glog.V(0).Info("Added host ", host.ID)
	w.WriteJson(&simpleResponse{"Added host", hostLinks(host.ID)})
}

//restUpdateHost updates a host. Request input is host.Host
func restUpdateHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		restBadRequest(w, err)
		return
	} else if len(hostID) == 0 {
		restBadRequest(w, fmt.Errorf("hostID must be specified for PUT"))
		return
	}
	glog.V(3).Infof("Received update request for %s", hostID)
	var payload host.Host
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Infof("Could not decode host payload: %v", err)
		restBadRequest(w, err)
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	err = facade.UpdateHost(dataCtx, &payload)
	if err != nil {
		glog.Errorf("Unable to update host: %v", err)
		restServerError(w, err)
		return
	}
	glog.V(1).Info("Updated host ", hostID)
	w.WriteJson(&simpleResponse{"Updated host", hostLinks(hostID)})
}

//restRemoveHost removes a host using host-id
func restRemoveHost(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		restBadRequest(w, err)
		return
	} else if len(hostID) == 0 {
		restBadRequest(w, fmt.Errorf("hostID must be specified for DELETE"))
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	err = facade.RemoveHost(dataCtx, hostID)
	if err != nil {
		glog.Errorf("Could not remove host: %v", err)
		restServerError(w, err)
		return
	}
	glog.V(0).Info("Removed host ", hostID)
	w.WriteJson(&simpleResponse{"Removed host", hostsLinks()})
}

func buildHostMonitoringProfile(host *host.Host) error {
	tags := map[string][]string{"controlplane_host_id": []string{host.ID}}
	profile, err := hostPoolProfile.ReBuild("1h-ago", tags)
	if err != nil {
		glog.Errorf("Failed to create host profile: %s", err)
		return err
	}

	//add graphs to profile
	profile.GraphConfigs = make([]domain.GraphConfig, 6)
	profile.GraphConfigs[0] = newCpuConfigGraph(tags, host.Cores)
	profile.GraphConfigs[1] = newLoadAverageGraph(tags)
	profile.GraphConfigs[2] = newRSSConfigGraph(tags, host.Memory)
	profile.GraphConfigs[3] = newOpenFileDescriptorsGraph(tags)
	profile.GraphConfigs[4] = newMajorPageFaultGraph(tags)
	profile.GraphConfigs[5] = newPagingGraph(tags)

	host.MonitoringProfile = *profile
	return nil
}
