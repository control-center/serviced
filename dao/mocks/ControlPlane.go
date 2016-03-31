package mocks

import "github.com/control-center/serviced/dao"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/domain"
import "github.com/control-center/serviced/domain/addressassignment"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicestate"
import "github.com/control-center/serviced/domain/user"
import "github.com/control-center/serviced/health"
import "github.com/control-center/serviced/metrics"

type ControlPlane struct {
	mock.Mock
}

// GetTenantId provides a mock function with given fields: serviceId, tenantId
func (_m *ControlPlane) GetTenantId(serviceId string, tenantId *string) error {
	ret := _m.Called(serviceId, tenantId)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *string) error); ok {
		r0 = rf(serviceId, tenantId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddService provides a mock function with given fields: svc, serviceId
func (_m *ControlPlane) AddService(svc service.Service, serviceId *string) error {
	ret := _m.Called(svc, serviceId)

	var r0 error
	if rf, ok := ret.Get(0).(func(service.Service, *string) error); ok {
		r0 = rf(svc, serviceId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CloneService provides a mock function with given fields: request, serviceId
func (_m *ControlPlane) CloneService(request dao.ServiceCloneRequest, serviceId *string) error {
	ret := _m.Called(request, serviceId)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ServiceCloneRequest, *string) error); ok {
		r0 = rf(request, serviceId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeployService provides a mock function with given fields: svc, serviceId
func (_m *ControlPlane) DeployService(svc dao.ServiceDeploymentRequest, serviceId *string) error {
	ret := _m.Called(svc, serviceId)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ServiceDeploymentRequest, *string) error); ok {
		r0 = rf(svc, serviceId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateService provides a mock function with given fields: svc, unused
func (_m *ControlPlane) UpdateService(svc service.Service, unused *int) error {
	ret := _m.Called(svc, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(service.Service, *int) error); ok {
		r0 = rf(svc, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MigrateServices provides a mock function with given fields: request, unused
func (_m *ControlPlane) MigrateServices(request dao.ServiceMigrationRequest, unused *int) error {
	ret := _m.Called(request, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ServiceMigrationRequest, *int) error); ok {
		r0 = rf(request, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveService provides a mock function with given fields: serviceId, unused
func (_m *ControlPlane) RemoveService(serviceId string, unused *int) error {
	ret := _m.Called(serviceId, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *int) error); ok {
		r0 = rf(serviceId, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetService provides a mock function with given fields: serviceId, svc
func (_m *ControlPlane) GetService(serviceId string, svc *service.Service) error {
	ret := _m.Called(serviceId, svc)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *service.Service) error); ok {
		r0 = rf(serviceId, svc)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetServices provides a mock function with given fields: request, services
func (_m *ControlPlane) GetServices(request dao.ServiceRequest, services *[]service.Service) error {
	ret := _m.Called(request, services)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ServiceRequest, *[]service.Service) error); ok {
		r0 = rf(request, services)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FindChildService provides a mock function with given fields: request, svc
func (_m *ControlPlane) FindChildService(request dao.FindChildRequest, svc *service.Service) error {
	ret := _m.Called(request, svc)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.FindChildRequest, *service.Service) error); ok {
		r0 = rf(request, svc)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetTaggedServices provides a mock function with given fields: request, services
func (_m *ControlPlane) GetTaggedServices(request dao.ServiceRequest, services *[]service.Service) error {
	ret := _m.Called(request, services)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ServiceRequest, *[]service.Service) error); ok {
		r0 = rf(request, services)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AssignIPs provides a mock function with given fields: assignmentRequest, unused
func (_m *ControlPlane) AssignIPs(assignmentRequest addressassignment.AssignmentRequest, unused *int) error {
	ret := _m.Called(assignmentRequest, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(addressassignment.AssignmentRequest, *int) error); ok {
		r0 = rf(assignmentRequest, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StartService provides a mock function with given fields: request, affected
func (_m *ControlPlane) StartService(request dao.ScheduleServiceRequest, affected *int) error {
	ret := _m.Called(request, affected)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ScheduleServiceRequest, *int) error); ok {
		r0 = rf(request, affected)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RestartService provides a mock function with given fields: request, affected
func (_m *ControlPlane) RestartService(request dao.ScheduleServiceRequest, affected *int) error {
	ret := _m.Called(request, affected)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ScheduleServiceRequest, *int) error); ok {
		r0 = rf(request, affected)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StopService provides a mock function with given fields: request, affected
func (_m *ControlPlane) StopService(request dao.ScheduleServiceRequest, affected *int) error {
	ret := _m.Called(request, affected)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ScheduleServiceRequest, *int) error); ok {
		r0 = rf(request, affected)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StopRunningInstance provides a mock function with given fields: request, unused
func (_m *ControlPlane) StopRunningInstance(request dao.HostServiceRequest, unused *int) error {
	ret := _m.Called(request, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.HostServiceRequest, *int) error); ok {
		r0 = rf(request, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WaitService provides a mock function with given fields: request, unused
func (_m *ControlPlane) WaitService(request dao.WaitServiceRequest, unused *int) error {
	ret := _m.Called(request, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.WaitServiceRequest, *int) error); ok {
		r0 = rf(request, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateServiceState provides a mock function with given fields: state, unused
func (_m *ControlPlane) UpdateServiceState(state servicestate.ServiceState, unused *int) error {
	ret := _m.Called(state, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(servicestate.ServiceState, *int) error); ok {
		r0 = rf(state, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetServiceStatus provides a mock function with given fields: serviceID, statusmap
func (_m *ControlPlane) GetServiceStatus(serviceID string, statusmap *map[string]dao.ServiceStatus) error {
	ret := _m.Called(serviceID, statusmap)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *map[string]dao.ServiceStatus) error); ok {
		r0 = rf(serviceID, statusmap)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetServiceStates provides a mock function with given fields: serviceId, states
func (_m *ControlPlane) GetServiceStates(serviceId string, states *[]servicestate.ServiceState) error {
	ret := _m.Called(serviceId, states)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *[]servicestate.ServiceState) error); ok {
		r0 = rf(serviceId, states)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetServiceLogs provides a mock function with given fields: serviceId, logs
func (_m *ControlPlane) GetServiceLogs(serviceId string, logs *string) error {
	ret := _m.Called(serviceId, logs)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *string) error); ok {
		r0 = rf(serviceId, logs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetServiceStateLogs provides a mock function with given fields: request, logs
func (_m *ControlPlane) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	ret := _m.Called(request, logs)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ServiceStateRequest, *string) error); ok {
		r0 = rf(request, logs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetRunningServices provides a mock function with given fields: request, runningServices
func (_m *ControlPlane) GetRunningServices(request dao.EntityRequest, runningServices *[]dao.RunningService) error {
	ret := _m.Called(request, runningServices)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.EntityRequest, *[]dao.RunningService) error); ok {
		r0 = rf(request, runningServices)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetRunningServicesForHost provides a mock function with given fields: hostId, runningServices
func (_m *ControlPlane) GetRunningServicesForHost(hostId string, runningServices *[]dao.RunningService) error {
	ret := _m.Called(hostId, runningServices)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *[]dao.RunningService) error); ok {
		r0 = rf(hostId, runningServices)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetRunningServicesForService provides a mock function with given fields: serviceId, runningServices
func (_m *ControlPlane) GetRunningServicesForService(serviceId string, runningServices *[]dao.RunningService) error {
	ret := _m.Called(serviceId, runningServices)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *[]dao.RunningService) error); ok {
		r0 = rf(serviceId, runningServices)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Action provides a mock function with given fields: request, unused
func (_m *ControlPlane) Action(request dao.AttachRequest, unused *int) error {
	ret := _m.Called(request, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.AttachRequest, *int) error); ok {
		r0 = rf(request, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetHostMemoryStats provides a mock function with given fields: req, stats
func (_m *ControlPlane) GetHostMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	ret := _m.Called(req, stats)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.MetricRequest, *metrics.MemoryUsageStats) error); ok {
		r0 = rf(req, stats)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetServiceMemoryStats provides a mock function with given fields: req, stats
func (_m *ControlPlane) GetServiceMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	ret := _m.Called(req, stats)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.MetricRequest, *metrics.MemoryUsageStats) error); ok {
		r0 = rf(req, stats)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetInstanceMemoryStats provides a mock function with given fields: req, stats
func (_m *ControlPlane) GetInstanceMemoryStats(req dao.MetricRequest, stats *[]metrics.MemoryUsageStats) error {
	ret := _m.Called(req, stats)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.MetricRequest, *[]metrics.MemoryUsageStats) error); ok {
		r0 = rf(req, stats)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetSystemUser provides a mock function with given fields: unused, usr
func (_m *ControlPlane) GetSystemUser(unused int, usr *user.User) error {
	ret := _m.Called(unused, usr)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, *user.User) error); ok {
		r0 = rf(unused, usr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateCredentials provides a mock function with given fields: usr, result
func (_m *ControlPlane) ValidateCredentials(usr user.User, result *bool) error {
	ret := _m.Called(usr, result)

	var r0 error
	if rf, ok := ret.Get(0).(func(user.User, *bool) error); ok {
		r0 = rf(usr, result)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LogHealthCheck provides a mock function with given fields: result, unused
func (_m *ControlPlane) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	ret := _m.Called(result, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(domain.HealthCheckResult, *int) error); ok {
		r0 = rf(result, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ServicedHealthCheck provides a mock function with given fields: IServiceNames, results
func (_m *ControlPlane) ServicedHealthCheck(IServiceNames []string, results *[]dao.IServiceHealthResult) error {
	ret := _m.Called(IServiceNames, results)

	var r0 error
	if rf, ok := ret.Get(0).(func([]string, *[]dao.IServiceHealthResult) error); ok {
		r0 = rf(IServiceNames, results)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReportHealthStatus provides a mock function with given fields: req, unused
func (_m *ControlPlane) ReportHealthStatus(req dao.HealthStatusRequest, unused *int) error {
	ret := _m.Called(req, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.HealthStatusRequest, *int) error); ok {
		r0 = rf(req, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReportInstanceDead provides a mock function with given fields: req, unused
func (_m *ControlPlane) ReportInstanceDead(req dao.ServiceInstanceRequest, unused *int) error {
	ret := _m.Called(req, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ServiceInstanceRequest, *int) error); ok {
		r0 = rf(req, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetServicesHealth provides a mock function with given fields: unused, results
func (_m *ControlPlane) GetServicesHealth(unused int, results *map[string]map[int]map[string]health.HealthStatus) error {
	ret := _m.Called(unused, results)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, *map[string]map[int]map[string]health.HealthStatus) error); ok {
		r0 = rf(unused, results)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Backup provides a mock function with given fields: dirpath, filename
func (_m *ControlPlane) Backup(dirpath string, filename *string) error {
	ret := _m.Called(dirpath, filename)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *string) error); ok {
		r0 = rf(dirpath, filename)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AsyncBackup provides a mock function with given fields: dirpath, filename
func (_m *ControlPlane) AsyncBackup(dirpath string, filename *string) error {
	ret := _m.Called(dirpath, filename)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *string) error); ok {
		r0 = rf(dirpath, filename)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Restore provides a mock function with given fields: filename, unused
func (_m *ControlPlane) Restore(filename string, unused *int) error {
	ret := _m.Called(filename, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *int) error); ok {
		r0 = rf(filename, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AsyncRestore provides a mock function with given fields: filename, unused
func (_m *ControlPlane) AsyncRestore(filename string, unused *int) error {
	ret := _m.Called(filename, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *int) error); ok {
		r0 = rf(filename, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// TagSnapshot provides a mock function with given fields: request, unused
func (_m *ControlPlane) TagSnapshot(request dao.TagSnapshotRequest, unused *int) error {
	ret := _m.Called(request, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.TagSnapshotRequest, *int) error); ok {
		r0 = rf(request, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveSnapshotTag provides a mock function with given fields: request, snapshotID
func (_m *ControlPlane) RemoveSnapshotTag(request dao.SnapshotByTagRequest, snapshotID *string) error {
	ret := _m.Called(request, snapshotID)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.SnapshotByTagRequest, *string) error); ok {
		r0 = rf(request, snapshotID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetSnapshotByServiceIDAndTag provides a mock function with given fields: request, snapshot
func (_m *ControlPlane) GetSnapshotByServiceIDAndTag(request dao.SnapshotByTagRequest, snapshot *dao.SnapshotInfo) error {
	ret := _m.Called(request, snapshot)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.SnapshotByTagRequest, *dao.SnapshotInfo) error); ok {
		r0 = rf(request, snapshot)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ListBackups provides a mock function with given fields: dirpath, files
func (_m *ControlPlane) ListBackups(dirpath string, files *[]dao.BackupFile) error {
	ret := _m.Called(dirpath, files)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *[]dao.BackupFile) error); ok {
		r0 = rf(dirpath, files)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BackupStatus provides a mock function with given fields: unused, status
func (_m *ControlPlane) BackupStatus(unused dao.EntityRequest, status *string) error {
	ret := _m.Called(unused, status)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.EntityRequest, *string) error); ok {
		r0 = rf(unused, status)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Snapshot provides a mock function with given fields: req, snapshotID
func (_m *ControlPlane) Snapshot(req dao.SnapshotRequest, snapshotID *string) error {
	ret := _m.Called(req, snapshotID)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.SnapshotRequest, *string) error); ok {
		r0 = rf(req, snapshotID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Rollback provides a mock function with given fields: req, unused
func (_m *ControlPlane) Rollback(req dao.RollbackRequest, unused *int) error {
	ret := _m.Called(req, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.RollbackRequest, *int) error); ok {
		r0 = rf(req, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteSnapshot provides a mock function with given fields: snapshotID, unused
func (_m *ControlPlane) DeleteSnapshot(snapshotID string, unused *int) error {
	ret := _m.Called(snapshotID, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *int) error); ok {
		r0 = rf(snapshotID, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteSnapshots provides a mock function with given fields: serviceID, unused
func (_m *ControlPlane) DeleteSnapshots(serviceID string, unused *int) error {
	ret := _m.Called(serviceID, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *int) error); ok {
		r0 = rf(serviceID, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ListSnapshots provides a mock function with given fields: serviceID, snapshots
func (_m *ControlPlane) ListSnapshots(serviceID string, snapshots *[]dao.SnapshotInfo) error {
	ret := _m.Called(serviceID, snapshots)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *[]dao.SnapshotInfo) error); ok {
		r0 = rf(serviceID, snapshots)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ResetRegistry provides a mock function with given fields: unused, unused_
func (_m *ControlPlane) ResetRegistry(unused dao.EntityRequest, unused_ *int) error {
	ret := _m.Called(unused, unused_)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.EntityRequest, *int) error); ok {
		r0 = rf(unused, unused_)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RepairRegistry provides a mock function with given fields: unused, unused_
func (_m *ControlPlane) RepairRegistry(unused dao.EntityRequest, unused_ *int) error {
	ret := _m.Called(unused, unused_)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.EntityRequest, *int) error); ok {
		r0 = rf(unused, unused_)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReadyDFS provides a mock function with given fields: serviceID, unused
func (_m *ControlPlane) ReadyDFS(serviceID string, unused *int) error {
	ret := _m.Called(serviceID, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *int) error); ok {
		r0 = rf(serviceID, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
