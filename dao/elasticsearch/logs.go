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
	"errors"
	"fmt"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/control-center/serviced/zzk/service2"
	"github.com/zenoss/glog"
)

func (this *ControlPlaneDao) GetServiceLogs(serviceID string, logs *string) error {
	glog.V(3).Info("ControlPlaneDao.GetServiceLogs serviceID=", serviceID)
	states, err := this.facade.GetServiceStates(datastore.Get(), serviceID)
	if err != nil {
		glog.Errorf("ControlPlaneDao.GetServiceLogs failed: %v", err)
		return err
	}

	if len(states) == 0 {
		glog.V(1).Info("Unable to find any running services for service:", serviceID)
		return nil
	}

	endpoint := fmt.Sprintf("%s:%d", states[0].HostIP, this.rpcPort)
	agentClient, err := agent.NewClient(endpoint)
	if err != nil {
		glog.Errorf("could not create client to %s", endpoint)
		return err
	}

	defer agentClient.Close()
	if mylogs, err := agentClient.GetDockerLogs(states[0].ContainerID); err != nil {
		glog.Errorf("could not get docker logs from agent client: %s", err)
		return err
	} else {
		*logs = mylogs
	}
	return nil
}

func (this *ControlPlaneDao) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	// FIXME: need good implementation

	states, err := this.facade.GetServiceStates(datastore.Get(), request.ServiceID)
	if err != nil {
		glog.Errorf("ControlPlaneDao.GetServiceStateLogs servicestate=%+v err=%s", states, err)
		return err
	}

	hostID, serviceID, instanceID, err := service.ParseStateID(request.ServiceStateID)
	if err != nil {
		return err
	}

	for _, state := range states {
		if hostID == state.HostID && serviceID == state.ServiceID && instanceID == state.InstanceID {
			endpoint := fmt.Sprintf("%s:%d", state.HostIP, this.rpcPort)
			agentClient, err := agent.NewClient(endpoint)
			if err != nil {
				glog.Errorf("could not create client to %s", endpoint)
				return err
			}

			defer agentClient.Close()
			if mylogs, err := agentClient.GetDockerLogs(state.ContainerID); err != nil {
				glog.Errorf("could not get docker logs from agent client: %s", err)
				return err
			} else {
				*logs = mylogs
			}
			return nil
		}
	}

	return errors.New("instance not found")
}
