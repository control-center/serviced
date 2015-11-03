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

import "github.com/control-center/serviced/domain/applicationendpoint"
import "github.com/control-center/serviced/domain/host"
import "github.com/control-center/serviced/domain/pool"
import "github.com/control-center/serviced/domain/registry"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicestate"

type ZZK struct {
	mock.Mock
}

func (_m *ZZK) UpdateService(svc *service.Service, locked bool) error {
	ret := _m.Called(svc, locked)

	var r0 error
	if rf, ok := ret.Get(0).(func(*service.Service, bool) error); ok {
		r0 = rf(svc, locked)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) RemoveService(svc *service.Service) error {
	ret := _m.Called(svc)

	var r0 error
	if rf, ok := ret.Get(0).(func(*service.Service) error); ok {
		r0 = rf(svc)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) WaitService(svc *service.Service, state service.DesiredState, cancel <-chan interface{}) error {
	ret := _m.Called(svc, state, cancel)

	var r0 error
	if rf, ok := ret.Get(0).(func(*service.Service, service.DesiredState, <-chan interface{}) error); ok {
		r0 = rf(svc, state, cancel)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) GetServiceStates(poolID string, states *[]servicestate.ServiceState, serviceIDs ...string) error {
	ret := _m.Called(poolID, states, serviceIDs)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *[]servicestate.ServiceState, ...string) error); ok {
		r0 = rf(poolID, states, serviceIDs...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) StopServiceInstance(poolID string, hostID string, stateID string) error {
	ret := _m.Called(poolID, hostID, stateID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(poolID, hostID, stateID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) CheckRunningVHost(vhost string, serviceID string) error {
	ret := _m.Called(vhost, serviceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(vhost, serviceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) AddHost(_host *host.Host) error {
	ret := _m.Called(_host)

	var r0 error
	if rf, ok := ret.Get(0).(func(*host.Host) error); ok {
		r0 = rf(_host)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) UpdateHost(_host *host.Host) error {
	ret := _m.Called(_host)

	var r0 error
	if rf, ok := ret.Get(0).(func(*host.Host) error); ok {
		r0 = rf(_host)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) RemoveHost(_host *host.Host) error {
	ret := _m.Called(_host)

	var r0 error
	if rf, ok := ret.Get(0).(func(*host.Host) error); ok {
		r0 = rf(_host)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) GetActiveHosts(poolID string, hosts *[]string) error {
	ret := _m.Called(poolID, hosts)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *[]string) error); ok {
		r0 = rf(poolID, hosts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) AddResourcePool(_pool *pool.ResourcePool) error {
	ret := _m.Called(_pool)

	var r0 error
	if rf, ok := ret.Get(0).(func(*pool.ResourcePool) error); ok {
		r0 = rf(_pool)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) UpdateResourcePool(_pool *pool.ResourcePool) error {
	ret := _m.Called(_pool)

	var r0 error
	if rf, ok := ret.Get(0).(func(*pool.ResourcePool) error); ok {
		r0 = rf(_pool)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) RemoveResourcePool(poolID string) error {
	ret := _m.Called(poolID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(poolID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) AddVirtualIP(vip *pool.VirtualIP) error {
	ret := _m.Called(vip)

	var r0 error
	if rf, ok := ret.Get(0).(func(*pool.VirtualIP) error); ok {
		r0 = rf(vip)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) RemoveVirtualIP(vip *pool.VirtualIP) error {
	ret := _m.Called(vip)

	var r0 error
	if rf, ok := ret.Get(0).(func(*pool.VirtualIP) error); ok {
		r0 = rf(vip)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) GetRegistryImage(id string) (*registry.Image, error) {
	ret := _m.Called(id)

	var r0 *registry.Image
	if rf, ok := ret.Get(0).(func(string) *registry.Image); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*registry.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ZZK) SetRegistryImage(rImage *registry.Image) error {
	ret := _m.Called(rImage)

	var r0 error
	if rf, ok := ret.Get(0).(func(*registry.Image) error); ok {
		r0 = rf(rImage)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) DeleteRegistryImage(id string) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) DeleteRegistryLibrary(id string) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) LockServices(svcs []service.Service) error {
	ret := _m.Called(svcs)

	var r0 error
	if rf, ok := ret.Get(0).(func([]service.Service) error); ok {
		r0 = rf(svcs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) UnlockServices(svcs []service.Service) error {
	ret := _m.Called(svcs)

	var r0 error
	if rf, ok := ret.Get(0).(func([]service.Service) error); ok {
		r0 = rf(svcs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *ZZK) GetServiceEndpoints(tenantID, serviceID string, endpoints *[]applicationendpoint.ApplicationEndpoint) error {
	ret := _m.Called(tenantID, serviceID, endpoints)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, *[]applicationendpoint.ApplicationEndpoint) error); ok {
		r0 = rf(tenantID, serviceID, endpoints)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
