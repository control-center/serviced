// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/zzk"

	"fmt"
)

func (this *ControlPlaneDao) GetRunningServices(request dao.EntityRequest, allRunningServices *[]*dao.RunningService) error {
	allPools, err := this.facade.GetResourcePools(datastore.Get())
	if err != nil {
		glog.Error("runningservice.go failed to get resource pool")
		return err
	} else if allPools == nil || len(allPools) == 0 {
		return fmt.Errorf("no resource pools found")
	}

	for _, aPool := range allPools {
		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(aPool.ID))
		if err != nil {
			glog.Error("runningservice.go Failed to get connection based on pool: %v", aPool.ID)
			return err
		}

		singlePoolRunningServices := []*dao.RunningService{}
		if err := zzk.GetAllRunningServices(poolBasedConn, &singlePoolRunningServices); err != nil {
			glog.Errorf("Failed GetAllRunningServices: %v", err)
			return err
		}

		*allRunningServices = append(*allRunningServices, singlePoolRunningServices...)
	}

	return nil
}

func (this *ControlPlaneDao) GetRunningServicesForHost(hostId string, services *[]*dao.RunningService) error {
	myHost, err := this.facade.GetHost(datastore.Get(), hostId)
	if err != nil {
		glog.Errorf("Unable to get host %v: %v", hostId, err)
		return err
	}

	poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myHost.PoolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", myHost.PoolID, err)
		return err
	}

	return zzk.GetRunningServicesForHost(poolBasedConn, hostId, services)
}

func (this *ControlPlaneDao) GetRunningServicesForService(serviceId string, services *[]*dao.RunningService) error {
	myService, err := this.facade.GetService(datastore.Get(), serviceId)
	if err != nil {
		glog.Errorf("Unable to get service %v: %v", serviceId, err)
		return err
	}

	poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myService.PoolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", myService.PoolID, err)
		return err
	}

	return zzk.GetRunningServicesForService(poolBasedConn, serviceId, services)
}

func (this *ControlPlaneDao) GetRunningService(request dao.ServiceStateRequest, running *dao.RunningService) error {
	glog.V(3).Infof("ControlPlaneDao.GetRunningService: request=%v", request)

	var myService service.Service
	if err := this.GetService(request.ServiceID, &myService); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServiceLogs service=%+v err=%s", request.ServiceID, err)
		return err
	}

	poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myService.PoolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", myService.PoolID, err)
		return err
	}

	return zzk.GetRunningService(poolBasedConn, request.ServiceID, request.ServiceStateID, running)
}
