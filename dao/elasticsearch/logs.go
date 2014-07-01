// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/rpc/agent"
	"github.com/zenoss/serviced/zzk"
)

func (this *ControlPlaneDao) GetServiceLogs(serviceID string, logs *string) error {
	glog.V(3).Info("ControlPlaneDao.GetServiceLogs serviceID=", serviceID)

	var myService service.Service
	if err := this.GetService(serviceID, &myService); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServiceLogs service=%+v err=%s", serviceID, err)
		return err
	}

	poolBasedConn, err := zzk.GetPoolBasedConnection(myService.PoolID)
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", myService.PoolID, err)
		return err
	}

	var serviceStates []*servicestate.ServiceState
	if err := zzk.GetServiceStates(poolBasedConn, &serviceStates, serviceID); err != nil {
		return err
	}
	if len(serviceStates) == 0 {
		glog.V(1).Info("Unable to find any running services for service:", serviceID)
		return nil
	}

	serviceState := serviceStates[0]
	// FIXME: don't assume port is 4979
	endpoint := serviceState.HostIP + ":4979"
	agentClient, err := agent.NewClient(endpoint)
	if err != nil {
		glog.Errorf("could not create client to %s", endpoint)
		return err
	}
	defer agentClient.Close()
	if mylogs, err := agentClient.GetDockerLogs(serviceState.DockerID); err != nil {
		glog.Errorf("could not get docker logs from agent client: %s", err)
		return err
	} else {
		*logs = mylogs
	}
	return nil
}

func (this *ControlPlaneDao) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	var myService service.Service
	if err := this.GetService(request.ServiceID, &myService); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServiceLogs service=%+v err=%s", request.ServiceID, err)
		return err
	}

	poolBasedConn, err := zzk.GetPoolBasedConnection(myService.PoolID)
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", myService.PoolID, err)
		return err
	}

	/* TODO: This command does not support logs on other hosts */
	glog.V(3).Info("ControlPlaneDao.GetServiceStateLogs id=", request)
	var serviceState servicestate.ServiceState
	if err := zzk.GetServiceState(poolBasedConn, &serviceState, request.ServiceID, request.ServiceStateID); err != nil {
		glog.Errorf("ControlPlaneDao.GetServiceStateLogs servicestate=%+v err=%s", serviceState, err)
		return err
	}

	// FIXME: don't assume port is 4979
	endpoint := serviceState.HostIP + ":4979"
	agentClient, err := agent.NewClient(endpoint)
	if err != nil {
		glog.Errorf("could not create client to %s", endpoint)
		return err
	}
	defer agentClient.Close()
	if mylogs, err := agentClient.GetDockerLogs(serviceState.DockerID); err != nil {
		glog.Errorf("could not get docker logs from agent client: %s", err)
		return err
	} else {
		*logs = mylogs
	}
	return nil
}
