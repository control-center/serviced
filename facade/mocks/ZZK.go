package mocks

import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/domain/applicationendpoint"
import "github.com/control-center/serviced/domain/host"
import "github.com/control-center/serviced/domain/pool"
import "github.com/control-center/serviced/domain/registry"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicestate"
import zkregistry "github.com/control-center/serviced/zzk/registry"
import zkservice2 "github.com/control-center/serviced/zzk/service2"

type ZZK struct {
	mock.Mock
}

func (_m *ZZK) UpdateService(svc *service.Service, setLockOnCreate bool, setLockOnUpdate bool) error {
	ret := _m.Called(svc, setLockOnCreate, setLockOnUpdate)

	var r0 error
	if rf, ok := ret.Get(0).(func(*service.Service, bool, bool) error); ok {
		r0 = rf(svc, setLockOnCreate, setLockOnUpdate)
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
func (_m *ZZK) UpdateServiceState(poolID string, state *servicestate.ServiceState) error {
	ret := _m.Called(poolID, state)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *servicestate.ServiceState) error); ok {
		r0 = rf(poolID, state)
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
func (_m *ZZK) CheckRunningPublicEndpoint(publicendpoint zkregistry.PublicEndpointKey, serviceID string) error {
	ret := _m.Called(publicendpoint, serviceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(zkregistry.PublicEndpointKey, string) error); ok {
		r0 = rf(publicendpoint, serviceID)
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
func (_m *ZZK) DeleteRegistryLibrary(tenantID string) error {
	ret := _m.Called(tenantID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(tenantID)
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
func (_m *ZZK) GetServiceEndpoints(tenantID string, serviceID string, endpoints *[]applicationendpoint.ApplicationEndpoint) error {
	ret := _m.Called(tenantID, serviceID, endpoints)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, *[]applicationendpoint.ApplicationEndpoint) error); ok {
		r0 = rf(tenantID, serviceID, endpoints)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) GetServiceStates2(poolID string, serviceID string) ([]zkservice2.State, error) {
	ret := _m.Called(poolID, serviceID)

	var r0 []zkservice2.State
	if rf, ok := ret.Get(0).(func(string, string) []zkservice2.State); ok {
		r0 = rf(poolID, serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]zkservice2.State)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(poolID, serviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ZZK) GetHostStates(poolID string, hostID string) ([]zkservice2.State, error) {
	ret := _m.Called(poolID, hostID)

	var r0 []zkservice2.State
	if rf, ok := ret.Get(0).(func(string, string) []zkservice2.State); ok {
		r0 = rf(poolID, hostID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]zkservice2.State)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(poolID, hostID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ZZK) GetServiceState(poolID string, serviceID string, instanceID int) (*zkservice2.State, error) {
	ret := _m.Called(poolID, serviceID, instanceID)

	var r0 *zkservice2.State
	if rf, ok := ret.Get(0).(func(string, string, int) *zkservice2.State); ok {
		r0 = rf(poolID, serviceID, instanceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*zkservice2.State)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, int) error); ok {
		r1 = rf(poolID, serviceID, instanceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ZZK) StopServiceInstance2(poolID string, serviceID string, instanceID int) error {
	ret := _m.Called(poolID, serviceID, instanceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, int) error); ok {
		r0 = rf(poolID, serviceID, instanceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) StopServiceInstances(poolID string, serviceID string) error {
	ret := _m.Called(poolID, serviceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(poolID, serviceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ZZK) SendDockerAction(poolID string, serviceID string, instanceID int, command string, args []string) error {
	ret := _m.Called(poolID, serviceID, instanceID, command, args)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, int, string, []string) error); ok {
		r0 = rf(poolID, serviceID, instanceID, command, args)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
