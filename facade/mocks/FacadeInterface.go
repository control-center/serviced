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

import "github.com/stretchr/testify/mock"

import "time"
import "github.com/control-center/serviced/dao"
import "github.com/control-center/serviced/datastore"
import "github.com/control-center/serviced/domain"
import "github.com/control-center/serviced/domain/host"
import "github.com/control-center/serviced/domain/pool"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicestate"
import "github.com/control-center/serviced/domain/servicetemplate"

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
func (m *FacadeInterface) RunMigrationScript(ctx datastore.Context, request dao.RunMigrationScriptRequest) error {
	ret := m.Called(ctx, request)

	r0 := ret.Error(0)

	return r0
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
func (m *FacadeInterface) WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, serviceIDs ...string) error {
	ret := m.Called(ctx, dstate, timeout, serviceIDs)

	r0 := ret.Error(0)

	return r0
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
func (m *FacadeInterface) UpdateServiceTemplate(ctx datastore.Context, template servicetemplate.ServiceTemplate) error {
	ret := m.Called(ctx, template)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) AddHost(ctx datastore.Context, entity *host.Host) error {
	ret := m.Called(ctx, entity)

	r0 := ret.Error(0)

	return r0
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
func (m *FacadeInterface) AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	ret := m.Called(ctx, entity)

	r0 := ret.Error(0)

	return r0
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
func (m *FacadeInterface) HasIP(ctx datastore.Context, poolID string, ipAddr string) (bool, error) {
	ret := m.Called(ctx, poolID, ipAddr)

	r0 := ret.Get(0).(bool)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *FacadeInterface) UpdateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	ret := m.Called(ctx, entity)

	r0 := ret.Error(0)

	return r0
}
func (m *FacadeInterface) GetHealthChecksForService(ctx datastore.Context, id string) (map[string]domain.HealthCheck, error) {
	ret := m.Called(ctx, id)

	var r0 map[string]domain.HealthCheck
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(map[string]domain.HealthCheck)
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
