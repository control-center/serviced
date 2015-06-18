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
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/facade"
	"github.com/zenoss/glog"
)

// AddService add a service. Return error if service already exists
func (this *ControlPlaneDao) AddService(svc service.Service, serviceId *string) error {
	if err := this.facade.AddService(datastore.Get(), svc, false, true); err != nil {
		return err
	}

	this.createTenantVolume(svc.ID)
	*serviceId = svc.ID
	return nil
}

// CloneService clones a service.  Return error if given serviceID is not found
func (this *ControlPlaneDao) CloneService(request dao.ServiceCloneRequest, clonedServiceId *string) error {
	svc, err := this.facade.GetService(datastore.Get(), request.ServiceID)
	if err != nil {
		glog.Errorf("ControlPlaneDao.CloneService: unable to find service id %+v: %s", request.ServiceID, err)
		return err
	}

	cloned, err := service.CloneService(svc, request.Suffix)
	if err != nil {
		glog.Errorf("ControlPlaneDao.CloneService: unable to rename service %+v %v: %s", svc.ID, svc.Name, err)
		return err
	}

	if err := this.AddService(*cloned, clonedServiceId); err != nil {
		return err
	}

	return nil
}

//
func (this *ControlPlaneDao) UpdateService(svc service.Service, unused *int) error {
	if err := this.facade.UpdateService(datastore.Get(), svc, false); err != nil {
		return err
	}

	this.createTenantVolume(svc.ID)
	return nil
}

//
func (this *ControlPlaneDao) RunMigrationScript(request dao.RunMigrationScriptRequest, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RunMigrationScript: start migration for service id %+v", request.ServiceID)
	if err := this.facade.RunMigrationScript(datastore.Get(), request); err != nil {
		glog.Errorf("ControlPlaneDao.RunMigrationScript: migration failed for id %+v: %s", request.ServiceID, err)
		return err
	}

	glog.Infof("ControlPlaneDao.RunMigrationScript: migrated service %+v (dry-run=%v)", request.ServiceID, request.DryRun)
	if !request.DryRun {
		this.createTenantVolume(request.ServiceID)
	}
	return nil
}

//
func (this *ControlPlaneDao) MigrateServices(request dao.ServiceMigrationRequest, unused *int) error {
	if err := this.facade.MigrateServices(datastore.Get(), request); err != nil {
		return err
	}
	if !request.DryRun {
		this.createTenantVolume(request.ServiceID)
	}
	return nil
}

func (this *ControlPlaneDao) GetServiceList(serviceID string, services *[]service.Service) error {
	var err error
	_, *services, err = this.facade.GetServicesByTenant(datastore.Get(), serviceID)
	return err
}

//
func (this *ControlPlaneDao) RemoveService(id string, unused *int) error {
	if err := this.facade.RemoveService(datastore.Get(), id); err != nil {
		return err
	} else if err := this.DeleteSnapshots(id, unused); err != nil {
		return err
	}
	return nil
}

// GetService gets a service.
func (this *ControlPlaneDao) GetService(id string, myService *service.Service) error {
	svc, err := this.facade.GetService(datastore.Get(), id)
	if svc != nil {
		*myService = *svc
	}
	return err
}

// Get the services (can filter by name and/or tenantID)
func (this *ControlPlaneDao) GetServices(request dao.ServiceRequest, services *[]service.Service) error {
	var filters []facade.FilterService
	var err error
	if request.UpdatedSince != 0 {
		filters = append(filters, facade.FilterServiceSince(time.Now().Add(-request.UpdatedSince)))
	}
	if request.NameRegex != "" {
		if filter, err := facade.FilterServiceByName(request.NameRegex); err != nil {
			return err
		} else {
			filters = append(filters, filter)
		}
	}

	if request.TenantID == "" {
		_, *services, err = this.facade.GetServicesByTenant(datastore.Get(), request.TenantID, filters...)
	} else {
		_, *services, err = this.facade.GetAllServices(datastore.Get(), filters...)
	}
	return err
}

//
func (this *ControlPlaneDao) FindChildService(request dao.FindChildRequest, service *service.Service) error {
	svc, err := this.facade.GetChildService(datastore.Get(), request.ServiceID, request.ChildName)
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

// Get tagged services (can also filter by name and/or tenantID)
func (this *ControlPlaneDao) GetTaggedServices(request dao.ServiceRequest, services *[]service.Service) error {
	var filters []facade.FilterService
	var err error

	if request.UpdatedSince != 0 {
		filters = append(filters, facade.FilterServiceSince(time.Now().Add(-request.UpdatedSince)))
	}
	if request.NameRegex != "" {
		if filter, err := facade.FilterServiceByName(request.NameRegex); err != nil {
			return err
		} else {
			filters = append(filters, filter)
		}
	}
	if request.TenantID != "" {
		filter := func(svc *service.Service) bool {
			tenantID, _ := this.facade.GetTenantID(datastore.Get(), svc.ID)
			return tenantID == request.TenantID
		}
		filters = append(filters, filter)
	}

	if request.Tags != nil && len(request.Tags) > 0 {
		_, *services, err = this.facade.GetServicesByTenant(datastore.Get(), request.TenantID, filters...)
	} else {
		_, *services, err = this.facade.GetAllServices(datastore.Get(), filters...)
	}
	return err
}

// The tenant id is the root service uuid. Walk the service tree to root to find the tenant id.
func (this *ControlPlaneDao) GetTenantId(serviceID string, tenantId *string) error {
	if tid, err := this.facade.GetTenantID(datastore.Get(), serviceID); err == nil {
		*tenantId = tid
		return nil
	} else {
		return err
	}
}

// start the provided service
func (this *ControlPlaneDao) StartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	*affected, err = this.facade.StartService(datastore.Get(), request.ServiceID, request.AutoLaunch)
	return err
}

// restart the provided service
func (this *ControlPlaneDao) RestartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	*affected, err = this.facade.RestartService(datastore.Get(), request.ServiceID, request.AutoLaunch)
	return err
}

// stop the provided service
func (this *ControlPlaneDao) StopService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	*affected, err = this.facade.StopService(datastore.Get(), request.ServiceID, request.AutoLaunch)
	return err
}

// WaitService waits for the given service IDs to reach a particular state
func (this *ControlPlaneDao) WaitService(request dao.WaitServiceRequest, _ *struct{}) (err error) {
	return this.facade.WaitService(datastore.Get(), request.DesiredState, request.Timeout, request.ServiceIDs...)
}

// assign an IP address to a service (and all its child services) containing non default AddressResourceConfig
func (this *ControlPlaneDao) AssignIPs(assignmentRequest dao.AssignmentRequest, _ *struct{}) error {
	return this.facade.AssignIPs(datastore.Get(), assignmentRequest.ServiceID, assignmentRequest.IPAddress)
}

// Create the tenant volume
func (this *ControlPlaneDao) createTenantVolume(serviceID string) {
	if tenantID, err := this.facade.GetTenantID(datastore.Get(), serviceID); err != nil {
		glog.Warningf("Could not get tenant for service %s: %s", serviceID, err)
	} else if _, err := this.dfs.GetVolume(tenantID); err != nil {
		glog.Warningf("Could not create volume for tenant %s: %s", tenantID, err)
	}
}
