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

package test

import (
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/facade"

	"github.com/stretchr/testify/mock"
)

// assert the interface
var _ facade.FacadeInterface = &MockFacade{}

type MockFacade struct {
	mock.Mock
}

func (mf *MockFacade) AddService(ctx datastore.Context, svc service.Service) error {
	return mf.Mock.Called(ctx, svc).Error(0)
}

func (mf *MockFacade) GetService(ctx datastore.Context, id string) (*service.Service, error) {
	args := mf.Mock.Called(ctx, id)

	var svc *service.Service
	if arg0 := args.Get(0); arg0 != nil {
		svc = arg0.(*service.Service)
	}
	return svc, args.Error(1)
}

func (mf *MockFacade) GetServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error) {
	args := mf.Mock.Called(ctx, request)

	var services []service.Service
	if arg0 := args.Get(0); arg0 != nil {
		services = arg0.([]service.Service)
	}
	return services, args.Error(1)
}

func (mf *MockFacade) GetServiceStates(ctx datastore.Context, serviceID string) ([]servicestate.ServiceState, error) {
	args := mf.Mock.Called(ctx, serviceID)
	return args.Get(0).([]servicestate.ServiceState), args.Error(1)
}

func (mf *MockFacade) GetTenantID(ctx datastore.Context, serviceID string) (string, error) {
	args := mf.Mock.Called(ctx, serviceID)
	return args.String(0), args.Error(1)
}

func (mf *MockFacade) MigrateService(ctx datastore.Context, request dao.ServiceMigrationRequest) error {
	return mf.Mock.Called(ctx, request).Error(0)
}

func (mf *MockFacade) RemoveService(ctx datastore.Context, id string) error {
	return mf.Mock.Called(ctx, id).Error(0)
}

func (mf *MockFacade) RestoreIPs(ctx datastore.Context, svc service.Service) error {
	return mf.Mock.Called(ctx, svc).Error(0)
}

func (mf *MockFacade) ScheduleService(ctx datastore.Context, serviceID string, autoLaunch bool, desiredState service.DesiredState) (int, error) {
	args := mf.Mock.Called(ctx, serviceID, autoLaunch, desiredState)
	return args.Int(0), args.Error(1)
}

func (mf *MockFacade) UpdateService(ctx datastore.Context, svc service.Service) error {
	return mf.Mock.Called(ctx, svc).Error(0)
}

func (mf *MockFacade) WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, serviceIDs ...string) error {
	return mf.Mock.Called(ctx, dstate, timeout, serviceIDs).Error(0)
}

func (mf *MockFacade) GetServiceTemplates(ctx datastore.Context) (map[string]servicetemplate.ServiceTemplate, error) {
	args := mf.Mock.Called(ctx)
	return args.Get(0).(map[string]servicetemplate.ServiceTemplate), args.Error(1)
}

func (mf *MockFacade) UpdateServiceTemplate(ctx datastore.Context, template servicetemplate.ServiceTemplate) error {
	return mf.Mock.Called(ctx, template).Error(0)
}

func (mf *MockFacade) AddHost(ctx datastore.Context, entity *host.Host) error {
	return mf.Mock.Called(ctx, entity).Error(0)
}

func (mf *MockFacade) GetHosts(ctx datastore.Context) ([]host.Host, error) {
	args := mf.Mock.Called(ctx)
	return args.Get(0).([]host.Host), args.Error(1)
}

func (mf *MockFacade) AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	return mf.Mock.Called(ctx, entity).Error(0)
}

func (mf *MockFacade) GetResourcePools(ctx datastore.Context) ([]pool.ResourcePool, error) {
	args := mf.Mock.Called(ctx)
	return args.Get(0).([]pool.ResourcePool), args.Error(1)
}

func (mf *MockFacade) HasIP(ctx datastore.Context, poolID string, ipAddr string) (bool, error) {
	args := mf.Mock.Called(ctx, poolID, ipAddr)
	return args.Bool(0), args.Error(1)
}

func (mf *MockFacade) UpdateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	return mf.Mock.Called(ctx, entity).Error(0)
}
