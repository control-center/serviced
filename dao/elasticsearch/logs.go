// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/rpc/agent"
)

func (this *ControlPlaneDao) GetServiceLogs(serviceID string, logs *string) error {
	glog.V(3).Info("ControlPlaneDao.GetServiceLogs serviceID=", serviceID)
	var serviceStates []*servicestate.ServiceState
	if err := this.GetServiceStates(serviceID, &serviceStates); err != nil {
		glog.Errorf("ControlPlaneDao.GetServiceLogs failed: %v", err)
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
	var serviceState servicestate.ServiceState
	if err := this.GetServiceState(request, &serviceState); err != nil {
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
