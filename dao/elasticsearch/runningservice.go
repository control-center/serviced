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
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
)

func (this *ControlPlaneDao) GetRunningServices(request dao.EntityRequest, allRunningServices *[]dao.RunningService) (err error) {
	since := time.Now().Add(-time.Hour)

	hosts, err := this.facade.GetHosts(datastore.GetContext())
	if err != nil {
		return err
	}
	var rss []dao.RunningService
	for _, h := range hosts {
		insts, err := this.facade.GetHostInstances(datastore.GetContext(), since, h.ID)
		if err != nil {
			return err
		}
		for _, inst := range insts {
			rss = append(rss, convertInstanceToRunningService(inst))
		}
	}
	*allRunningServices = rss
	return nil
}

func (this *ControlPlaneDao) GetRunningServicesForHost(hostID string, services *[]dao.RunningService) error {
	since := time.Now().Add(-time.Hour)

	insts, err := this.facade.GetHostInstances(datastore.GetContext(), since, hostID)
	if err != nil {
		return nil
	}

	rss := make([]dao.RunningService, len(insts))
	for i, inst := range insts {
		rss[i] = convertInstanceToRunningService(inst)
	}
	*services = rss
	return nil
}

func (this *ControlPlaneDao) GetRunningServicesForService(serviceID string, services *[]dao.RunningService) error {
	since := time.Now().Add(-time.Hour)

	insts, err := this.facade.GetServiceInstances(datastore.GetContext(), since, serviceID)
	if err != nil {
		return err
	}

	rss := make([]dao.RunningService, len(insts))
	for i, inst := range insts {
		rss[i] = convertInstanceToRunningService(inst)
	}
	*services = rss
	return nil
}

// FIXME: this will be deleted
func convertInstanceToRunningService(inst service.Instance) dao.RunningService {
	return dao.RunningService{
		ID:            fmt.Sprintf("%s-%s-%d", inst.HostID, inst.ServiceID, inst.InstanceID),
		ServiceID:     inst.ServiceID,
		HostID:        inst.HostID,
		DockerID:      inst.ContainerID,
		StartedAt:     inst.Started,
		InSync:        inst.ImageSynced,
		Name:          inst.ServiceName,
		DesiredState:  int(inst.DesiredState),
		RAMCommitment: utils.NewEngNotation(inst.RAMCommitment),
		RAMThreshold:  inst.RAMThreshold,
		InstanceID:    inst.InstanceID,
	}
}
