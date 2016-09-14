package mocks

import "github.com/stretchr/testify/mock"

import "time"
import "github.com/control-center/serviced/domain/applicationendpoint"
import "github.com/control-center/serviced/health"
import "github.com/control-center/serviced/domain/host"
import "github.com/control-center/serviced/domain/pool"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicedefinition"
import "github.com/control-center/serviced/domain/servicetemplate"
import "github.com/control-center/serviced/domain/user"
import "github.com/control-center/serviced/isvcs"
import "github.com/control-center/serviced/volume"

type ClientInterface struct {
	mock.Mock
}

// Close provides a mock function with given fields:
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

// GetHost provides a mock function with given fields: hostID
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

// GetHosts provides a mock function with given fields:
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

// GetActiveHostIDs provides a mock function with given fields:
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

// AddHost provides a mock function with given fields: h
func (_m *ClientInterface) AddHost(h host.Host) error {
	ret := _m.Called(h)

	var r0 error
	if rf, ok := ret.Get(0).(func(host.Host) error); ok {
		r0 = rf(h)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateHost provides a mock function with given fields: h
func (_m *ClientInterface) UpdateHost(h host.Host) error {
	ret := _m.Called(h)

	var r0 error
	if rf, ok := ret.Get(0).(func(host.Host) error); ok {
		r0 = rf(h)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveHost provides a mock function with given fields: hostID
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

// FindHostsInPool provides a mock function with given fields: poolID
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

// GetResourcePool provides a mock function with given fields: poolID
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

// GetResourcePools provides a mock function with given fields:
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

// AddResourcePool provides a mock function with given fields: p
func (_m *ClientInterface) AddResourcePool(p pool.ResourcePool) error {
	ret := _m.Called(p)

	var r0 error
	if rf, ok := ret.Get(0).(func(pool.ResourcePool) error); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateResourcePool provides a mock function with given fields: p
func (_m *ClientInterface) UpdateResourcePool(p pool.ResourcePool) error {
	ret := _m.Called(p)

	var r0 error
	if rf, ok := ret.Get(0).(func(pool.ResourcePool) error); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveResourcePool provides a mock function with given fields: poolID
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

// GetPoolIPs provides a mock function with given fields: poolID
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

// AddVirtualIP provides a mock function with given fields: requestVirtualIP
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

// RemoveVirtualIP provides a mock function with given fields: requestVirtualIP
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

// ServiceUse provides a mock function with given fields: serviceID, imageID, registry, replaceImgs, noOp
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

// WaitService provides a mock function with given fields: serviceIDs, state, timeout, recursive
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

// GetServiceInstances provides a mock function with given fields: serviceID
func (_m *ClientInterface) GetServiceInstances(serviceID string) ([]service.Instance, error) {
	ret := _m.Called(serviceID)

	var r0 []service.Instance
	if rf, ok := ret.Get(0).(func(string) []service.Instance); ok {
		r0 = rf(serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Instance)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(serviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEvaluatedService provides a mock function with given fields: serviceID, instanceID
func (_m *ClientInterface) GetEvaluatedService(serviceID string, instanceID int) (*service.Service, error) {
	ret := _m.Called(serviceID, instanceID)

	var r0 *service.Service
	if rf, ok := ret.Get(0).(func(string, int) *service.Service); ok {
		r0 = rf(serviceID, instanceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int) error); ok {
		r1 = rf(serviceID, instanceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTenantID provides a mock function with given fields: serviceID
func (_m *ClientInterface) GetTenantID(serviceID string) (string, error) {
	ret := _m.Called(serviceID)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(serviceID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(serviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StopServiceInstance provides a mock function with given fields: serviceID, instanceID
func (_m *ClientInterface) StopServiceInstance(serviceID string, instanceID int) error {
	ret := _m.Called(serviceID, instanceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int) error); ok {
		r0 = rf(serviceID, instanceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LocateServiceInstance provides a mock function with given fields: serviceID, instanceID
func (_m *ClientInterface) LocateServiceInstance(serviceID string, instanceID int) (*service.LocationInstance, error) {
	ret := _m.Called(serviceID, instanceID)

	var r0 *service.LocationInstance
	if rf, ok := ret.Get(0).(func(string, int) *service.LocationInstance); ok {
		r0 = rf(serviceID, instanceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.LocationInstance)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int) error); ok {
		r1 = rf(serviceID, instanceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendDockerAction provides a mock function with given fields: serviceID, instanceID, action, args
func (_m *ClientInterface) SendDockerAction(serviceID string, instanceID int, action string, args []string) error {
	ret := _m.Called(serviceID, instanceID, action, args)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int, string, []string) error); ok {
		r0 = rf(serviceID, instanceID, action, args)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddServiceTemplate provides a mock function with given fields: serviceTemplate
func (_m *ClientInterface) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate) (string, error) {
	ret := _m.Called(serviceTemplate)

	var r0 string
	if rf, ok := ret.Get(0).(func(servicetemplate.ServiceTemplate) string); ok {
		r0 = rf(serviceTemplate)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(servicetemplate.ServiceTemplate) error); ok {
		r1 = rf(serviceTemplate)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetServiceTemplates provides a mock function with given fields:
func (_m *ClientInterface) GetServiceTemplates() (map[string]servicetemplate.ServiceTemplate, error) {
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

// RemoveServiceTemplate provides a mock function with given fields: serviceTemplateID
func (_m *ClientInterface) RemoveServiceTemplate(serviceTemplateID string) error {
	ret := _m.Called(serviceTemplateID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(serviceTemplateID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeployTemplate provides a mock function with given fields: request
func (_m *ClientInterface) DeployTemplate(request servicetemplate.ServiceTemplateDeploymentRequest) ([]string, error) {
	ret := _m.Called(request)

	var r0 []string
	if rf, ok := ret.Get(0).(func(servicetemplate.ServiceTemplateDeploymentRequest) []string); ok {
		r0 = rf(request)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(servicetemplate.ServiceTemplateDeploymentRequest) error); ok {
		r1 = rf(request)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetVolumeStatus provides a mock function with given fields:
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

// GetServiceEndpoints provides a mock function with given fields: serviceIDs, reportImports, reportExports, validate
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

// ResetRegistry provides a mock function with given fields:
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

// SyncRegistry provides a mock function with given fields:
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

// UpgradeRegistry provides a mock function with given fields: endpoint, override
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

// DockerOverride provides a mock function with given fields: newImage, oldImage
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

// AddPublicEndpointPort provides a mock function with given fields: serviceid, endpointName, portAddr, usetls, protocol, isEnabled, restart
func (_m *ClientInterface) AddPublicEndpointPort(serviceid string, endpointName string, portAddr string, usetls bool, protocol string, isEnabled bool, restart bool) (*servicedefinition.Port, error) {
	ret := _m.Called(serviceid, endpointName, portAddr, usetls, protocol, isEnabled, restart)

	var r0 *servicedefinition.Port
	if rf, ok := ret.Get(0).(func(string, string, string, bool, string, bool, bool) *servicedefinition.Port); ok {
		r0 = rf(serviceid, endpointName, portAddr, usetls, protocol, isEnabled, restart)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*servicedefinition.Port)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string, bool, string, bool, bool) error); ok {
		r1 = rf(serviceid, endpointName, portAddr, usetls, protocol, isEnabled, restart)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemovePublicEndpointPort provides a mock function with given fields: serviceid, endpointName, portAddr
func (_m *ClientInterface) RemovePublicEndpointPort(serviceid string, endpointName string, portAddr string) error {
	ret := _m.Called(serviceid, endpointName, portAddr)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(serviceid, endpointName, portAddr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// EnablePublicEndpointPort provides a mock function with given fields: serviceid, endpointName, portAddr, isEnabled
func (_m *ClientInterface) EnablePublicEndpointPort(serviceid string, endpointName string, portAddr string, isEnabled bool) error {
	ret := _m.Called(serviceid, endpointName, portAddr, isEnabled)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, bool) error); ok {
		r0 = rf(serviceid, endpointName, portAddr, isEnabled)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddPublicEndpointVHost provides a mock function with given fields: serviceid, endpointName, vhost, isEnabled, restart
func (_m *ClientInterface) AddPublicEndpointVHost(serviceid string, endpointName string, vhost string, isEnabled bool, restart bool) (*servicedefinition.VHost, error) {
	ret := _m.Called(serviceid, endpointName, vhost, isEnabled, restart)

	var r0 *servicedefinition.VHost
	if rf, ok := ret.Get(0).(func(string, string, string, bool, bool) *servicedefinition.VHost); ok {
		r0 = rf(serviceid, endpointName, vhost, isEnabled, restart)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*servicedefinition.VHost)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string, bool, bool) error); ok {
		r1 = rf(serviceid, endpointName, vhost, isEnabled, restart)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemovePublicEndpointVHost provides a mock function with given fields: serviceid, endpointName, vhost
func (_m *ClientInterface) RemovePublicEndpointVHost(serviceid string, endpointName string, vhost string) error {
	ret := _m.Called(serviceid, endpointName, vhost)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(serviceid, endpointName, vhost)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// EnablePublicEndpointVHost provides a mock function with given fields: serviceid, endpointName, vhost, isEnabled
func (_m *ClientInterface) EnablePublicEndpointVHost(serviceid string, endpointName string, vhost string, isEnabled bool) error {
	ret := _m.Called(serviceid, endpointName, vhost, isEnabled)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, bool) error); ok {
		r0 = rf(serviceid, endpointName, vhost, isEnabled)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetSystemUser provides a mock function with given fields:
func (_m *ClientInterface) GetSystemUser() (user.User, error) {
	ret := _m.Called()

	var r0 user.User
	if rf, ok := ret.Get(0).(func() user.User); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(user.User)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ValidateCredentials provides a mock function with given fields: user
func (_m *ClientInterface) ValidateCredentials(someUser user.User) (bool, error) {
	ret := _m.Called(someUser)

	var r0 bool
	if rf, ok := ret.Get(0).(func(user.User) bool); ok {
		r0 = rf(someUser)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(user.User) error); ok {
		r1 = rf(someUser)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetISvcsHealth provides a mock function with given fields: IServiceNames
func (_m *ClientInterface) GetISvcsHealth(IServiceNames []string) ([]isvcs.IServiceHealthResult, error) {
	ret := _m.Called(IServiceNames)

	var r0 []isvcs.IServiceHealthResult
	if rf, ok := ret.Get(0).(func([]string) []isvcs.IServiceHealthResult); ok {
		r0 = rf(IServiceNames)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]isvcs.IServiceHealthResult)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]string) error); ok {
		r1 = rf(IServiceNames)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetServicesHealth provides a mock function with given fields:
func (_m *ClientInterface) GetServicesHealth() (map[string]map[int]map[string]health.HealthStatus, error) {
	ret := _m.Called()

	var r0 map[string]map[int]map[string]health.HealthStatus
	if rf, ok := ret.Get(0).(func() map[string]map[int]map[string]health.HealthStatus); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]map[int]map[string]health.HealthStatus)
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

// ReportHealthStatus provides a mock function with given fields: key, value, expires
func (_m *ClientInterface) ReportHealthStatus(key health.HealthStatusKey, value health.HealthStatus, expires time.Duration) error {
	ret := _m.Called(key, value, expires)

	var r0 error
	if rf, ok := ret.Get(0).(func(health.HealthStatusKey, health.HealthStatus, time.Duration) error); ok {
		r0 = rf(key, value, expires)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReportInstanceDead provides a mock function with given fields: serviceID, instanceID
func (_m *ClientInterface) ReportInstanceDead(serviceID string, instanceID int) error {
	ret := _m.Called(serviceID, instanceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int) error); ok {
		r0 = rf(serviceID, instanceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
