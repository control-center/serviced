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
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

const (
	userLockTimeout = time.Second
)

// AddService adds a new service. Returns an error if service already exists.
func (this *ControlPlaneDao) AddService(svc service.Service, serviceId *string) error {
	if err := this.facade.DFSLock(datastore.GetContext()).LockWithTimeout("add service", userLockTimeout); err != nil {
		glog.Warningf("Cannot add service: %s", err)
		return err
	}
	defer this.facade.DFSLock(datastore.GetContext()).Unlock()

	return this.addService(svc, serviceId)
}

func (this *ControlPlaneDao) addService(svc service.Service, serviceId *string) error {
	if err := this.facade.AddService(datastore.GetContext(), svc); err != nil {
		return err
	}
	*serviceId = svc.ID
	return nil
}

// CloneService clones a service. Returns an error if given serviceID is not found.
func (this *ControlPlaneDao) CloneService(request dao.ServiceCloneRequest, clonedServiceId *string) error {
	if err := this.facade.DFSLock(datastore.GetContext()).LockWithTimeout("clone service", userLockTimeout); err != nil {
		glog.Warningf("Cannot clone service: %s", err)
		return err
	}
	defer this.facade.DFSLock(datastore.GetContext()).Unlock()

	svc, err := this.facade.GetService(datastore.GetContext(), request.ServiceID)
	if err != nil {
		glog.Errorf("ControlPlaneDao.CloneService: unable to find service id %+v: %s", request.ServiceID, err)
		return err
	}

	cloned, err := service.CloneService(svc, request.Suffix)
	if err != nil {
		glog.Errorf("ControlPlaneDao.CloneService: unable to rename service %+v %v: %s", svc.ID, svc.Name, err)
		return err
	}

	if err := this.addService(*cloned, clonedServiceId); err != nil {
		return err
	}

	return nil
}

//
func (this *ControlPlaneDao) UpdateService(svc service.Service, unused *int) error {
	ctx := datastore.GetContext()
	if err := this.facade.DFSLock(ctx).LockWithTimeout("update service", userLockTimeout); err != nil {
		glog.Warningf("Cannot update service: %s", err)
		return err
	}
	defer this.facade.DFSLock(ctx).Unlock()

	err := this.facade.UpdateService(ctx, svc)
	if err == nil {
		// CC-3646 - rebuild logstash config in case the set of auditable log files has changed
		this.facade.ReloadLogstashConfig(ctx)
	}
	return err
}

//
func (this *ControlPlaneDao) MigrateServices(request dao.ServiceMigrationRequest, unused *int) error {
	if err := this.facade.DFSLock(datastore.GetContext()).LockWithTimeout("migrate service", userLockTimeout); err != nil {
		glog.Warningf("Cannot migrate service: %s", err)
		return err
	}
	defer this.facade.DFSLock(datastore.GetContext()).Unlock()

	return this.facade.MigrateServices(datastore.GetContext(), request)
}

func (this *ControlPlaneDao) GetServiceList(serviceID string, services *[]service.Service) error {
	if svcs, err := this.facade.GetServiceList(datastore.GetContext(), serviceID); err != nil {
		return err
	} else {
		var out []service.Service
		for _, svc := range svcs {
			out = append(out, *svc)
		}
		*services = out
		return nil
	}
}

//
func (this *ControlPlaneDao) RemoveService(id string, unused *int) error {
	if err := this.facade.DFSLock(datastore.GetContext()).LockWithTimeout("remove service", userLockTimeout); err != nil {
		glog.Warningf("Cannot remove service: %s", err)
		return err
	}
	defer this.facade.DFSLock(datastore.GetContext()).Unlock()

	return this.facade.RemoveService(datastore.GetContext(), id)
}

// GetService gets a service.
func (this *ControlPlaneDao) GetService(id string, myService *service.Service) error {
	svc, err := this.facade.GetService(datastore.GetContext(), id)
	if svc != nil {
		*myService = *svc
	}
	return err
}

// Get a list of tenant IDs
func (this *ControlPlaneDao) GetTenantIDs(unused struct{}, tenantIDs *[]string) error {
	ctx := datastore.GetContext()
	defer ctx.Metrics().Stop(ctx.Metrics().Start("dao.GetTenantIDs"))
	if ids, err := this.facade.GetTenantIDs(ctx); err == nil {
		*tenantIDs = ids
		return nil
	} else {
		return err
	}
}

//
func (this *ControlPlaneDao) FindChildService(request dao.FindChildRequest, service *service.Service) error {
	ctx := datastore.GetContext()
	defer ctx.Metrics().Stop(ctx.Metrics().Start("dao.FindChildService"))
	svc, err := this.facade.FindChildService(ctx, request.ServiceID, request.ChildName)
	if err != nil {
		return err
	}

	if svc != nil {
		*service = *svc
	} else {
		glog.Warningf("unable to find child of service: %+v", service)
	}
	return nil
}

// start the provided service
func (this *ControlPlaneDao) StartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	*affected, err = this.facade.StartService(datastore.GetContext(), request)
	return err
}

// restart the provided service
func (this *ControlPlaneDao) RestartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	*affected, err = this.facade.RestartService(datastore.GetContext(), request)
	return err
}

// rebalance the provided service
func (this *ControlPlaneDao) RebalanceService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	*affected, err = this.facade.RebalanceService(datastore.GetContext(), request)
	return err
}

// stop the provided service
func (this *ControlPlaneDao) StopService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	*affected, err = this.facade.StopService(datastore.GetContext(), request)
	return err
}

// pause the provided service
func (this *ControlPlaneDao) PauseService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	*affected, err = this.facade.PauseService(datastore.GetContext(), request)
	return err
}

// WaitService waits for the given service IDs to reach a particular state
func (this *ControlPlaneDao) WaitService(request dao.WaitServiceRequest, _ *int) (err error) {
	return this.facade.WaitService(datastore.GetContext(), request.DesiredState, request.Timeout, request.Recursive, request.ServiceIDs...)
}

// assign an IP address to a service (and all its child services) containing non default AddressResourceConfig
func (this *ControlPlaneDao) AssignIPs(assignmentRequest addressassignment.AssignmentRequest, _ *int) error {
	return this.facade.AssignIPs(datastore.GetContext(), assignmentRequest)
}

func (this *ControlPlaneDao) DeployService(request dao.ServiceDeploymentRequest, serviceID *string) (err error) {
	*serviceID, err = this.facade.DeployService(datastore.GetContext(), request.PoolID, request.ParentID, request.Overwrite, request.Service)
	return
}
