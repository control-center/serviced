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

package test

import (
	"unsafe"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/volume"

	"github.com/stretchr/testify/mock"
)

// assert the interface
var _ dao.ControlPlane = &MockControlPlane{}

type MockControlPlane struct {
	mock.Mock

	//
	// The methods on the ControlPlane interface all take 2 arguments.
	// The names and types vary, but the first is request argument and the second is
	// the response from the function. In many cases, the callers of ControlPlane
	// are passing pointers to local variables for the response argument, which makes
	// those cases impossible to mock using the native capabilities of the testify.Mock
	//
	// Sooo, for cases like that the caller can store the values they want for the response
	// in this map.
	//
	// NOTE: Not all methods below have been updated to use Responses, so depending on what
	//       you are trying to test, you may need to update one or more methods below to use
	//       Responses.
	Responses map[string]unsafe.Pointer
}

func New() *MockControlPlane {
	mcp := &MockControlPlane{}
	mcp.Responses = make(map[string]unsafe.Pointer)
	return mcp
}

// Helper method that returns the arguments for first call to the specified mock
func (mcp *MockControlPlane) GetArgsForMockCall(methodName string) mock.Arguments {
	for _, call := range mcp.Calls {
		if call.Method == methodName {
			return call.Arguments
		}
	}
	return nil
}

//for a service, get it's tenant Id
func (mcp *MockControlPlane) GetTenantId(serviceId string, tenantId *string) error {
	if mcp.Responses["GetTenantId"] != nil {
		*tenantId = *((*string)(mcp.Responses["GetTenantId"]))
	}
	return mcp.Mock.Called(serviceId, tenantId).Error(0)
}

// Add a new service
func (mcp *MockControlPlane) AddService(service service.Service, serviceId *string) error {
	if mcp.Responses["AddService"] != nil {
		*serviceId = *((*string)(mcp.Responses["AddService"]))
	}
	return mcp.Mock.Called(service, serviceId).Error(0)
}

// Clone a new service
func (mcp *MockControlPlane) CloneService(request dao.ServiceCloneRequest, serviceId *string) error {
	if mcp.Responses["CloneService"] != nil {
		*serviceId = *((*string)(mcp.Responses["CloneService"]))
	}
	return mcp.Mock.Called(request, serviceId).Error(0)
}

// Deploy a new service
func (mcp *MockControlPlane) DeployService(service dao.ServiceDeploymentRequest, serviceId *string) error {
	if mcp.Responses["DeployService"] != nil {
		*serviceId = *((*string)(mcp.Responses["DeployService"]))
	}
	return mcp.Mock.Called(service, serviceId).Error(0)
}

// Update an existing service
func (mcp *MockControlPlane) UpdateService(service service.Service, unused *int) error {
	return mcp.Mock.Called(service, unused).Error(0)
}

// Migrate a service definition
func (mcp *MockControlPlane) MigrateService(request dao.ServiceMigrationRequest, unused *int) error {
	return mcp.Mock.Called(request, unused).Error(0)
}

// Remove a service definition
func (mcp *MockControlPlane) RemoveService(serviceId string, unused *int) error {
	return mcp.Mock.Called(serviceId, unused).Error(0)
}

// Get a service from serviced
func (mcp *MockControlPlane) GetService(serviceId string, response *service.Service) error {
	if mcp.Responses["GetService"] != nil {
		*response = *((*service.Service)(mcp.Responses["GetService"]))
	}
	return mcp.Mock.Called(serviceId, response).Error(0)
}

// Get a list of services from serviced
func (mcp *MockControlPlane) GetServices(request dao.ServiceRequest, services *[]service.Service) error {
	if mcp.Responses["GetServices"] != nil {
		*services = *((*[]service.Service)(mcp.Responses["GetServices"]))
	}
	return mcp.Mock.Called(request, services).Error(0)
}

// Find a child service with the given name
func (mcp *MockControlPlane) FindChildService(request dao.FindChildRequest, response *service.Service) error {
	if mcp.Responses["FindChildService"] != nil {
		*response = *((*service.Service)(mcp.Responses["FindChildService"]))
	}
	return mcp.Mock.Called(request, response).Error(0)
}

// Get services with the given tag(s)
func (mcp *MockControlPlane) GetTaggedServices(request dao.ServiceRequest, services *[]service.Service) error {
	if mcp.Responses["GetTaggedServices"] != nil {
		*services = *((*[]service.Service)(mcp.Responses["GetTaggedServices"]))
	}
	return mcp.Mock.Called(request, services).Error(0)
}

// Find all service endpoint matches
func (mcp *MockControlPlane) GetServiceEndpoints(serviceId string, response *map[string][]dao.ApplicationEndpoint) error {
	return mcp.Mock.Called(serviceId, response).Error(0)
}

