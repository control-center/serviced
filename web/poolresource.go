// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of sc source code is governed by a
// license that can be found in the LICENSE file.

package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced/domain/pool"

	"github.com/zenoss/serviced/dao"
	"net/url"
)

//RestGetPools retrieves all Resource Pools. Response is map[pool-id]ResourcePool
func RestGetPools(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	client, err := ctx.getMasterClient()
	if err != nil {
		RestServerError(w)
		return
	}

	pools, err := client.GetResourcePools()
	if err != nil {
		glog.Error("Could not get resource pools: ", err)
		RestServerError(w)
		return
	}
	poolsMap := make(map[string]*pool.ResourcePool)
	for _, pool := range pools {
		poolsMap[pool.ID] = pool
	}
	glog.V(4).Infof("RestGetPools: pools %#v", poolsMap)
	w.WriteJson(&poolsMap)
}

//RestAddPool add a resource pool. Request input is pool.ResourcePool
func RestAddPool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	var payload pool.ResourcePool
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode pool payload: ", err)
		RestBadRequest(w)
		return
	}
	client, err := ctx.getMasterClient()
	if err != nil {
		RestServerError(w)
		return
	}

	err = client.AddResourcePool(payload)
	if err != nil {
		glog.Error("Unable to add pool: ", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Added pool ", payload.ID)
	w.WriteJson(&SimpleResponse{"Added resource pool", poolLinks(payload.ID)})
}

//RestUpdatePool updates a resource pool. Request input is pool.ResourcePool
func RestUpdatePool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	poolID, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var payload pool.ResourcePool
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode pool payload: ", err)
		RestBadRequest(w)
		return
	}
	client, err := ctx.getMasterClient()
	if err != nil {
		RestServerError(w)
		return
	}
	err = client.UpdateResourcePool(payload)
	if err != nil {
		glog.Error("Unable to update pool: ", err)
		RestServerError(w)
		return
	}
	glog.V(1).Info("Updated pool ", poolID)
	w.WriteJson(&SimpleResponse{"Updated resource pool", poolLinks(poolID)})
}

//RestRemovePool removes a resource pool using pool-id
func RestRemovePool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	poolID, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	client, err := ctx.getMasterClient()
	if err != nil {
		RestServerError(w)
		return
	}
	err = client.RemoveResourcePool(poolID)
	if err != nil {
		glog.Error("Could not remove resource pool: ", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Removed pool ", poolID)
	w.WriteJson(&SimpleResponse{"Removed resource pool", poolsLinks()})
}

//RestGetHostsForResourcePool gets all Hosts in a resource pool. response is [dao.PoolHost]
func RestGetHostsForResourcePool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	poolHosts := make([]*dao.PoolHost, 0)
	poolID, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		glog.V(1).Infof("Unable to acquire pool ID: %v", err)
		RestBadRequest(w)
		return
	}
	client, err := ctx.getMasterClient()
	if err != nil {
		RestServerError(w)
		return
	}
	hosts, err := client.FindHostsInPool(poolID)
	if err != nil {
		glog.Errorf("Could not get hosts: %v", err)
		RestServerError(w)
		return
	}
	for _, host := range hosts {
		ph := dao.PoolHost{
			HostId: host.ID,
			PoolId: poolID,
			HostIp: host.IPAddr,
		}
		poolHosts = append(poolHosts, &ph)
	}
	glog.V(2).Infof("Returning %d hosts for pool %s", len(poolHosts), poolID)
	w.WriteJson(&poolHosts)
}
