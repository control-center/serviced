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

import "github.com/control-center/serviced/dao"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/domain"
import "github.com/control-center/serviced/domain/addressassignment"
import "github.com/control-center/serviced/domain/applicationendpoint"
import "github.com/control-center/serviced/domain/service"
import "github.com/control-center/serviced/domain/servicestate"
import "github.com/control-center/serviced/domain/servicetemplate"
import "github.com/control-center/serviced/domain/user"
import "github.com/control-center/serviced/metrics"

type ControlPlane struct {
	mock.Mock
}

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
func (_m *ControlPlane) DeployService(service dao.ServiceDeploymentRequest, serviceId *string) error {
	ret := _m.Called(service, serviceId)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ServiceDeploymentRequest, *string) error); ok {
		r0 = rf(service, serviceId)
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
func (_m *ControlPlane) RunMigrationScript(request dao.RunMigrationScriptRequest, unused *int) error {
	ret := _m.Called(request, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.RunMigrationScriptRequest, *int) error); ok {
		r0 = rf(request, unused)
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
func (_m *ControlPlane) GetServiceEndpoints(serviceId string, response *map[string][]applicationendpoint.ApplicationEndpoint) error {
	ret := _m.Called(serviceId, response)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *map[string][]applicationendpoint.ApplicationEndpoint) error); ok {
		r0 = rf(serviceId, response)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ControlPlane) AssignIPs(assignmentRequest dao.AssignmentRequest, unused *int) error {
	ret := _m.Called(assignmentRequest, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.AssignmentRequest, *int) error); ok {
		r0 = rf(assignmentRequest, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ControlPlane) GetServiceAddressAssignments(serviceID string, addresses *[]addressassignment.AddressAssignment) error {
	ret := _m.Called(serviceID, addresses)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *[]addressassignment.AddressAssignment) error); ok {
		r0 = rf(serviceID, addresses)
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
func (_m *ControlPlane) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, tenantIDs *[]string) error {
	ret := _m.Called(request, tenantIDs)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.ServiceTemplateDeploymentRequest, *[]string) error); ok {
		r0 = rf(request, tenantIDs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ControlPlane) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, templateId *string) error {
	ret := _m.Called(serviceTemplate, templateId)

	var r0 error
	if rf, ok := ret.Get(0).(func(servicetemplate.ServiceTemplate, *string) error); ok {
		r0 = rf(serviceTemplate, templateId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ControlPlane) UpdateServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, unused *int) error {
	ret := _m.Called(serviceTemplate, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(servicetemplate.ServiceTemplate, *int) error); ok {
		r0 = rf(serviceTemplate, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ControlPlane) RemoveServiceTemplate(serviceTemplateID string, unused *int) error {
	ret := _m.Called(serviceTemplateID, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *int) error); ok {
		r0 = rf(serviceTemplateID, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ControlPlane) GetServiceTemplates(unused int, serviceTemplates *map[string]servicetemplate.ServiceTemplate) error {
	ret := _m.Called(unused, serviceTemplates)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, *map[string]servicetemplate.ServiceTemplate) error); ok {
		r0 = rf(unused, serviceTemplates)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
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
func (_m *ControlPlane) BackupStatus(req dao.EntityRequest, status *string) error {
	ret := _m.Called(req, status)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.EntityRequest, *string) error); ok {
		r0 = rf(req, status)
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
func (_m *ControlPlane) ResetRegistry(req dao.EntityRequest, unused *int) error {
	ret := _m.Called(req, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.EntityRequest, *int) error); ok {
		r0 = rf(req, unused)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ControlPlane) RepairRegistry(req dao.EntityRequest, unused *int) error {
	ret := _m.Called(req, unused)

	var r0 error
	if rf, ok := ret.Get(0).(func(dao.EntityRequest, *int) error); ok {
		r0 = rf(req, unused)
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
