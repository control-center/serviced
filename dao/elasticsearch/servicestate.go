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
	//	"errors"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

func (this *ControlPlaneDao) GetServiceState(request dao.ServiceStateRequest, serviceState *servicestate.ServiceState) error {
	glog.V(3).Infof("ControlPlaneDao.GetServiceState: request=%v", request)

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

	return zkservice.GetServiceState(poolBasedConn, serviceState, request.ServiceID, request.ServiceStateID)
}

func (this *ControlPlaneDao) GetServiceStates(serviceId string, serviceStates *[]*servicestate.ServiceState) error {
	glog.V(2).Infof("ControlPlaneDao.GetServiceStates: serviceId=%s", serviceId)

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

	*serviceStates, err = zkservice.GetServiceStates(poolBasedConn, serviceId)
	return err
}

/* This method assumes that if a service instance exists, it has not yet been terminated */
func (this *ControlPlaneDao) getNonTerminatedServiceStates(serviceId string, serviceStates *[]*servicestate.ServiceState) error {
	glog.V(2).Infof("ControlPlaneDao.getNonTerminatedServiceStates: serviceId=%s", serviceId)

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

	*serviceStates, err = zkservice.GetServiceStates(poolBasedConn, serviceId)
	return err
}

// Update the current state of a service instance.
func (this *ControlPlaneDao) UpdateServiceState(state servicestate.ServiceState, unused *int) error {
	glog.V(2).Infoln("ControlPlaneDao.UpdateServiceState state=%+v", state)

	myService, err := this.facade.GetService(datastore.Get(), state.ServiceID)
	if err != nil {
		glog.Errorf("Unable to get service %v: %v", state.ServiceID, err)
		return err
	}

	poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myService.PoolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", myService.PoolID, err)
		return err
	}

	return zkservice.UpdateServiceState(poolBasedConn, &state)
}

func (this *ControlPlaneDao) StopRunningInstance(request dao.HostServiceRequest, unused *int) error {
	myHost, err := this.facade.GetHost(datastore.Get(), request.HostID)
	if err != nil {
		glog.Errorf("Unable to get host %v: %v", request.HostID, err)
		return err
	}

	poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(myHost.PoolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", myHost.PoolID, err)
		return err
	}

	if err := zkservice.StopServiceInstance(poolBasedConn, request.HostID, request.ServiceStateID); err != nil {
		glog.Errorf("zkservice.StopServiceInstance failed (conn: %+v hostID: %v serviceStateID: %v): %v", poolBasedConn, request.HostID, request.ServiceStateID, err)
		return err
	}

	return nil
}

func (this *ControlPlaneDao) GetServiceStatus(serviceID string, status *map[*servicestate.ServiceState]dao.Status) error {
	svc, err := this.facade.GetService(datastore.Get(), serviceID)
	if err != nil {
		glog.Errorf("Unable to get service %s: %s", serviceID, err)
		return err
	}

	poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(svc.PoolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %s: %s", svc.PoolID, err)
		return err
	}

	*status, err = zkservice.GetServiceStatus(poolBasedConn, svc.ID)
	if err != nil {
		glog.Errorf("zkservice.GetServiceStatus failed (conn: %+v serviceID: %s): %s", poolBasedConn, serviceID, err)
		return err
	}

	return nil
}
