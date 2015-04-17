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
	"fmt"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/zenoss/glog"
)

func (this *ControlPlaneDao) GetServiceLogs(serviceID string, logs *string) error {
	glog.V(3).Info("ControlPlaneDao.GetServiceLogs serviceID=", serviceID)
	var serviceStates []servicestate.ServiceState
	if err := this.GetServiceStates(serviceID, &serviceStates); err != nil {
		glog.Errorf("ControlPlaneDao.GetServiceLogs failed: %v", err)
		return err
	}

	if len(serviceStates) == 0 {
		glog.V(1).Info("Unable to find any running services for service:", serviceID)
		return nil
	}

	serviceState := serviceStates[0]
	endpoint := fmt.Sprintf("%s:%d", serviceState.HostIP, this.rpcPort)
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

	endpoint := fmt.Sprintf("%s:%d", serviceState.HostIP, this.rpcPort)

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
