// Copyright 2016 The Serviced Authors.
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

import "github.com/control-center/serviced/domain/service"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/datastore"

import "time"

type Store struct {
	mock.Mock
}

func (_m *Store) Put(ctx datastore.Context, svc *service.Service) error {
	ret := _m.Called(ctx, svc)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *service.Service) error); ok {
		r0 = rf(ctx, svc)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Store) Get(ctx datastore.Context, id string) (*service.Service, error) {
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
func (_m *Store) Delete(ctx datastore.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Store) GetServices(ctx datastore.Context) ([]service.Service, error) {
	ret := _m.Called(ctx)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context) []service.Service); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
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
func (_m *Store) GetUpdatedServices(ctx datastore.Context, since time.Duration) ([]service.Service, error) {
	ret := _m.Called(ctx, since)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, time.Duration) []service.Service); ok {
		r0 = rf(ctx, since)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, time.Duration) error); ok {
		r1 = rf(ctx, since)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetTaggedServices(ctx datastore.Context, tags ...string) ([]service.Service, error) {
	ret := _m.Called(ctx, tags)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, ...string) []service.Service); ok {
		r0 = rf(ctx, tags...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, ...string) error); ok {
		r1 = rf(ctx, tags...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetServicesByPool(ctx datastore.Context, poolID string) ([]service.Service, error) {
	ret := _m.Called(ctx, poolID)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []service.Service); ok {
		r0 = rf(ctx, poolID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, poolID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetServicesByDeployment(ctx datastore.Context, deploymentID string) ([]service.Service, error) {
	ret := _m.Called(ctx, deploymentID)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []service.Service); ok {
		r0 = rf(ctx, deploymentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetChildServices(ctx datastore.Context, parentID string) ([]service.Service, error) {
	ret := _m.Called(ctx, parentID)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []service.Service); ok {
		r0 = rf(ctx, parentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, parentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) FindChildService(ctx datastore.Context, deploymentID string, parentID string, serviceName string) (*service.Service, error) {
	ret := _m.Called(ctx, deploymentID, parentID, serviceName)

	var r0 *service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string) *service.Service); ok {
		r0 = rf(ctx, deploymentID, parentID, serviceName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string, string) error); ok {
		r1 = rf(ctx, deploymentID, parentID, serviceName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) FindTenantByDeploymentID(ctx datastore.Context, deploymentID string, name string) (*service.Service, error) {
	ret := _m.Called(ctx, deploymentID, name)

	var r0 *service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) *service.Service); ok {
		r0 = rf(ctx, deploymentID, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string) error); ok {
		r1 = rf(ctx, deploymentID, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
