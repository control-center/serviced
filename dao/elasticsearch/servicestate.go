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

	"fmt"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

func (this *ControlPlaneDao) getPoolBasedConnection(serviceID string) (client.Connection, error) {
	poolID, err := this.facade.GetPoolForService(datastore.Get(), serviceID)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetPoolForService service=%+v err=%s", serviceID, err)
		return nil, err
	}

	poolBasedConn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		glog.Errorf("Error in getting a connection based on pool %v: %v", poolID, err)
		return nil, err
	}
	return poolBasedConn, nil
}

func (this *ControlPlaneDao) GetServiceState(request dao.ServiceStateRequest, state *servicestate.ServiceState) error {
	glog.V(3).Infof("ControlPlaneDao.GetServiceState: request=%v", request)
	*state = servicestate.ServiceState{}

	poolBasedConn, err := this.getPoolBasedConnection(request.ServiceID)
	if err != nil {
		return err
	}
	err = zkservice.GetServiceState(poolBasedConn, state, request.ServiceID, request.ServiceStateID)
	if state == nil {
		*state = servicestate.ServiceState{}
	}
	return err
}

func (this *ControlPlaneDao) GetServiceStates(serviceId string, states *[]servicestate.ServiceState) error {
	glog.V(2).Infof("ControlPlaneDao.GetServiceStates: serviceId=%s", serviceId)
	*states = make([]servicestate.ServiceState, 0)

	poolBasedConn, err := this.getPoolBasedConnection(serviceId)
	if err != nil {
		return err
	}

	serviceStates, err := zkservice.GetServiceStates(poolBasedConn, serviceId)
	if serviceStates != nil {
		*states = serviceStates
	}
	return err
}

/* This method assumes that if a service instance exists, it has not yet been terminated */
func (this *ControlPlaneDao) getNonTerminatedServiceStates(serviceId string, serviceStates *[]servicestate.ServiceState) error {
	glog.V(2).Infof("ControlPlaneDao.getNonTerminatedServiceStates: serviceId=%s", serviceId)

	poolBasedConn, err := this.getPoolBasedConnection(serviceId)
	if err != nil {
		return err
	}

	*serviceStates, err = zkservice.GetServiceStates(poolBasedConn, serviceId)
	return err
}

// Update the current state of a service instance.
func (this *ControlPlaneDao) UpdateServiceState(state servicestate.ServiceState, unused *int) error {
	glog.V(2).Infoln("ControlPlaneDao.UpdateServiceState state=%+v", state)

	poolBasedConn, err := this.getPoolBasedConnection(state.ServiceID)
	if err != nil {
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
	if myHost == nil {
		return fmt.Errorf("Host %s does not exist", request.HostID)
	}
	poolBasedConn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(myHost.PoolID))
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

func (this *ControlPlaneDao) GetServiceStatus(serviceID string, status *map[string]dao.ServiceStatus) error {
	*status = make(map[string]dao.ServiceStatus, 0)

	poolBasedConn, err := this.getPoolBasedConnection(serviceID)
	if err != nil {
		return err
	}

	st, err := zkservice.GetServiceStatus(poolBasedConn, serviceID)
	if err != nil {
		glog.Errorf("zkservice.GetServiceStatus failed (conn: %+v serviceID: %s): %s", poolBasedConn, serviceID, err)
		return err
	}

	if st != nil {
		//get all healthcheck statuses for this service
		healthStatuses, err := this.facade.GetServiceHealth(datastore.Get(), serviceID) // map[int]map[string]*domain.HealthCheckStatus
		if err != nil {
			glog.Errorf("Error getting service health checks (%s)", err)
			return nil
		}

		//merge st with healthcheck info into *status
		*status = make(map[string]dao.ServiceStatus, len(st))
		for stateID, instanceStatus := range st {
			instanceStatus.HealthCheckStatuses = healthStatuses[instanceStatus.State.InstanceID]
			(*status)[stateID] = instanceStatus
		}
	}

	return nil
}
