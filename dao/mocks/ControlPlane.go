package mocks

import "github.com/control-center/serviced/dao"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/domain"
import "github.com/control-center/serviced/domain/addressassignment"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/health"
import "github.com/control-center/serviced/metrics"

type ControlPlane struct {
	mock.Mock
}

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
func (_m *ControlPlane) GetServiceStatus(serviceID string, status *[]service.Instance) error {
	ret := _m.Called(serviceID, status)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *[]service.Instance) error); ok {
		r0 = rf(serviceID, status)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
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
func (_m *ControlPlane) Backup(backupRequest dao.BackupRequest, filename *string) error {
	ret := _m.Called(backupRequest, filename)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.BackupRequest, *string) error); ok {
		r0 = rf(backupRequest, filename)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ControlPlane) AsyncBackup(backupRequest dao.BackupRequest, filename *string) error {
	ret := _m.Called(backupRequest, filename)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.BackupRequest, *string) error); ok {
		r0 = rf(backupRequest, filename)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
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
