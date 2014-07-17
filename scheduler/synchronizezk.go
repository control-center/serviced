// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package scheduler

import (
	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/zzk"
	zkservice "github.com/zenoss/serviced/zzk/service"
	zkutils "github.com/zenoss/serviced/zzk/utils"
	"github.com/zenoss/serviced/zzk/virtualips"

	"time"
)

type Synchronizer struct {
	facade  *facade.Facade
	context datastore.Context
	conn    coordclient.Connection
}

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

	// create a map of ALL the services found in elastic search
	poolsMap := make(map[string]*pool.ResourcePool)
	for _, aPool := range allPools {
		poolsMap[aPool.ID] = aPool
	}

	// retrieve the pools found in zookeeper
	rootConn, err := zzk.GetBasePathConnection("/")
	if err != nil {
		glog.Errorf("could not get root zk connection: %v", err)
		return err
	}
	poolIDs, err := rootConn.Children("/pools")
	if err != nil {
		glog.Errorf("could not retrieve children of /pools: %v", err)
		return err
	}

	// remove the pools that are in zookeeper but not in elastic search
	for _, poolID := range poolIDs {
		if _, ok := poolsMap[poolID]; !ok {
			removePoolPath := "/pools/" + poolID
			glog.Infof("removing pool %v from zookeeper", removePoolPath, poolID)
			rootConn.Delete(removePoolPath)
		}
	}

	// update zookeeper for all the pools found in elastic search
	for _, aPool := range allPools {
		poolPath := "/pools/" + aPool.ID
		exists, err := zkutils.PathExists(rootConn, poolPath)
		if err != nil {
			return err
		}
		if !exists {
			if err := rootConn.CreateDir(poolPath); err != nil {
				glog.Errorf("failed to create %v: %v", poolPath, err)
				return err
			}
		}
	}

	return nil
}

func (s *Synchronizer) syncServices() error {
	// retrieve ALL of the services in elastic search
	myServices, err := s.facade.GetServices(s.context)
	if err != nil {
		glog.Errorf("could not retrieve services: %v", err)
		return err
	}

	// create a map of ALL the services found in elastic search
	servicesMap := make(map[string]*service.Service)
	for _, myService := range myServices {
		servicesMap[myService.ID] = myService
	}

	// retrieve ALL of the services found in zookeeper (in all pools)
	allPools, err := s.facade.GetResourcePools(s.context)
	if err != nil {
		glog.Errorf("failed to get resource pools: %v", err)
	} else if allPools == nil || len(allPools) == 0 {
		glog.Error("no resource pools found")
	}

	var allServiceIDs []string
	for _, aPool := range allPools {
		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(aPool.ID))
		if err != nil {
			glog.Errorf("could not get pool based zk connection to %v: %v", aPool.ID, err)
			return err
		}

		// retrieve a pool's services in zookeeper
		serviceIDs, err := poolBasedConn.Children("/services")
		if err != nil {
			glog.Errorf("could not retrieve children of /pools/%v/services: %v", aPool.ID, err)
			return err
		}
		allServiceIDs = append(allServiceIDs, serviceIDs...)
	}

	// remove the services that are in zookeeper but not in elastic search
	for _, serviceID := range allServiceIDs {
		if _, ok := servicesMap[serviceID]; !ok {
			myService, err := s.facade.GetService(s.context, serviceID)
			if err != nil {
				return err
			}
			poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myService.PoolID))
			if err == nil {
				glog.Errorf("could not get pool based zk connection to %v: %v", myService.PoolID, err)
			}
			removeServicePath := "/services/" + serviceID
			glog.Infof("removing service %v (in pool %v) from zookeeper", removeServicePath, myService.PoolID)
			poolBasedConn.Delete(removeServicePath)
		}
	}

	// update all services found in elastic search (will create zookeeper nodes if needed)
	for _, myService := range myServices {
		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myService.PoolID))
		if err == nil {
			glog.Errorf("could not get pool based zk connection to %v: %v", myService.PoolID, err)
		}
		if err := zzk.UpdateService(poolBasedConn, myService); err != nil {
			glog.Errorf("could not update service %v: %v", myService.ID, err)
			return err
		}
	}

	return nil
}

func (s *Synchronizer) syncHosts() error {
	// retrieve ALL of the hosts in elastic search
	myHosts, err := s.facade.GetHosts(s.context)
	if err != nil {
		glog.Errorf("could not retrieve hosts: %v", err)
		return err
	}

	// create a map of ALL the hosts found in elastic search
	hostsMap := make(map[string]*host.Host)
	for _, myHost := range myHosts {
		hostsMap[myHost.ID] = myHost
	}

	// retrieve ALL of the hosts found in zookeeper (in all pools)
	allPools, err := s.facade.GetResourcePools(s.context)
	if err != nil {
		glog.Errorf("failed to get resource pools: %v", err)
	} else if allPools == nil || len(allPools) == 0 {
		glog.Error("no resource pools found")
	}

	var allHostIDs []string
	for _, aPool := range allPools {
		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(aPool.ID))
		if err != nil {
			glog.Errorf("Could not get pool based zk connection to %v: %v", aPool.ID, err)
			return err
		}

		// retrieve a pool's hosts in zookeeper
		hostIDs, err := poolBasedConn.Children("/hosts")
		if err != nil {
			glog.Errorf("Could not retrieve children of /pools/%v/hosts: %v", aPool.ID, err)
			return err
		}
		allHostIDs = append(allHostIDs, hostIDs...)
	}

	// remove the services that are in zookeeper but not in elastic search
	for _, hostID := range allHostIDs {
		if _, ok := hostsMap[hostID]; !ok {
			myHost, err := s.facade.GetHost(s.context, hostID)
			if err != nil {
				return err
			}
			poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myHost.PoolID))
			if err == nil {
				glog.Errorf("Could not get pool based zk connection to %v: %v", myHost.PoolID, err)
			}
			removeHostPath := "/hosts/" + hostID
			glog.Infof("Removing /pools/%v/hosts/%v", myHost.PoolID, hostID)
			poolBasedConn.Delete(removeHostPath)
		}
	}

	// update all hosts found in elastic search
	for _, myHost := range myHosts {
		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myHost.PoolID))
		if err == nil {
			glog.Errorf("Could not get pool based zk connection to %v: %v", myHost.PoolID, err)
		}
		if err := zkservice.RegisterHost(poolBasedConn, myHost.ID); err != nil {
			glog.Errorf("Could not register host %v: %v", myHost.ID, err)
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

func (s *Synchronizer) Sync() {
	syncSuccess := false
	for {
		if syncSuccess {
			time.Sleep(time.Hour * 3)
			syncSuccess = false
		} else {
			time.Sleep(time.Second * 30)
		}

		if err := s.syncPools(); err != nil {
			glog.Errorf("syncPools failed to sync: %v", err)
			continue // syncSuccess is false
		}

		if err := s.syncServices(); err != nil {
			glog.Errorf("syncServices failed to sync: %v", err)
			continue // syncSuccess is false
		}

		if err := s.syncHosts(); err != nil {
			glog.Errorf("syncHosts failed to sync: %v", err)
			continue // syncSuccess is false
		}

		if err := s.syncVirtualIPs(); err != nil {
			glog.Errorf("syncVirtualIPs failed to sync: %v", err)
			continue // syncSuccess is false
		}

		syncSuccess = true
	}
}
