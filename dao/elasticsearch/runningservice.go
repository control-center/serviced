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

package elasticsearch

import (
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"

	"fmt"
)

func (this *ControlPlaneDao) GetRunningServices(request dao.EntityRequest, allRunningServices *[]dao.RunningService) error {
	// we initialize the data container to something here in case it has not been initialized yet
	*allRunningServices = make([]dao.RunningService, 0)
	allPools, err := this.facade.GetResourcePools(datastore.Get())
	if err != nil {
		glog.Error("runningservice.go failed to get resource pool")
		return err
	} else if allPools == nil || len(allPools) == 0 {
		return fmt.Errorf("no resource pools found")
	}

	for _, aPool := range allPools {
		poolBasedConn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(aPool.ID))
		if err != nil {
			glog.Error("runningservice.go Failed to get connection based on pool: %v", aPool.ID)
			return err
		}

		singlePoolRunningServices := []dao.RunningService{}
		singlePoolRunningServices, err = zkservice.LoadRunningServices(poolBasedConn)
		if err != nil {
			glog.Errorf("Failed GetAllRunningServices: %v", err)
			return err
		}

		for _, rs := range singlePoolRunningServices {
			*allRunningServices = append(*allRunningServices, rs)
		}
	}

	return nil
}

func (this *ControlPlaneDao) GetRunningServicesForHost(hostID string, services *[]dao.RunningService) error {
	// we initialize the data container to something here in case it has not been initialized yet
	*services = make([]dao.RunningService, 0)
	myHost, err := this.facade.GetHost(datastore.Get(), hostID)
	if err != nil {
		glog.Errorf("Unable to get host %v: %v", hostID, err)
		return err
	} else if myHost == nil {
		return nil
	}

	poolBasedConn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(myHost.PoolID))
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

func (this *ControlPlaneDao) GetRunningServicesForService(serviceID string, services *[]dao.RunningService) error {
	// we initialize the data container to something here in case it has not been initialized yet
	*services = make([]dao.RunningService, 0)

	poolID, err := this.facade.GetPoolForService(datastore.Get(), serviceID)
	if err != nil {
		glog.Errorf("Unable to get service %v: %v", serviceID, err)
		return err
	}

	poolBasedConn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", poolID, err)
		return err
	}

	svcs, err := zkservice.LoadRunningServicesByService(poolBasedConn, serviceID)
	if err != nil {
		glog.Errorf("LoadRunningServicesByService failed (conn: %+v serviceID: %v): %v", poolBasedConn, serviceID, err)
		return err
	}

	for _, svc := range svcs {
		*services = append(*services, svc)
	}

	return nil
}

func (this *ControlPlaneDao) GetRunningService(request dao.ServiceStateRequest, running *dao.RunningService) error {
	glog.V(3).Infof("ControlPlaneDao.GetRunningService: request=%v", request)
	*running = dao.RunningService{}

	serviceID := request.ServiceID
	poolID, err := this.facade.GetPoolForService(datastore.Get(), serviceID)
	if err != nil {
		glog.Errorf("Unable to get service %v: %v", request, err)
		return err
	}

	poolBasedConn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", poolID, err)
		return err
	}

	if thisRunning, err := zkservice.LoadRunningService(poolBasedConn, request.ServiceID, request.ServiceStateID); err != nil {
		glog.Errorf("zkservice.LoadRunningService failed (conn: %+v serviceID: %v): %v", poolBasedConn, request.ServiceID, err)
		return err
	} else {
		if thisRunning != nil {
			*running = *thisRunning
		}
	}

	return nil
}
