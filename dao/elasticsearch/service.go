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
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

// AddService add a service. Return error if service already exists
func (this *ControlPlaneDao) AddService(svc service.Service, serviceId *string) error {
	if err := this.facade.AddService(datastore.Get(), svc); err != nil {
		return err
	}
	*serviceId = svc.ID
	return nil
}

//
func (this *ControlPlaneDao) UpdateService(svc service.Service, unused *int) error {
	if err := this.facade.UpdateService(datastore.Get(), svc); err != nil {
		return err
	}
	return nil
}

//
func (this *ControlPlaneDao) RemoveService(id string, unused *int) error {
	if err := this.facade.RemoveService(datastore.Get(), id); err != nil {
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
	if svcs, err := this.facade.GetServices(datastore.Get(), request); err == nil {
		*services = svcs
		return nil
	} else {
		return err
	}
}

//
func (this *ControlPlaneDao) FindChildService(request dao.FindChildRequest, service *service.Service) error {
	svc, err := this.facade.FindChildService(datastore.Get(), request.ServiceID, request.ChildName)
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
	if svcs, err := this.facade.GetTaggedServices(datastore.Get(), request); err == nil {
		*services = svcs
		return nil
	} else {
		return err
	}
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

// Get a service endpoint.
func (this *ControlPlaneDao) GetServiceEndpoints(serviceID string, response *map[string][]dao.ApplicationEndpoint) (err error) {
	if result, err := this.facade.GetServiceEndpoints(datastore.Get(), serviceID); err == nil {
		*response = result
		return nil
	} else {
		return err
	}
}

// start the provided service
func (this *ControlPlaneDao) StartService(serviceID string, unused *string) error {
	return this.facade.StartService(datastore.Get(), serviceID)
}

// stop the provided service
func (this *ControlPlaneDao) StopService(id string, unused *int) error {
	return this.facade.StopService(datastore.Get(), id)
}

// assign an IP address to a service (and all its child services) containing non default AddressResourceConfig
func (this *ControlPlaneDao) AssignIPs(assignmentRequest dao.AssignmentRequest, _ *struct{}) error {
	return this.facade.AssignIPs(datastore.Get(), assignmentRequest)
}