// Assign IP addresses to all services at and below the provided service
func (mcp *MockControlPlane) AssignIPs(assignmentRequest dao.AssignmentRequest, unused *struct{}) (err error) {
	return mcp.Mock.Called(assignmentRequest, unused).Error(0)
}

// Get the IP addresses assigned to an service
func (mcp *MockControlPlane) GetServiceAddressAssignments(serviceID string, addresses *[]addressassignment.AddressAssignment) error {
	return mcp.Mock.Called(serviceID, addresses).Error(0)
}

//---------------------------------------------------------------------------
//ServiceState CRUD

// Schedule the given service to start
func (mcp *MockControlPlane) StartService(request dao.ScheduleServiceRequest, affected *int) error {
	return mcp.Mock.Called(request, affected).Error(0)
}

// Schedule the given service to restart
func (mcp *MockControlPlane) RestartService(request dao.ScheduleServiceRequest, affected *int) error {
	return mcp.Mock.Called(request, affected).Error(0)
}

// Schedule the given service to stop
func (mcp *MockControlPlane) StopService(request dao.ScheduleServiceRequest, affected *int) error {
	return mcp.Mock.Called(request, affected).Error(0)
}

// Stop a running instance of a service
func (mcp *MockControlPlane) StopRunningInstance(request dao.HostServiceRequest, unused *int) error {
	return mcp.Mock.Called(request, unused).Error(0)
}

// Wait for a particular service state
func (mcp *MockControlPlane) WaitService(request dao.WaitServiceRequest, unused *struct{}) error {
	return mcp.Mock.Called(request, unused).Error(0)
}

// Update the service state
func (mcp *MockControlPlane) UpdateServiceState(state servicestate.ServiceState, unused *int) error {
	return mcp.Mock.Called(state, unused).Error(0)
}

// Computes the status of the service based on its service instances
func (mcp *MockControlPlane) GetServiceStatus(serviceID string, statusmap *map[string]dao.ServiceStatus) error {
	return mcp.Mock.Called(serviceID, statusmap).Error(0)
}

// Get the services instances for a given service
func (mcp *MockControlPlane) GetServiceStates(serviceId string, states *[]servicestate.ServiceState) error {
	return mcp.Mock.Called(serviceId, states).Error(0)
}

// Get logs for the given app
func (mcp *MockControlPlane) GetServiceLogs(serviceId string, logs *string) error {
	return mcp.Mock.Called(serviceId, logs).Error(0)
}

// Get logs for the given app
func (mcp *MockControlPlane) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	return mcp.Mock.Called(request, logs).Error(0)
}

// Get all running services
func (mcp *MockControlPlane) GetRunningServices(request dao.EntityRequest, runningServices *[]dao.RunningService) error {
	return mcp.Mock.Called(request, runningServices).Error(0)
}

// Get the services instances for a given service
func (mcp *MockControlPlane) GetRunningServicesForHost(hostId string, runningServices *[]dao.RunningService) error {
	return mcp.Mock.Called(hostId, runningServices).Error(0)
}

// Get the service instances for a given service
func (mcp *MockControlPlane) GetRunningServicesForService(serviceId string, runningServices *[]dao.RunningService) error {
	return mcp.Mock.Called(serviceId, runningServices).Error(0)
}

// Attach to a running container with a predefined action
func (mcp *MockControlPlane) Action(request dao.AttachRequest, unused *int) error {
	return mcp.Mock.Called(request, unused).Error(0)
}

//---------------------------------------------------------------------------
// ServiceTemplate CRUD

// Deploy an application template in to production
func (mcp *MockControlPlane) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, tenantIDs *[]string) error {
	return mcp.Mock.Called(request, tenantIDs).Error(0)
}

// Add a new service Template
func (mcp *MockControlPlane) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, templateId *string) error {
	return mcp.Mock.Called(serviceTemplate, templateId).Error(0)
}

// Update a new service Template
func (mcp *MockControlPlane) UpdateServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, unused *int) error {
	return mcp.Mock.Called(serviceTemplate, unused).Error(0)
}

// Update a new service Template
func (mcp *MockControlPlane) RemoveServiceTemplate(serviceTemplateID string, unused *int) error {
	return mcp.Mock.Called(serviceTemplateID, unused).Error(0)
}

// Get a list of ServiceTemplates
func (mcp *MockControlPlane) GetServiceTemplates(unused int, serviceTemplates *map[string]servicetemplate.ServiceTemplate) error {
	return mcp.Mock.Called(unused, serviceTemplates).Error(0)
}

//---------------------------------------------------------------------------
// Service CRUD

//GetSystemUser retrieves the credentials for the system_user account
func (mcp *MockControlPlane) GetSystemUser(unused int, user *user.User) error {
	return mcp.Mock.Called(unused, user).Error(0)
}

