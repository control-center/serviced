// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package scheduler

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/zzk"
	zkscheduler "github.com/zenoss/serviced/zzk/scheduler"
	zkservice "github.com/zenoss/serviced/zzk/service"
	"github.com/zenoss/serviced/zzk/virtualips"

	"time"
)

type Synchronizer struct {
	facade  *facade.Facade
	context datastore.Context
}

// create a new Synchronizer
func NewSynchronizer(myFacade *facade.Facade, myContext datastore.Context) *Synchronizer {
	s := new(Synchronizer)
	s.facade = myFacade
	s.context = myContext
	return s
}

func (s *Synchronizer) syncPools() error {
	// retrieve the pools in elastic search
	allPools, err := s.facade.GetResourcePools(s.context)
	if err != nil {
		glog.Errorf("failed to get resource pools: %v", err)
	} else if allPools == nil || len(allPools) == 0 {
		glog.Error("no resource pools found")
	}

	// retrieve the pools found in zookeeper
	rootConn, err := zzk.GetBasePathConnection("/")
	if err != nil {
		glog.Errorf("could not get root zk connection: %v", err)
		return err
	}
	return zkscheduler.SyncResourcePools(rootConn, allPools)
}

func (s *Synchronizer) syncServices() error {
	// create a map of services by PoolID
	servicesMap := make(map[string][]*service.Service)

	// retrieve ALL of the services found in zookeeper (in all pools)
	allPools, err := s.facade.GetResourcePools(s.context)
	if err != nil {
		glog.Errorf("failed to get resource pools: %v", err)
	} else if allPools == nil || len(allPools) == 0 {
		glog.Error("no resource pools found")
	}

	for _, pool := range allPools {
		servicesMap[pool.ID] = []*service.Service{}
	}

	// retrieve ALL of the services in elastic search
	myServices, err := s.facade.GetServices(s.context)
	if err != nil {
		glog.Errorf("could not retrieve services: %v", err)
		return err
	}

	for _, myService := range myServices {
		services := servicesMap[myService.PoolID]
		servicesMap[myService.PoolID] = append(services, myService)
	}

	// sync services by PoolID
	for poolID, services := range servicesMap {
		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(poolID))
		if err != nil {
			glog.Errorf("could not get pool based zk connection to %v: %v", aPool.ID, err)
			return err
		}
		if err := zkservice.SyncServices(poolBasedConn, services); err != nil {
			glog.Errorf("Could not sync services for pool %s: %s", poolID, err)
			return err
		}
	}

	return nil
}

func (s *Synchronizer) syncHosts() error {
	// create a map of hosts by PoolID
	hostsMap := make(map[string][]*host.Host)

	// retrieve ALL of the hosts found in zookeeper (in all pools)
	allPools, err := s.facade.GetResourcePools(s.context)
	if err != nil {
		glog.Errorf("failed to get resource pools: %v", err)
	} else if allPools == nil || len(allPools) == 0 {
		glog.Error("no resource pools found")
	}

	for _, pool := range allPools {
		hostsMap[pool.ID] = []*host.Host{}
	}

	// retrieve ALL of the hosts in elastic search
	myHosts, err := s.facade.GetHosts(s.context)
	if err != nil {
		glog.Errorf("could not retrieve hosts: %v", err)
		return err
	}

	for _, myHost := range myHosts {
		hosts := hostsMap[myHost.PoolID]
		hostsMap[myHost.PoolID] = append(hosts, myHost)
	}

	// sync hosts by PoolID
	for poolID, hosts := range servicesMap {
		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(poolID))
		if err != nil {
			glog.Errorf("could not get pool based zk connection to %v: %v", poolID, err)
			return err
		}
		if err := zkservice.SyncHosts(poolBasedConn, hosts); err != nil {
			glog.Errorf("Could not sync services for pool %s: %s", poolID, err)
			return err
		}
	}

	return nil
}

func (s *Synchronizer) syncVirtualIPs() error {
	allPools, err := s.facade.GetResourcePools(s.context)
	if err != nil {
		glog.Errorf("failed to get resource pools: %v", err)
	} else if allPools == nil || len(allPools) == 0 {
		glog.Error("no resource pools found")
	}

	for _, aPool := range allPools {
		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(aPool.ID))
		if err != nil {
			glog.Errorf("Could not get pool based zk connection to %v: %v", aPool.ID, err)
			return err
		}

		if err := virtualips.SyncVirtualIPs(poolBasedConn, aPool.VirtualIPs); err != nil {
			glog.Errorf("virtualips.SyncVirtualIPs on pool %v failed: %v", aPool.ID, err)
			return err
		}
	}

	return nil
}

// SyncAll will sync the following:
//   pools
//   services in a pool
//   hosts in a pool
//   virtual IPs
func (s *Synchronizer) SyncAll() bool {
	if err := s.syncPools(); err != nil {
		glog.Errorf("syncPools failed to sync: %v", err)
		return false
	}

	if err := s.syncServices(); err != nil {
		glog.Errorf("syncServices failed to sync: %v", err)
		return false
	}

	if err := s.syncHosts(); err != nil {
		glog.Errorf("syncHosts failed to sync: %v", err)
		return false
	}

	if err := s.syncVirtualIPs(); err != nil {
		glog.Errorf("syncVirtualIPs failed to sync: %v", err)
		return false
	}

	return true
}

// SyncAll will sync the following every 3 hours:
//   pools
//   services in a pool
//   hosts in a pool
//   virtual IPs
func (s *Synchronizer) SyncLoop(shutdown <-chan interface{}) {

	var wait <-chan time.Time
	for {
		if success := s.SyncAll(); success {
			wait = time.After(3 * time.Hour)
		} else {
			wait = time.After(30 * time.Second)
		}

		select {
		case <-wait:
			//pass
		case <-shutdown:
			return
		}
	}
}
