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

	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	zks "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

func (this *ControlPlaneDao) getPoolBasedConnection(serviceID string) (client.Connection, error) {
	poolID, err := this.facade.GetPoolForService(datastore.GetContext(), serviceID)
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

func (this *ControlPlaneDao) StopRunningInstance(request dao.HostServiceRequest, unused *int) error {
	_, serviceID, instanceID, err := zks.ParseStateID(request.ServiceStateID)
	if err != nil {
		return err
	}
	return this.facade.StopServiceInstance(datastore.GetContext(), serviceID, instanceID)
}

func (this *ControlPlaneDao) GetServiceStatus(serviceID string, status *[]service.Instance) error {
	since := time.Now().Add(-time.Hour)
	inst, err := this.facade.GetServiceInstances(datastore.GetContext(), since, serviceID)
	if err != nil {
		return err
	}
	*status = inst
	return nil
}
