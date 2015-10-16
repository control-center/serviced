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
import "github.com/control-center/serviced/domain/host"
import "github.com/control-center/serviced/domain/pool"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicestate"
import "github.com/control-center/serviced/domain/servicetemplate"

type FacadeInterface struct {
	mock.Mock
}

func (_m *FacadeInterface) AddService(ctx datastore.Context, svc service.Service) error {
	ret := _m.Called(ctx, svc)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, service.Service) error); ok {
		r0 = rf(ctx, svc)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) GetService(ctx datastore.Context, id string) (*service.Service, error) {
	ret := _m.Called(ctx, id)

	var r0 *service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *service.Service); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error) {
	ret := _m.Called(ctx, request)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, dao.EntityRequest) []service.Service); ok {
		r0 = rf(ctx, request)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, dao.EntityRequest) error); ok {
		r1 = rf(ctx, request)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetServiceStates(ctx datastore.Context, serviceID string) ([]servicestate.ServiceState, error) {
	ret := _m.Called(ctx, serviceID)

	var r0 []servicestate.ServiceState
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []servicestate.ServiceState); ok {
		r0 = rf(ctx, serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]servicestate.ServiceState)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, serviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetTenantID(ctx datastore.Context, serviceID string) (string, error) {
	ret := _m.Called(ctx, serviceID)

	var r0 string
	if rf, ok := ret.Get(0).(func(datastore.Context, string) string); ok {
		r0 = rf(ctx, serviceID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, serviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) RunMigrationScript(ctx datastore.Context, request dao.RunMigrationScriptRequest) error {
	ret := _m.Called(ctx, request)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, dao.RunMigrationScriptRequest) error); ok {
		r0 = rf(ctx, request)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) MigrateServices(ctx datastore.Context, request dao.ServiceMigrationRequest) error {
	ret := _m.Called(ctx, request)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, dao.ServiceMigrationRequest) error); ok {
		r0 = rf(ctx, request)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) RemoveService(ctx datastore.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) RestoreIPs(ctx datastore.Context, svc service.Service) error {
	ret := _m.Called(ctx, svc)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, service.Service) error); ok {
		r0 = rf(ctx, svc)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) ScheduleService(ctx datastore.Context, serviceID string, autoLaunch bool, desiredState service.DesiredState) (int, error) {
	ret := _m.Called(ctx, serviceID, autoLaunch, desiredState)

	var r0 int
	if rf, ok := ret.Get(0).(func(datastore.Context, string, bool, service.DesiredState) int); ok {
		r0 = rf(ctx, serviceID, autoLaunch, desiredState)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, bool, service.DesiredState) error); ok {
		r1 = rf(ctx, serviceID, autoLaunch, desiredState)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) UpdateService(ctx datastore.Context, svc service.Service) error {
	ret := _m.Called(ctx, svc)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, service.Service) error); ok {
		r0 = rf(ctx, svc)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, serviceIDs ...string) error {
	ret := _m.Called(ctx, dstate, timeout, serviceIDs)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, service.DesiredState, time.Duration, ...string) error); ok {
		r0 = rf(ctx, dstate, timeout, serviceIDs...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) GetServiceTemplates(ctx datastore.Context) (map[string]servicetemplate.ServiceTemplate, error) {
	ret := _m.Called(ctx)

	var r0 map[string]servicetemplate.ServiceTemplate
	if rf, ok := ret.Get(0).(func(datastore.Context) map[string]servicetemplate.ServiceTemplate); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]servicetemplate.ServiceTemplate)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) UpdateServiceTemplate(ctx datastore.Context, template servicetemplate.ServiceTemplate) error {
	ret := _m.Called(ctx, template)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, servicetemplate.ServiceTemplate) error); ok {
		r0 = rf(ctx, template)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) AddHost(ctx datastore.Context, entity *host.Host) error {
	ret := _m.Called(ctx, entity)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *host.Host) error); ok {
		r0 = rf(ctx, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) GetHosts(ctx datastore.Context) ([]host.Host, error) {
	ret := _m.Called(ctx)

	var r0 []host.Host
	if rf, ok := ret.Get(0).(func(datastore.Context) []host.Host); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]host.Host)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	ret := _m.Called(ctx, entity)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *pool.ResourcePool) error); ok {
		r0 = rf(ctx, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) GetResourcePools(ctx datastore.Context) ([]pool.ResourcePool, error) {
	ret := _m.Called(ctx)

	var r0 []pool.ResourcePool
	if rf, ok := ret.Get(0).(func(datastore.Context) []pool.ResourcePool); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pool.ResourcePool)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) HasIP(ctx datastore.Context, poolID string, ipAddr string) (bool, error) {
	ret := _m.Called(ctx, poolID, ipAddr)

	var r0 bool
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) bool); ok {
		r0 = rf(ctx, poolID, ipAddr)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string) error); ok {
		r1 = rf(ctx, poolID, ipAddr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) UpdateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	ret := _m.Called(ctx, entity)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *pool.ResourcePool) error); ok {
		r0 = rf(ctx, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