//ValidateCredentials verifies if the passed in user has the correct username and password
func (mcp *MockControlPlane) ValidateCredentials(user user.User, result *bool) error {
	return mcp.Mock.Called(user, result).Error(0)
}

// Register a health check result
func (mcp *MockControlPlane) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	return mcp.Mock.Called(result, unused).Error(0)
}

// Return the number of layers in an image
func (mcp *MockControlPlane) ImageLayerCount(imageUUID string, layers *int) error {
	return mcp.Mock.Called(imageUUID, layers).Error(0)
}

// Volume returns a service's volume
func (mcp *MockControlPlane) GetVolume(serviceID string, volume volume.Volume) error {
	return mcp.Mock.Called(serviceID, volume).Error(0)
}

// SetRegistry resets the path to the docker registry
func (mcp *MockControlPlane) ResetRegistry(request dao.EntityRequest, unused *int) error {
	return mcp.Mock.Called(request, unused).Error(0)
}

// Deletes a particular snapshot
func (mcp *MockControlPlane) DeleteSnapshot(snapshotID string, unused *int) error {
	return mcp.Mock.Called(snapshotID, unused).Error(0)
}

// Deletes all snapshots for a specific tenant
func (mcp *MockControlPlane) DeleteSnapshots(tenantID string, unused *int) error {
	return mcp.Mock.Called(tenantID, unused).Error(0)
}

// Rollback a service to a particular snapshot
func (mcp *MockControlPlane) Rollback(request dao.RollbackRequest, unused *int) error {
	return mcp.Mock.Called(request, unused).Error(0)
}

// Snapshot takes a snapshot of the filesystem and images
func (mcp *MockControlPlane) Snapshot(request dao.SnapshotRequest, snapshotID *string) error {
	return mcp.Mock.Called(request, snapshotID).Error(0)
}

// AsyncSnapshot performs a snapshot asynchronously
func (mcp *MockControlPlane) AsyncSnapshot(serviceID string, snapshotID *string) error {
	return mcp.Mock.Called(serviceID, snapshotID).Error(0)
}

// ListSnapshots lists all the snapshots for a particular service
func (mcp *MockControlPlane) ListSnapshots(serviceID string, snapshots *[]dao.SnapshotInfo) error {
	return mcp.Mock.Called(serviceID, snapshots).Error(0)
}

// Commit commits a docker container to a service image
func (mcp *MockControlPlane) Commit(containerID string, snapshotID *string) error {
	return mcp.Mock.Called(containerID, snapshotID).Error(0)
}

// ReadyDFS notifies whether there are any running operations
func (mcp *MockControlPlane) ReadyDFS(unused bool, unusedint *int) error {
	return mcp.Mock.Called(unused, unusedint).Error(0)
}

// ListBackups lists the backup files for a particular directory
func (mcp *MockControlPlane) ListBackups(dirpath string, files *[]dao.BackupFile) error {
	return mcp.Mock.Called(dirpath, files).Error(0)
}

// Backup backs up dfs and imagesWrite a tgz file containing all templates and services
func (mcp *MockControlPlane) Backup(dirpath string, filename *string) error {
	return mcp.Mock.Called(dirpath, filename).Error(0)
}

// AsyncBackup performs asynchronous backups
func (mcp *MockControlPlane) AsyncBackup(dirpath string, filename *string) error {
	return mcp.Mock.Called(dirpath, filename).Error(0)
}

// Restore templates and services from a tgz file (inverse of Backup)
func (mcp *MockControlPlane) Restore(filename string, unused *int) error {
	return mcp.Mock.Called(filename, unused).Error(0)
}

// AsyncRestore performs an asynchronous restore
func (mcp *MockControlPlane) AsyncRestore(filename string, unused *int) error {
	return mcp.Mock.Called(filename, unused).Error(0)
}

// BackupStatus monitors the status of a backup or restore
func (mcp *MockControlPlane) BackupStatus(unused int, status *string) error {
	return mcp.Mock.Called(unused, status).Error(0)
}

// GetHostMemoryStats returns the memory stats of a host
func (mcp *MockControlPlane) GetHostMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	return mcp.Mock.Called(req, stats).Error(0)
}

// GetHostMemoryStats returns the memory stats of a service
func (mcp *MockControlPlane) GetServiceMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	return mcp.Mock.Called(req, stats).Error(0)
}

// GetHostMemoryStats returns the memory stats of service instances
func (mcp *MockControlPlane) GetInstanceMemoryStats(req dao.MetricRequest, stats *[]metrics.MemoryUsageStats) error {
	return mcp.Mock.Called(req, stats).Error(0)
}

func (mcp *MockControlPlane) ServicedHealthCheck(IServiceNames []string, results *[]dao.IServiceHealthResult) error {
	return mcp.Mock.Called(IServiceNames, results).Error(0)
}
