// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/zzk"
	zkservice "github.com/zenoss/serviced/zzk/service"
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
