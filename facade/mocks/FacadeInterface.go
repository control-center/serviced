// Copyright 2015 The Serviced Authors.
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

package mocks

import (
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/health"
	"github.com/stretchr/testify/mock"
)

type FacadeInterface struct {
	mock.Mock
}

func (m *FacadeInterface) AddService(ctx datastore.Context, svc service.Service) error {
	ret := m.Called(ctx, svc)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) GetService(ctx datastore.Context, id string) (*service.Service, error) {
	ret := m.Called(ctx, id)

	var r0 *service.Service
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Service)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) GetServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error) {
	ret := m.Called(ctx, request)

	var r0 []service.Service
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]service.Service)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) GetServicesByImage(ctx datastore.Context, imageID string) ([]service.Service, error) {
	ret := m.Called(ctx, imageID)

	var r0 []service.Service
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]service.Service)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) GetServiceStates(ctx datastore.Context, serviceID string) ([]servicestate.ServiceState, error) {
	ret := m.Called(ctx, serviceID)

	var r0 []servicestate.ServiceState
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]servicestate.ServiceState)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) GetTenantID(ctx datastore.Context, serviceID string) (string, error) {
	ret := m.Called(ctx, serviceID)

	r0 := ret.Get(0).(string)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) MigrateServices(ctx datastore.Context, request dao.ServiceMigrationRequest) error {
	ret := m.Called(ctx, request)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) RemoveService(ctx datastore.Context, id string) error {
	ret := m.Called(ctx, id)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) ScheduleService(ctx datastore.Context, serviceID string, autoLaunch bool, desiredState service.DesiredState) (int, error) {
	ret := m.Called(ctx, serviceID, autoLaunch, desiredState)

	r0 := ret.Get(0).(int)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) UpdateService(ctx datastore.Context, svc service.Service) error {
	ret := m.Called(ctx, svc)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, recursive bool, serviceIDs ...string) error {
	ret := m.Called(ctx, dstate, timeout, recursive, serviceIDs)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface)AssignIPs(ctx datastore.Context, assignmentRequest addressassignment.AssignmentRequest) (err error) {
	ret := m.Called(ctx, assignmentRequest)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) AddServiceTemplate(ctx datastore.Context, serviceTemplate servicetemplate.ServiceTemplate) (string, error) {
	ret := m.Called(ctx, serviceTemplate)

	r0 := ret.Get(0).(string)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) GetServiceTemplates(ctx datastore.Context) (map[string]servicetemplate.ServiceTemplate, error) {
	ret := m.Called(ctx)

	var r0 map[string]servicetemplate.ServiceTemplate
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(map[string]servicetemplate.ServiceTemplate)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) RemoveServiceTemplate(ctx datastore.Context, templateID string) error {
	ret := m.Called(ctx, templateID)

	return ret.Error(0)
}
func (m *FacadeInterface) UpdateServiceTemplate(ctx datastore.Context, template servicetemplate.ServiceTemplate) error {
	ret := m.Called(ctx, template)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) DeployTemplate(ctx datastore.Context, poolID string, templateID string, deploymentID string) ([]string, error) {
	ret := m.Called(ctx, poolID, templateID, deploymentID)

	var r0 []string
	if rf, ok := ret.Get(0).(func()[]string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *FacadeInterface) DeployTemplateActive() (active []map[string]string, err error) {
	ret := m.Called()

	var r0 []map[string]string
	if rf, ok := ret.Get(0).(func()[]map[string]string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]map[string]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *FacadeInterface) DeployTemplateStatus(deploymentID string) (status string, err error) {
	ret := m.Called(deploymentID)

	var r0 string
	if rf, ok := ret.Get(0).(func()string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *FacadeInterface) AddHost(ctx datastore.Context, entity *host.Host) error {
	ret := m.Called(ctx, entity)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) GetHost(ctx datastore.Context, hostID string) (*host.Host, error) {
	ret := m.Called(ctx, hostID)

	var r0 *host.Host
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*host.Host)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) GetHosts(ctx datastore.Context) ([]host.Host, error) {
	ret := m.Called(ctx)

	var r0 []host.Host
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]host.Host)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m* FacadeInterface) GetActiveHostIDs(ctx datastore.Context) ([]string, error) {
	ret := m.Called(ctx)

	var r0 []string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m* FacadeInterface) UpdateHost(ctx datastore.Context, entity *host.Host) error {
	ret := m.Called(ctx, entity)

	r0 := ret.Error(0)

	return r0
}
func (m* FacadeInterface) RemoveHost(ctx datastore.Context, hostID string) error {
	ret := m.Called(ctx, hostID)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) FindHostsInPool(ctx datastore.Context, poolID string) ([]host.Host, error) {
	ret := m.Called(ctx, poolID)

	var r0 []host.Host
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]host.Host)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	ret := m.Called(ctx, entity)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) GetResourcePool(ctx datastore.Context, poolID string) (*pool.ResourcePool, error) {
	ret := m.Called(ctx, poolID)

	var r0 *pool.ResourcePool
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*pool.ResourcePool)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) GetResourcePools(ctx datastore.Context) ([]pool.ResourcePool, error) {
	ret := m.Called(ctx)

	var r0 []pool.ResourcePool
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]pool.ResourcePool)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) GetPoolIPs(ctx datastore.Context, poolID string) (*pool.PoolIPs, error) {
	ret := m.Called(ctx, poolID)

	var r0 *pool.PoolIPs
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*pool.PoolIPs)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) HasIP(ctx datastore.Context, poolID string, ipAddr string) (bool, error) {
	ret := m.Called(ctx, poolID, ipAddr)

	r0 := ret.Get(0).(bool)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) RemoveResourcePool(ctx datastore.Context, id string) error {
	ret := m.Called(ctx, id)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) UpdateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	ret := m.Called(ctx, entity)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) GetHealthChecksForService(ctx datastore.Context, id string) (map[string]health.HealthCheck, error) {
	ret := m.Called(ctx, id)

	var r0 map[string]health.HealthCheck
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(map[string]health.HealthCheck)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (_m *FacadeInterface) UpgradeRegistry(ctx datastore.Context, fromRegistryHost string, force bool) error {
	ret := _m.Called(ctx, fromRegistryHost, force)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, bool) error); ok {
		r0 = rf(ctx, fromRegistryHost, force)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
