package mocks

import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/domain/applicationendpoint"
import "github.com/control-center/serviced/domain/host"
import "github.com/control-center/serviced/domain/pool"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicetemplate"
import "github.com/control-center/serviced/volume"
import "time"

type ClientInterface struct {
	mock.Mock
}

func (_m *ClientInterface) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) GetHost(hostID string) (*host.Host, error) {
	ret := _m.Called(hostID)

	var r0 *host.Host
	if rf, ok := ret.Get(0).(func(string) *host.Host); ok {
		r0 = rf(hostID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*host.Host)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(hostID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ClientInterface) GetHosts() ([]host.Host, error) {
	ret := _m.Called()

	var r0 []host.Host
	if rf, ok := ret.Get(0).(func() []host.Host); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]host.Host)
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
func (_m *ClientInterface) GetActiveHostIDs() ([]string, error) {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
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
func (_m *ClientInterface) AddHost(targetHost host.Host) error {
	ret := _m.Called(targetHost)

	var r0 error
	if rf, ok := ret.Get(0).(func(host.Host) error); ok {
		r0 = rf(targetHost)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) UpdateHost(targetHost host.Host) error {
	ret := _m.Called(targetHost)

	var r0 error
	if rf, ok := ret.Get(0).(func(host.Host) error); ok {
		r0 = rf(targetHost)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) RemoveHost(hostID string) error {
	ret := _m.Called(hostID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(hostID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) FindHostsInPool(poolID string) ([]host.Host, error) {
	ret := _m.Called(poolID)

	var r0 []host.Host
	if rf, ok := ret.Get(0).(func(string) []host.Host); ok {
		r0 = rf(poolID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]host.Host)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(poolID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ClientInterface) GetResourcePool(poolID string) (*pool.ResourcePool, error) {
	ret := _m.Called(poolID)

	var r0 *pool.ResourcePool
	if rf, ok := ret.Get(0).(func(string) *pool.ResourcePool); ok {
		r0 = rf(poolID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pool.ResourcePool)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(poolID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ClientInterface) GetResourcePools() ([]pool.ResourcePool, error) {
	ret := _m.Called()

	var r0 []pool.ResourcePool
	if rf, ok := ret.Get(0).(func() []pool.ResourcePool); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pool.ResourcePool)
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
func (_m *ClientInterface) AddResourcePool(targetPool pool.ResourcePool) error {
	ret := _m.Called(targetPool)

	var r0 error
	if rf, ok := ret.Get(0).(func(pool.ResourcePool) error); ok {
		r0 = rf(targetPool)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) UpdateResourcePool(targetPool pool.ResourcePool) error {
	ret := _m.Called(targetPool)

	var r0 error
	if rf, ok := ret.Get(0).(func(pool.ResourcePool) error); ok {
		r0 = rf(targetPool)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) RemoveResourcePool(poolID string) error {
	ret := _m.Called(poolID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(poolID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) GetPoolIPs(poolID string) (*pool.PoolIPs, error) {
	ret := _m.Called(poolID)

	var r0 *pool.PoolIPs
	if rf, ok := ret.Get(0).(func(string) *pool.PoolIPs); ok {
		r0 = rf(poolID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pool.PoolIPs)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(poolID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ClientInterface) AddVirtualIP(requestVirtualIP pool.VirtualIP) error {
	ret := _m.Called(requestVirtualIP)

	var r0 error
	if rf, ok := ret.Get(0).(func(pool.VirtualIP) error); ok {
		r0 = rf(requestVirtualIP)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) RemoveVirtualIP(requestVirtualIP pool.VirtualIP) error {
	ret := _m.Called(requestVirtualIP)

	var r0 error
	if rf, ok := ret.Get(0).(func(pool.VirtualIP) error); ok {
		r0 = rf(requestVirtualIP)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) ServiceUse(serviceID string, imageID string, registry string, replaceImgs []string, noOp bool) (string, error) {
	ret := _m.Called(serviceID, imageID, registry, replaceImgs, noOp)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string, string, []string, bool) string); ok {
		r0 = rf(serviceID, imageID, registry, replaceImgs, noOp)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string, []string, bool) error); ok {
		r1 = rf(serviceID, imageID, registry, replaceImgs, noOp)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ClientInterface) WaitService(serviceIDs []string, state service.DesiredState, timeout time.Duration, recursive bool) error {
	ret := _m.Called(serviceIDs, state, timeout, recursive)

	var r0 error
	if rf, ok := ret.Get(0).(func([]string, service.DesiredState, time.Duration, bool) error); ok {
		r0 = rf(serviceIDs, state, timeout, recursive)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate) (templateID string, err error) {
	ret := _m.Called(serviceTemplate)

	var r0 string
	if rf, ok := ret.Get(0).(func(servicetemplate.ServiceTemplate) string); ok {
		r0 = rf(serviceTemplate)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(servicetemplate.ServiceTemplate) error); ok {
		r1 = rf(serviceTemplate)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ClientInterface) GetServiceTemplates() (serviceTemplates map[string]servicetemplate.ServiceTemplate, err error) {
	ret := _m.Called()

	var r0 map[string]servicetemplate.ServiceTemplate
	if rf, ok := ret.Get(0).(func() map[string]servicetemplate.ServiceTemplate); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]servicetemplate.ServiceTemplate)
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
func (_m *ClientInterface) RemoveServiceTemplate(templateID string) error {
	ret := _m.Called(templateID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(templateID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) DeployTemplate(request servicetemplate.ServiceTemplateDeploymentRequest) (tenantIDs []string, err error) {
	ret := _m.Called(request)

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
func (_m *ClientInterface) GetVolumeStatus() (*volume.Statuses, error) {
	ret := _m.Called()

	var r0 *volume.Statuses
	if rf, ok := ret.Get(0).(func() *volume.Statuses); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*volume.Statuses)
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
func (_m *ClientInterface) GetServiceEndpoints(serviceIDs []string, reportImports bool, reportExports bool, validate bool) ([]applicationendpoint.EndpointReport, error) {
	ret := _m.Called(serviceIDs, reportImports, reportExports, validate)

	var r0 []applicationendpoint.EndpointReport
	if rf, ok := ret.Get(0).(func([]string, bool, bool, bool) []applicationendpoint.EndpointReport); ok {
		r0 = rf(serviceIDs, reportImports, reportExports, validate)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]applicationendpoint.EndpointReport)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]string, bool, bool, bool) error); ok {
		r1 = rf(serviceIDs, reportImports, reportExports, validate)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ClientInterface) ResetRegistry() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) SyncRegistry() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) UpgradeRegistry(endpoint string, override bool) error {
	ret := _m.Called(endpoint, override)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, bool) error); ok {
		r0 = rf(endpoint, override)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ClientInterface) DockerOverride(newImage string, oldImage string) error {
	ret := _m.Called(newImage, oldImage)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(newImage, oldImage)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
