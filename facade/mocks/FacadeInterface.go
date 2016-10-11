package mocks

import "github.com/stretchr/testify/mock"

import "time"
import "github.com/control-center/serviced/dao"
import "github.com/control-center/serviced/datastore"
import "github.com/control-center/serviced/domain"
import "github.com/control-center/serviced/health"
import "github.com/control-center/serviced/domain/addressassignment"
import "github.com/control-center/serviced/domain/host"
import "github.com/control-center/serviced/domain/pool"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicedefinition"
import "github.com/control-center/serviced/domain/servicetemplate"
import "github.com/control-center/serviced/domain/user"

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
func (_m *FacadeInterface) GetEvaluatedService(ctx datastore.Context, servicedID string, instanceID int) (*service.Service, error) {
	ret := _m.Called(ctx, servicedID, instanceID)

	var r0 *service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string, int) *service.Service); ok {
		r0 = rf(ctx, servicedID, instanceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, int) error); ok {
		r1 = rf(ctx, servicedID, instanceID)
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
func (_m *FacadeInterface) GetServicesByImage(ctx datastore.Context, imageID string) ([]service.Service, error) {
	ret := _m.Called(ctx, imageID)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []service.Service); ok {
		r0 = rf(ctx, imageID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, imageID)
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
func (_m *FacadeInterface) SyncServiceRegistry(ctx datastore.Context, svc *service.Service) error {
	ret := _m.Called(ctx, svc)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *service.Service) error); ok {
		r0 = rf(ctx, svc)
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
func (_m *FacadeInterface) WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, recursive bool, serviceIDs ...string) error {
	ret := _m.Called(ctx, dstate, timeout, recursive, serviceIDs)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, service.DesiredState, time.Duration, bool, ...string) error); ok {
		r0 = rf(ctx, dstate, timeout, recursive, serviceIDs...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) AssignIPs(ctx datastore.Context, assignmentRequest addressassignment.AssignmentRequest) error {
	ret := _m.Called(ctx, assignmentRequest)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, addressassignment.AssignmentRequest) error); ok {
		r0 = rf(ctx, assignmentRequest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) AddServiceTemplate(ctx datastore.Context, serviceTemplate servicetemplate.ServiceTemplate) (string, error) {
	ret := _m.Called(ctx, serviceTemplate)

	var r0 string
	if rf, ok := ret.Get(0).(func(datastore.Context, servicetemplate.ServiceTemplate) string); ok {
		r0 = rf(ctx, serviceTemplate)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, servicetemplate.ServiceTemplate) error); ok {
		r1 = rf(ctx, serviceTemplate)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
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
func (_m *FacadeInterface) RemoveServiceTemplate(ctx datastore.Context, templateID string) error {
	ret := _m.Called(ctx, templateID)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, templateID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
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
func (_m *FacadeInterface) DeployTemplate(ctx datastore.Context, poolID string, templateID string, deploymentID string) ([]string, error) {
	ret := _m.Called(ctx, poolID, templateID, deploymentID)

	var r0 []string
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string) []string); ok {
		r0 = rf(ctx, poolID, templateID, deploymentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string, string) error); ok {
		r1 = rf(ctx, poolID, templateID, deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) DeployTemplateActive() ([]map[string]string, error) {
	ret := _m.Called()

	var r0 []map[string]string
	if rf, ok := ret.Get(0).(func() []map[string]string); ok {
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
func (_m *FacadeInterface) DeployTemplateStatus(deploymentID string) (string, error) {
	ret := _m.Called(deploymentID)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(deploymentID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) AddHost(ctx datastore.Context, entity *host.Host) ([]byte, error) {
	ret := _m.Called(ctx, entity)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(datastore.Context, *host.Host) []byte); ok {
		r0 = rf(ctx, entity)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, *host.Host) error); ok {
		r1 = rf(ctx, entity)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetHost(ctx datastore.Context, hostID string) (*host.Host, error) {
	ret := _m.Called(ctx, hostID)

	var r0 *host.Host
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *host.Host); ok {
		r0 = rf(ctx, hostID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*host.Host)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, hostID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
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
func (_m *FacadeInterface) GetHostKey(ctx datastore.Context, hostID string) ([]byte, error) {
	ret := _m.Called(ctx, hostID)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []byte); ok {
		r0 = rf(ctx, hostID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, hostID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) ResetHostKey(ctx datastore.Context, hostID string) ([]byte, error) {
	ret := _m.Called(ctx, hostID)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []byte); ok {
		r0 = rf(ctx, hostID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, hostID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetActiveHostIDs(ctx datastore.Context) ([]string, error) {
	ret := _m.Called(ctx)

	var r0 []string
	if rf, ok := ret.Get(0).(func(datastore.Context) []string); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
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
func (_m *FacadeInterface) UpdateHost(ctx datastore.Context, entity *host.Host) error {
	ret := _m.Called(ctx, entity)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *host.Host) error); ok {
		r0 = rf(ctx, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) RemoveHost(ctx datastore.Context, hostID string) error {
	ret := _m.Called(ctx, hostID)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, hostID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) FindHostsInPool(ctx datastore.Context, poolID string) ([]host.Host, error) {
	ret := _m.Called(ctx, poolID)

	var r0 []host.Host
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []host.Host); ok {
		r0 = rf(ctx, poolID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]host.Host)
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
func (_m *FacadeInterface) GetResourcePool(ctx datastore.Context, poolID string) (*pool.ResourcePool, error) {
	ret := _m.Called(ctx, poolID)

	var r0 *pool.ResourcePool
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *pool.ResourcePool); ok {
		r0 = rf(ctx, poolID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pool.ResourcePool)
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
func (_m *FacadeInterface) GetPoolIPs(ctx datastore.Context, poolID string) (*pool.PoolIPs, error) {
	ret := _m.Called(ctx, poolID)

	var r0 *pool.PoolIPs
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *pool.PoolIPs); ok {
		r0 = rf(ctx, poolID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pool.PoolIPs)
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
func (_m *FacadeInterface) RemoveResourcePool(ctx datastore.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
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
func (_m *FacadeInterface) GetHealthChecksForService(ctx datastore.Context, id string) (map[string]health.HealthCheck, error) {
	ret := _m.Called(ctx, id)

	var r0 map[string]health.HealthCheck
	if rf, ok := ret.Get(0).(func(datastore.Context, string) map[string]health.HealthCheck); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]health.HealthCheck)
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
func (_m *FacadeInterface) AddPublicEndpointPort(ctx datastore.Context, serviceid string, endpointName string, portAddr string, usetls bool, protocol string, isEnabled bool, restart bool) (*servicedefinition.Port, error) {
	ret := _m.Called(ctx, serviceid, endpointName, portAddr, usetls, protocol, isEnabled, restart)

	var r0 *servicedefinition.Port
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string, bool, string, bool, bool) *servicedefinition.Port); ok {
		r0 = rf(ctx, serviceid, endpointName, portAddr, usetls, protocol, isEnabled, restart)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*servicedefinition.Port)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string, string, bool, string, bool, bool) error); ok {
		r1 = rf(ctx, serviceid, endpointName, portAddr, usetls, protocol, isEnabled, restart)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) RemovePublicEndpointPort(ctx datastore.Context, serviceid string, endpointName string, portAddr string) error {
	ret := _m.Called(ctx, serviceid, endpointName, portAddr)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string) error); ok {
		r0 = rf(ctx, serviceid, endpointName, portAddr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) EnablePublicEndpointPort(ctx datastore.Context, serviceid string, endpointName string, portAddr string, isEnabled bool) error {
	ret := _m.Called(ctx, serviceid, endpointName, portAddr, isEnabled)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string, bool) error); ok {
		r0 = rf(ctx, serviceid, endpointName, portAddr, isEnabled)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) AddPublicEndpointVHost(ctx datastore.Context, serviceid string, endpointName string, vhost string, isEnabled bool, restart bool) (*servicedefinition.VHost, error) {
	ret := _m.Called(ctx, serviceid, endpointName, vhost, isEnabled, restart)

	var r0 *servicedefinition.VHost
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string, bool, bool) *servicedefinition.VHost); ok {
		r0 = rf(ctx, serviceid, endpointName, vhost, isEnabled, restart)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*servicedefinition.VHost)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string, string, bool, bool) error); ok {
		r1 = rf(ctx, serviceid, endpointName, vhost, isEnabled, restart)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) RemovePublicEndpointVHost(ctx datastore.Context, serviceid string, endpointName string, vhost string) error {
	ret := _m.Called(ctx, serviceid, endpointName, vhost)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string) error); ok {
		r0 = rf(ctx, serviceid, endpointName, vhost)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) EnablePublicEndpointVHost(ctx datastore.Context, serviceid string, endpointName string, vhost string, isEnabled bool) error {
	ret := _m.Called(ctx, serviceid, endpointName, vhost, isEnabled)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string, bool) error); ok {
		r0 = rf(ctx, serviceid, endpointName, vhost, isEnabled)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) GetHostInstances(ctx datastore.Context, since time.Time, hostid string) ([]service.Instance, error) {
	ret := _m.Called(ctx, since, hostid)

	var r0 []service.Instance
	if rf, ok := ret.Get(0).(func(datastore.Context, time.Time, string) []service.Instance); ok {
		r0 = rf(ctx, since, hostid)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Instance)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, time.Time, string) error); ok {
		r1 = rf(ctx, since, hostid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetServiceInstances(ctx datastore.Context, since time.Time, serviceid string) ([]service.Instance, error) {
	ret := _m.Called(ctx, since, serviceid)

	var r0 []service.Instance
	if rf, ok := ret.Get(0).(func(datastore.Context, time.Time, string) []service.Instance); ok {
		r0 = rf(ctx, since, serviceid)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Instance)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, time.Time, string) error); ok {
		r1 = rf(ctx, since, serviceid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetAggregateServices(ctx datastore.Context, since time.Time, serviceids []string) ([]service.AggregateService, error) {
	ret := _m.Called(ctx, since, serviceids)

	var r0 []service.AggregateService
	if rf, ok := ret.Get(0).(func(datastore.Context, time.Time, []string) []service.AggregateService); ok {
		r0 = rf(ctx, since, serviceids)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.AggregateService)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, time.Time, []string) error); ok {
		r1 = rf(ctx, since, serviceids)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetReadPools(ctx datastore.Context) ([]pool.ReadPool, error) {
	ret := _m.Called(ctx)

	var r0 []pool.ReadPool
	if rf, ok := ret.Get(0).(func(datastore.Context) []pool.ReadPool); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pool.ReadPool)
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
func (_m *FacadeInterface) GetReadHosts(ctx datastore.Context) ([]host.ReadHost, error) {
	ret := _m.Called(ctx)

	var r0 []host.ReadHost
	if rf, ok := ret.Get(0).(func(datastore.Context) []host.ReadHost); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]host.ReadHost)
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
func (_m *FacadeInterface) FindReadHostsInPool(ctx datastore.Context, poolID string) ([]host.ReadHost, error) {
	ret := _m.Called(ctx, poolID)

	var r0 []host.ReadHost
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []host.ReadHost); ok {
		r0 = rf(ctx, poolID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]host.ReadHost)
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
func (_m *FacadeInterface) GetAllServiceDetails(ctx datastore.Context) ([]service.ServiceDetails, error) {
	ret := _m.Called(ctx)

	var r0 []service.ServiceDetails
	if rf, ok := ret.Get(0).(func(datastore.Context) []service.ServiceDetails); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ServiceDetails)
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
func (_m *FacadeInterface) GetServiceDetails(ctx datastore.Context, serviceID string) (*service.ServiceDetails, error) {
	ret := _m.Called(ctx, serviceID)

	var r0 *service.ServiceDetails
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *service.ServiceDetails); ok {
		r0 = rf(ctx, serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.ServiceDetails)
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
func (_m *FacadeInterface) GetServiceDetailsAncestry(ctx datastore.Context, serviceID string) (*service.ServiceDetails, error) {
	ret := _m.Called(ctx, serviceID)

	var r0 *service.ServiceDetails
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *service.ServiceDetails); ok {
		r0 = rf(ctx, serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.ServiceDetails)
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
func (_m *FacadeInterface) GetServiceDetailsByParentID(ctx datastore.Context, serviceID string) ([]service.ServiceDetails, error) {
	ret := _m.Called(ctx, serviceID)

	var r0 []service.ServiceDetails
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []service.ServiceDetails); ok {
		r0 = rf(ctx, serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ServiceDetails)
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
func (_m *FacadeInterface) GetServiceMonitoringProfile(ctx datastore.Context, serviceID string) (*domain.MonitorProfile, error) {
	ret := _m.Called(ctx, serviceID)

	var r0 *domain.MonitorProfile
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *domain.MonitorProfile); ok {
		r0 = rf(ctx, serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.MonitorProfile)
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
func (_m *FacadeInterface) GetServicePublicEndpoints(ctx datastore.Context, serviceID string, children bool) ([]service.PublicEndpoint, error) {
	ret := _m.Called(ctx, serviceID, children)

	var r0 []service.PublicEndpoint
	if rf, ok := ret.Get(0).(func(datastore.Context, string, bool) []service.PublicEndpoint); ok {
		r0 = rf(ctx, serviceID, children)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.PublicEndpoint)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, bool) error); ok {
		r1 = rf(ctx, serviceID, children)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetServiceAddressAssignmentDetails(ctx datastore.Context, serviceID string, children bool) ([]service.IPAssignment, error) {
	ret := _m.Called(ctx, serviceID, children)

	var r0 []service.IPAssignment
	if rf, ok := ret.Get(0).(func(datastore.Context, string, bool) []service.IPAssignment); ok {
		r0 = rf(ctx, serviceID, children)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.IPAssignment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, bool) error); ok {
		r1 = rf(ctx, serviceID, children)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetServiceExportedEndpoints(ctx datastore.Context, serviceID string, children bool) ([]service.ExportedEndpoint, error) {
	ret := _m.Called(ctx, serviceID, children)

	var r0 []service.ExportedEndpoint
	if rf, ok := ret.Get(0).(func(datastore.Context, string, bool) []service.ExportedEndpoint); ok {
		r0 = rf(ctx, serviceID, children)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ExportedEndpoint)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, bool) error); ok {
		r1 = rf(ctx, serviceID, children)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) AddUser(ctx datastore.Context, newUser user.User) error {
	ret := _m.Called(ctx, newUser)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, user.User) error); ok {
		r0 = rf(ctx, newUser)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) GetUser(ctx datastore.Context, userName string) (user.User, error) {
	ret := _m.Called(ctx, userName)

	var r0 user.User
	if rf, ok := ret.Get(0).(func(datastore.Context, string) user.User); ok {
		r0 = rf(ctx, userName)
	} else {
		r0 = ret.Get(0).(user.User)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, userName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) UpdateUser(ctx datastore.Context, u user.User) error {
	ret := _m.Called(ctx, u)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, user.User) error); ok {
		r0 = rf(ctx, u)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) RemoveUser(ctx datastore.Context, userName string) error {
	ret := _m.Called(ctx, userName)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, userName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) GetSystemUser(ctx datastore.Context) (user.User, error) {
	ret := _m.Called(ctx)

	var r0 user.User
	if rf, ok := ret.Get(0).(func(datastore.Context) user.User); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(user.User)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) ValidateCredentials(ctx datastore.Context, u user.User) (bool, error) {
	ret := _m.Called(ctx, u)

	var r0 bool
	if rf, ok := ret.Get(0).(func(datastore.Context, user.User) bool); ok {
		r0 = rf(ctx, u)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, user.User) error); ok {
		r1 = rf(ctx, u)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) GetServicesHealth(ctx datastore.Context) (map[string]map[int]map[string]health.HealthStatus, error) {
	ret := _m.Called(ctx)

	var r0 map[string]map[int]map[string]health.HealthStatus
	if rf, ok := ret.Get(0).(func(datastore.Context) map[string]map[int]map[string]health.HealthStatus); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]map[int]map[string]health.HealthStatus)
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
func (_m *FacadeInterface) ReportHealthStatus(key health.HealthStatusKey, value health.HealthStatus, expires time.Duration) {
	_m.Called(key, value, expires)
}
func (_m *FacadeInterface) ReportInstanceDead(serviceID string, instanceID int) {
	_m.Called(serviceID, instanceID)
}
func (_m *FacadeInterface) GetServiceConfigs(ctx datastore.Context, serviceID string) ([]service.Config, error) {
	ret := _m.Called(ctx, serviceID)

	var r0 []service.Config
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []service.Config); ok {
		r0 = rf(ctx, serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Config)
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
func (_m *FacadeInterface) GetServiceConfig(ctx datastore.Context, fileID string) (*servicedefinition.ConfigFile, error) {
	ret := _m.Called(ctx, fileID)

	var r0 *servicedefinition.ConfigFile
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *servicedefinition.ConfigFile); ok {
		r0 = rf(ctx, fileID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*servicedefinition.ConfigFile)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, fileID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FacadeInterface) AddServiceConfig(ctx datastore.Context, serviceID string, conf servicedefinition.ConfigFile) error {
	ret := _m.Called(ctx, serviceID, conf)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, servicedefinition.ConfigFile) error); ok {
		r0 = rf(ctx, serviceID, conf)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) UpdateServiceConfig(ctx datastore.Context, fileID string, conf servicedefinition.ConfigFile) error {
	ret := _m.Called(ctx, fileID, conf)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, servicedefinition.ConfigFile) error); ok {
		r0 = rf(ctx, fileID, conf)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *FacadeInterface) DeleteServiceConfig(ctx datastore.Context, fileID string) error {
	ret := _m.Called(ctx, fileID)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, fileID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
