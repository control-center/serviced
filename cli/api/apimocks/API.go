package mocks

import "github.com/control-center/serviced/cli/api"
import "github.com/stretchr/testify/mock"

import "io"
import "github.com/control-center/serviced/dao"
import "github.com/control-center/serviced/domain/applicationendpoint"
import "github.com/control-center/serviced/domain/host"
import "github.com/control-center/serviced/domain/pool"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicedefinition"
import template "github.com/control-center/serviced/domain/servicetemplate"
import "github.com/control-center/serviced/isvcs"
import "github.com/control-center/serviced/metrics"
import "github.com/control-center/serviced/script"
import "github.com/control-center/serviced/volume"

type API struct {
	mock.Mock
}

func (_m *API) StartServer() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) ServicedHealthCheck(IServiceNames []string) ([]isvcs.IServiceHealthResult, error) {
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
func (_m *API) GetHosts() ([]host.Host, error) {
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
func (_m *API) GetHost(_a0 string) (*host.Host, error) {
	ret := _m.Called(_a0)

	var r0 *host.Host
	if rf, ok := ret.Get(0).(func(string) *host.Host); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*host.Host)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) GetHostMap() (map[string]host.Host, error) {
	ret := _m.Called()

	var r0 map[string]host.Host
	if rf, ok := ret.Get(0).(func() map[string]host.Host); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]host.Host)
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
func (_m *API) AddHost(_a0 api.HostConfig) (*host.Host, []byte, error) {
	ret := _m.Called(_a0)

	var r0 *host.Host
	if rf, ok := ret.Get(0).(func(api.HostConfig) *host.Host); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*host.Host)
		}
	}

	var r1 []byte
	if rf, ok := ret.Get(1).(func(api.HostConfig) []byte); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]byte)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(api.HostConfig) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *API) AuthenticateHost(_a0 string) (string, int64, error) {
	ret := _m.Called(_a0)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(string)
		}
	}

	var r1 int64
	if rf, ok := ret.Get(1).(func(string) int64); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(int64)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *API) RemoveHost(_a0 string) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) GetHostMemory(_a0 string) (*metrics.MemoryUsageStats, error) {
	ret := _m.Called(_a0)

	var r0 *metrics.MemoryUsageStats
	if rf, ok := ret.Get(0).(func(string) *metrics.MemoryUsageStats); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*metrics.MemoryUsageStats)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) SetHostMemory(_a0 api.HostUpdateConfig) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(api.HostUpdateConfig) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) GetHostPublicKey(_a0 string) ([]byte, error) {
	ret := _m.Called(_a0)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(string) []byte); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) RegisterHost(_a0 []byte) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) RegisterRemoteHost(_a0 *host.Host, _a1 []byte, _a2 bool) error {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 error
	if rf, ok := ret.Get(0).(func(*host.Host, []byte, bool) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) WriteDelegateKey(_a0 string, _a1 []byte) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, []byte) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) ResetHostKey(_a0 string) ([]byte, error) {
	ret := _m.Called(_a0)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(string) []byte); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) GetResourcePools() ([]pool.ResourcePool, error) {
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
func (_m *API) GetResourcePool(_a0 string) (*pool.ResourcePool, error) {
	ret := _m.Called(_a0)

	var r0 *pool.ResourcePool
	if rf, ok := ret.Get(0).(func(string) *pool.ResourcePool); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pool.ResourcePool)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) AddResourcePool(_a0 api.PoolConfig) (*pool.ResourcePool, error) {
	ret := _m.Called(_a0)

	var r0 *pool.ResourcePool
	if rf, ok := ret.Get(0).(func(api.PoolConfig) *pool.ResourcePool); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pool.ResourcePool)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(api.PoolConfig) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) RemoveResourcePool(_a0 string) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) UpdateResourcePool(p pool.ResourcePool) error {
	ret := _m.Called(p)

	var r0 error
	if rf, ok := ret.Get(0).(func(pool.ResourcePool) error); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) GetPoolIPs(_a0 string) (*pool.PoolIPs, error) {
	ret := _m.Called(_a0)

	var r0 *pool.PoolIPs
	if rf, ok := ret.Get(0).(func(string) *pool.PoolIPs); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pool.PoolIPs)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) AddVirtualIP(_a0 pool.VirtualIP) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(pool.VirtualIP) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) RemoveVirtualIP(_a0 pool.VirtualIP) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(pool.VirtualIP) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) GetServicesDeprecated() ([]service.Service, error) {
	ret := _m.Called()

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func() []service.Service); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
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
func (_m *API) GetServiceStatus(_a0 string) (map[string]map[string]interface{}, error) {
	ret := _m.Called(_a0)

	var r0 map[string]map[string]interface{}
	if rf, ok := ret.Get(0).(func(string) map[string]map[string]interface{}); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]map[string]interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) GetService(_a0 string) (*service.Service, error) {
	ret := _m.Called(_a0)

	var r0 *service.Service
	if rf, ok := ret.Get(0).(func(string) *service.Service); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) GetAllServiceDetails() ([]service.ServiceDetails, error) {
	ret := _m.Called()

	var r0 []service.ServiceDetails
	if rf, ok := ret.Get(0).(func() []service.ServiceDetails); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ServiceDetails)
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
func (_m *API) GetServiceDetails(_a0 string) (*service.ServiceDetails, error) {
	ret := _m.Called(_a0)

	var r0 *service.ServiceDetails
	if rf, ok := ret.Get(0).(func(string) *service.ServiceDetails); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.ServiceDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) AddService(_a0 api.ServiceConfig) (*service.ServiceDetails, error) {
	ret := _m.Called(_a0)

	var r0 *service.ServiceDetails
	if rf, ok := ret.Get(0).(func(api.ServiceConfig) *service.ServiceDetails); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.ServiceDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(api.ServiceConfig) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) CloneService(_a0 string, _a1 string) (*service.ServiceDetails, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *service.ServiceDetails
	if rf, ok := ret.Get(0).(func(string, string) *service.ServiceDetails); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.ServiceDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) RemoveService(_a0 string) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) UpdateService(_a0 io.Reader) (*service.ServiceDetails, error) {
	ret := _m.Called(_a0)

	var r0 *service.ServiceDetails
	if rf, ok := ret.Get(0).(func(io.Reader) *service.ServiceDetails); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.ServiceDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(io.Reader) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) StartService(_a0 api.SchedulerConfig) (int, error) {
	ret := _m.Called(_a0)

	var r0 int
	if rf, ok := ret.Get(0).(func(api.SchedulerConfig) int); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(api.SchedulerConfig) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) RestartService(_a0 api.SchedulerConfig) (int, error) {
	ret := _m.Called(_a0)

	var r0 int
	if rf, ok := ret.Get(0).(func(api.SchedulerConfig) int); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(api.SchedulerConfig) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) StopService(_a0 api.SchedulerConfig) (int, error) {
	ret := _m.Called(_a0)

	var r0 int
	if rf, ok := ret.Get(0).(func(api.SchedulerConfig) int); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(api.SchedulerConfig) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) AssignIP(_a0 api.IPConfig) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(api.IPConfig) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) GetEndpoints(serviceID string, reportImports bool, reportExports bool, validate bool) ([]applicationendpoint.EndpointReport, error) {
	ret := _m.Called(serviceID, reportImports, reportExports, validate)

	var r0 []applicationendpoint.EndpointReport
	if rf, ok := ret.Get(0).(func(string, bool, bool, bool) []applicationendpoint.EndpointReport); ok {
		r0 = rf(serviceID, reportImports, reportExports, validate)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]applicationendpoint.EndpointReport)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, bool, bool, bool) error); ok {
		r1 = rf(serviceID, reportImports, reportExports, validate)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) StartShell(_a0 api.ShellConfig) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(api.ShellConfig) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) RunShell(_a0 api.ShellConfig, _a1 chan struct{}) (int, error) {
	ret := _m.Called(_a0, _a1)

	var r0 int
	if rf, ok := ret.Get(0).(func(api.ShellConfig, chan struct{}) int); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(api.ShellConfig, chan struct{}) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) GetSnapshots() ([]dao.SnapshotInfo, error) {
	ret := _m.Called()

	var r0 []dao.SnapshotInfo
	if rf, ok := ret.Get(0).(func() []dao.SnapshotInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]dao.SnapshotInfo)
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
func (_m *API) GetSnapshotsByServiceID(_a0 string) ([]dao.SnapshotInfo, error) {
	ret := _m.Called(_a0)

	var r0 []dao.SnapshotInfo
	if rf, ok := ret.Get(0).(func(string) []dao.SnapshotInfo); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]dao.SnapshotInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) GetSnapshotByServiceIDAndTag(_a0 string, _a1 string) (string, error) {
	ret := _m.Called(_a0, _a1)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) AddSnapshot(_a0 api.SnapshotConfig) (string, error) {
	ret := _m.Called(_a0)

	var r0 string
	if rf, ok := ret.Get(0).(func(api.SnapshotConfig) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(api.SnapshotConfig) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) RemoveSnapshot(_a0 string) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) Rollback(_a0 string, _a1 bool) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) TagSnapshot(_a0 string, _a1 string) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) RemoveSnapshotTag(_a0 string, _a1 string) (string, error) {
	ret := _m.Called(_a0, _a1)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) GetServiceTemplates() ([]template.ServiceTemplate, error) {
	ret := _m.Called()

	var r0 []template.ServiceTemplate
	if rf, ok := ret.Get(0).(func() []template.ServiceTemplate); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]template.ServiceTemplate)
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
func (_m *API) GetServiceTemplate(_a0 string) (*template.ServiceTemplate, error) {
	ret := _m.Called(_a0)

	var r0 *template.ServiceTemplate
	if rf, ok := ret.Get(0).(func(string) *template.ServiceTemplate); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*template.ServiceTemplate)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) AddServiceTemplate(_a0 io.Reader) (*template.ServiceTemplate, error) {
	ret := _m.Called(_a0)

	var r0 *template.ServiceTemplate
	if rf, ok := ret.Get(0).(func(io.Reader) *template.ServiceTemplate); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*template.ServiceTemplate)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(io.Reader) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) RemoveServiceTemplate(_a0 string) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) CompileServiceTemplate(_a0 api.CompileTemplateConfig) (*template.ServiceTemplate, error) {
	ret := _m.Called(_a0)

	var r0 *template.ServiceTemplate
	if rf, ok := ret.Get(0).(func(api.CompileTemplateConfig) *template.ServiceTemplate); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*template.ServiceTemplate)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(api.CompileTemplateConfig) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) DeployServiceTemplate(_a0 api.DeployTemplateConfig) ([]service.ServiceDetails, error) {
	ret := _m.Called(_a0)

	var r0 []service.ServiceDetails
	if rf, ok := ret.Get(0).(func(api.DeployTemplateConfig) []service.ServiceDetails); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ServiceDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(api.DeployTemplateConfig) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) Backup(_a0 string, _a1 []string) (string, error) {
	ret := _m.Called(_a0, _a1)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, []string) string); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, []string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) Restore(_a0 string) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) ResetRegistry() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) RegistrySync() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) UpgradeRegistry(endpoint string, override bool) error {
	ret := _m.Called(endpoint, override)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, bool) error); ok {
		r0 = rf(endpoint, override)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) DockerOverride(newImage string, oldImage string) error {
	ret := _m.Called(newImage, oldImage)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(newImage, oldImage)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) ExportLogs(config api.ExportLogsConfig) error {
	ret := _m.Called(config)

	var r0 error
	if rf, ok := ret.Get(0).(func(api.ExportLogsConfig) error); ok {
		r0 = rf(config)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) PostMetric(metricName string, metricValue string) (string, error) {
	ret := _m.Called(metricName, metricValue)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(metricName, metricValue)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(metricName, metricValue)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) ScriptParse(fileName string, config *script.Config) error {
	ret := _m.Called(fileName, config)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *script.Config) error); ok {
		r0 = rf(fileName, config)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) ScriptRun(a0 string, a1 *script.Config, stopChan chan struct{}) error {
	ret := _m.Called(a0, a1, stopChan)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *script.Config, chan struct{}) error); ok {
		r0 = rf(a0, a1, stopChan)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *API) GetVolumeStatus() (*volume.Statuses, error) {
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
func (_m *API) AddPublicEndpointPort(serviceid string, endpointName string, portAddr string, usetls bool, protocol string, isEnabled bool, restart bool) (*servicedefinition.Port, error) {
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
func (_m *API) RemovePublicEndpointPort(serviceid string, endpointName string, portAddr string) error {
	ret := _m.Called(serviceid, endpointName, portAddr)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(serviceid, endpointName, portAddr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) EnablePublicEndpointPort(serviceid string, endpointName string, portAddr string, isEnabled bool) error {
	ret := _m.Called(serviceid, endpointName, portAddr, isEnabled)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, bool) error); ok {
		r0 = rf(serviceid, endpointName, portAddr, isEnabled)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) AddPublicEndpointVHost(serviceid string, endpointName string, vhost string, isEnabled bool, restart bool) (*servicedefinition.VHost, error) {
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
func (_m *API) RemovePublicEndpointVHost(serviceid string, endpointName string, vhost string) error {
	ret := _m.Called(serviceid, endpointName, vhost)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(serviceid, endpointName, vhost)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) EnablePublicEndpointVHost(serviceid string, endpointName string, vhost string, isEnabled bool) error {
	ret := _m.Called(serviceid, endpointName, vhost, isEnabled)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, bool) error); ok {
		r0 = rf(serviceid, endpointName, vhost, isEnabled)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) GetServiceInstances(serviceID string) ([]service.Instance, error) {
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
func (_m *API) StopServiceInstance(serviceID string, instanceID int) error {
	ret := _m.Called(serviceID, instanceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int) error); ok {
		r0 = rf(serviceID, instanceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) AttachServiceInstance(serviceID string, instanceID int, command string, args []string) error {
	ret := _m.Called(serviceID, instanceID, command, args)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int, string, []string) error); ok {
		r0 = rf(serviceID, instanceID, command, args)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) LogsForServiceInstance(serviceID string, instanceID int, command string, args []string) error {
	ret := _m.Called(serviceID, instanceID, command, args)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int, string, []string) error); ok {
		r0 = rf(serviceID, instanceID, command, args)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) SendDockerAction(serviceID string, instanceID int, action string, args []string) error {
	ret := _m.Called(serviceID, instanceID, action, args)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int, string, []string) error); ok {
		r0 = rf(serviceID, instanceID, action, args)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *API) GetHostWithAuthInfo(_a0 string) (*api.AuthHost, error) {
	ret := _m.Called(_a0)

	var r0 *api.AuthHost
	if rf, ok := ret.Get(0).(func(string) *api.AuthHost); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.AuthHost)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *API) GetHostsWithAuthInfo() ([]api.AuthHost, error) {
	ret := _m.Called()

	var r0 []api.AuthHost
	if rf, ok := ret.Get(0).(func() []api.AuthHost); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]api.AuthHost)
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
func (_m *API) DebugEnableMetrics() (string, error) {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
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
func (_m *API) DebugDisableMetrics() (string, error) {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
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
func (_m *API) GetAllPublicEndpoints() ([]service.PublicEndpoint, error) {
	ret := _m.Called()

	var r0 []service.PublicEndpoint
	if rf, ok := ret.Get(0).(func() []service.PublicEndpoint); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.PublicEndpoint)
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
