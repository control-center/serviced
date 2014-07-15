// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/service"
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

//
func (this *ControlPlaneDao) GetService(id string, myService *service.Service) error {
	if svc, err := this.facade.GetService(datastore.Get(), id); err == nil {
		*myService = *svc
		return nil
	} else {
		return err
	}
}

//
func (this *ControlPlaneDao) GetServices(request dao.EntityRequest, services *[]*service.Service) error {
	if svcs, err := this.facade.GetServices(datastore.Get(), request); err == nil {
		*services = svcs
		return nil
	} else {
		return err
	}
}

//
func (this *ControlPlaneDao) FindChildService(request dao.FindChildRequest, service *service.Service) error {
	if svc, err := this.facade.FindChildService(datastore.Get(), request.ServiceID, request.ChildName); err == nil {
		*service = *svc
		return nil
	} else {
		return err
	}
}

//
func (this *ControlPlaneDao) GetTaggedServices(request dao.EntityRequest, services *[]*service.Service) error {
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
func (this *ControlPlaneDao) GetServiceEndpoints(serviceID string, response *map[string][]*dao.ApplicationEndpoint) (err error) {
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
func (this *ControlPlaneDao) StopService(id string, unused *int) error {
	return this.facade.StopService(datastore.Get(), id)
}

// assign an IP address to a service (and all its child services) containing non default AddressResourceConfig
func (this *ControlPlaneDao) AssignIPs(assignmentRequest dao.AssignmentRequest, _ *struct{}) error {
	return this.facade.AssignIPs(datastore.Get(), assignmentRequest)
}
