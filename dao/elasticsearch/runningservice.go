// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"

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
		singlePoolRunningServices, err = zkservice.LoadRunningServices(poolBasedConn)
		if err != nil {
			glog.Errorf("Failed GetAllRunningServices: %v", err)
			return err
		}

		*allRunningServices = append(*allRunningServices, singlePoolRunningServices...)
	}

	return nil
}

func (this *ControlPlaneDao) GetRunningServicesForHost(hostID string, services *[]*dao.RunningService) error {
	myHost, err := this.facade.GetHost(datastore.Get(), hostID)
	if err != nil {
		glog.Errorf("Unable to get host %v: %v", hostID, err)
		return err
	} else if myHost == nil {
		return nil
	}

	poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myHost.PoolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", myHost.PoolID, err)
		return err
	}

	*services, err = zkservice.LoadRunningServicesByHost(poolBasedConn, hostID)
	if err != nil {
		glog.Errorf("zkservice.LoadRunningServicesByHost (conn: %+v host: %v) failed: %v", poolBasedConn, hostID, err)
		return err
	}

	return nil
}

func (this *ControlPlaneDao) GetRunningServicesForService(serviceID string, services *[]*dao.RunningService) error {
	myService, err := this.facade.GetService(datastore.Get(), serviceID)
	if err != nil {
		glog.Errorf("Unable to get service %v: %v", serviceID, err)
		return err
	}

	poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myService.PoolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", myService.PoolID, err)
		return err
	}

	*services, err = zkservice.LoadRunningServicesByService(poolBasedConn, serviceID)
	if err != nil {
		glog.Errorf("LoadRunningServicesByService failed (conn: %+v serviceID: %v): %v", poolBasedConn, serviceID, err)
		return err
	}

	return nil
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

	running, err = zkservice.LoadRunningService(poolBasedConn, request.ServiceID, request.ServiceStateID)
	if err != nil {
		glog.Errorf("zkservice.LoadRunningService failed (conn: %+v serviceID: %v): %v", poolBasedConn, request.ServiceID, err)
		return err
	}

	return nil
}
