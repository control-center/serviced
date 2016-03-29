// Copyright 2014 The Serviced Authors.
// Use of sc source code is governed by a

package web

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/pool"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

	"net/url"

	"github.com/control-center/serviced/facade"
	"fmt"
)

//restGetPools retrieves all Resource Pools. Response is map[pool-id]ResourcePool
func restGetPools(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	pools, err := facade.GetResourcePools(dataCtx)
	if err != nil {
		glog.Error("Could not get resource pools: ", err)
		restServerError(w, err)
		return
	}

	poolsMap := make(map[string]*pool.ResourcePool)
	for i, pool := range pools {
		hostIDs, err := getPoolHostIds(pool.ID, facade, dataCtx)
		if err != nil {
			restServerError(w, err)
			return
		}

		if err := buildPoolMonitoringProfile(&pools[i], hostIDs, facade, dataCtx); err != nil {
			restServerError(w, err)
			return
		}

		poolsMap[pool.ID] = &pools[i]
	}
	glog.V(4).Infof("restGetPools: pools %#v", poolsMap)
	w.WriteJson(&poolsMap)
}

//restGetPool retrieves a Resource Pools. Response is ResourcePool
func restGetPool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	poolID, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		restBadRequest(w, err)
		return
	} else if len(poolID) == 0 {
		restBadRequest(w, fmt.Errorf("poolID must be specified for PUT"))
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	pool, err := facade.GetResourcePool(dataCtx, poolID)
	if err != nil {
		glog.Error("Could not get resource pool: ", err)
		restServerError(w, err)
		return
	}

	hostIDs, err := getPoolHostIds(pool.ID, facade, dataCtx)
	if err != nil {
		restServerError(w, err)
		return
	}

	if err := buildPoolMonitoringProfile(pool, hostIDs, facade, dataCtx); err != nil {
		restServerError(w, err)
		return
	}

	glog.V(4).Infof("restGetPool: id %s, pool %#v", poolID, pool)
	w.WriteJson(&pool)
}

//restAddPool add a resource pool. Request input is pool.ResourcePool
func restAddPool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	var payload pool.ResourcePool
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode pool payload: ", err)
		restBadRequest(w, err)
		return
	}

	facade := ctx.getFacade()
	err = facade.AddResourcePool(ctx.getDatastoreContext(), &payload)
	if err != nil {
		glog.Error("Unable to add pool: ", err)
		restServerError(w, err)
		return
	}
	glog.V(0).Info("Added pool ", payload.ID)
	w.WriteJson(&simpleResponse{"Added resource pool", poolLinks(payload.ID)})
}

//restUpdatePool updates a resource pool. Request input is pool.ResourcePool
func restUpdatePool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	poolID, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		restBadRequest(w, err)
		return
	} else if len(poolID) == 0 {
		restBadRequest(w, fmt.Errorf("poolID must be specified for PUT"))
		return
	}

	var payload pool.ResourcePool
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode pool payload: ", err)
		restBadRequest(w, err)
		return
	}

	facade := ctx.getFacade()
	err = facade.UpdateResourcePool(ctx.getDatastoreContext(), &payload)
	if err != nil {
		glog.Error("Unable to update pool: ", err)
		restServerError(w, err)
		return
	}
	glog.V(1).Info("Updated pool ", poolID)
	w.WriteJson(&simpleResponse{"Updated resource pool", poolLinks(poolID)})
}

//restRemovePool removes a resource pool using pool-id
func restRemovePool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	poolID, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		restBadRequest(w, err)
		return
	} else if len(poolID) == 0 {
		restBadRequest(w, fmt.Errorf("poolID must be specified for DELETE"))
		return
	}

	facade := ctx.getFacade()
	err = facade.RemoveResourcePool(ctx.getDatastoreContext(), poolID)
	if err != nil {
		glog.Error("Could not remove resource pool: ", err)
		restServerError(w, err)
		return
	}
	glog.V(0).Info("Removed pool ", poolID)
	w.WriteJson(&simpleResponse{"Removed resource pool", poolsLinks()})
}

//restGetHostsForResourcePool gets all Hosts in a resource pool. response is []pool.PoolHost
func restGetHostsForResourcePool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	poolHosts := []*pool.PoolHost{}
	poolID, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		glog.V(1).Infof("Unable to acquire pool ID: %v", err)
		restBadRequest(w, err)
		return
	} else if len(poolID) == 0 {
		restBadRequest(w, fmt.Errorf("poolID must be specified for DELETE"))
		return
	}

	facade := ctx.getFacade()
	hosts, err := facade.FindHostsInPool(ctx.getDatastoreContext(), poolID)
	if err != nil {
		glog.Errorf("Could not get hosts: %v", err)
		restServerError(w, err)
		return
	}
	for _, host := range hosts {
		ph := pool.PoolHost{
			HostID: host.ID,
			PoolID: poolID,
			HostIP: host.IPAddr,
		}
		poolHosts = append(poolHosts, &ph)
	}
	glog.V(2).Infof("Returning %d hosts for pool %s", len(poolHosts), poolID)
	w.WriteJson(&poolHosts)
}

//restGetPoolIps retrieves a Resource Pools. Response is ResourcePool
func restGetPoolIps(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	poolID, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		restBadRequest(w, err)
		return
	} else if len(poolID) == 0 {
		restBadRequest(w, fmt.Errorf("poolID must be specified for GET"))
		return
	}

	facade := ctx.getFacade()
	ips, err := facade.GetPoolIPs(ctx.getDatastoreContext(), poolID)
	if err != nil {
		glog.Error("Could not get resource pool: ", err)
		restServerError(w, err)
		return
	}

	glog.V(4).Infof("restGetPoolIps: id %s, pool %#v", poolID, ips)
	w.WriteJson(&ips)
}

func getPoolHostIds(poolID string, facade facade.FacadeInterface, dataCtx datastore.Context) ([]string, error) {
	hosts, err := facade.FindHostsInPool(dataCtx, poolID)
	if err != nil {
		glog.Errorf("Could not get hosts: %v", err)
		return nil, err
	}

	hostIDs := make([]string, len(hosts))
	for i := range hosts {
		hostIDs[i] = hosts[i].ID
	}
	return hostIDs, nil
}

func buildPoolMonitoringProfile(pool *pool.ResourcePool, hostIDs []string, facade facade.FacadeInterface, dataCtx datastore.Context) error {
	var totalMemory uint64
	var totalCores int

	//calculate total memory and total cores
	for i := range hostIDs {
		host, err := facade.GetHost(dataCtx, hostIDs[i])
		if err != nil {
			glog.Errorf("Failed to get host for id=%s -> %s", hostIDs[i], err)
			return err
		}

		totalCores += host.Cores
		totalMemory += host.Memory
	}

	tags := map[string][]string{"controlplane_host_id": hostIDs}
	profile, err := hostPoolProfile.ReBuild("1h-ago", tags)
	if err != nil {
		glog.Errorf("Failed to create pool profile: %s", err)
		return err
	}

	//add graphs to profile
	profile.GraphConfigs = make([]domain.GraphConfig, 3)
	profile.GraphConfigs[0] = newCpuConfigGraph(tags, totalCores)
	profile.GraphConfigs[1] = newRSSConfigGraph(tags, totalMemory)
	profile.GraphConfigs[2] = newMajorPageFaultGraph(tags)

	pool.MonitoringProfile = *profile
	return nil
}
