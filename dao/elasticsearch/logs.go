// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/servicestate"

	"os/exec"
)

func (this *ControlPlaneDao) GetServiceLogs(id string, logs *string) error {
	glog.V(3).Info("ControlPlaneDao.GetServiceLogs id=", id)
	var serviceStates []*servicestate.ServiceState
	err := this.zkDao.GetServiceStates(&serviceStates, id)
	if err != nil {
		return err
	}
	if len(serviceStates) == 0 {
		glog.V(1).Info("Unable to find any running services for ", id)
		return nil
	}
	cmd := exec.Command("docker", "logs", serviceStates[0].DockerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Unable to return logs because: %v", err)
		return err
	}
	*logs = string(output)
	return nil
}

func (this *ControlPlaneDao) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	/* TODO: This command does not support logs on other hosts */
	glog.V(3).Info("ControlPlaneDao.GetServiceStateLogs id=", request)
	var serviceState servicestate.ServiceState
	err := this.zkDao.GetServiceState(&serviceState, request.ServiceID, request.ServiceStateID)
	if err != nil {
		glog.Errorf("ControlPlaneDao.GetServiceStateLogs servicestate=%+v err=%s", serviceState, err)
		return err
	}

	cmd := exec.Command("docker", "logs", serviceState.DockerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Unable to return logs because: %v", err)
		return err
	}
	*logs = string(output)
	return nil
}
