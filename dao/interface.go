// Copyright 2014 The Serviced Authors.
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

package dao

import (
	"time"

	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/metrics"
)

// A generic ControlPlane error
type ControlPlaneError struct {
	Msg string
}

// Implement the Error() interface for ControlPlaneError
func (s ControlPlaneError) Error() string {
	return s.Msg
}

// An request for a control center object.
type EntityRequest interface{}

type ServiceRequest struct {
	Tags         []string
	TenantID     string
	UpdatedSince time.Duration
	NameRegex    string
}

type ServiceCloneRequest struct {
	ServiceID string
	Suffix    string
}

// Only use one of ScriptName or ScriptBody. If both are specified, ScriptBody has precedence.
type RunMigrationScriptRequest struct {
	ServiceID  string // The ID of the service to migrate.
	ScriptBody string // The content of the service migration script to use.
	ScriptName string // The name of the service migration script in the docker image for the specified service.
	SDKVersion string // The version of the service migration SDK to use.
	DryRun     bool
}

type ServiceMigrationRequest struct {
	ServiceID string
	Modified  []*service.Service
	Added     []*service.Service
	Deploy    []*ServiceDeploymentRequest
	DryRun    bool
}

type ServiceStateRequest struct {
	ServiceID      string
	ServiceStateID string
}

type ScheduleServiceRequest struct {
	ServiceID  string
	AutoLaunch bool
}

type WaitServiceRequest struct {
	ServiceIDs   []string             // List of service IDs to monitor
	DesiredState service.DesiredState // State which to monitor for
	Timeout      time.Duration        // Time to wait before cancelling the subprocess
}

type HostServiceRequest struct {
	HostID         string
	ServiceStateID string
}

type AttachRequest struct {
	Running *RunningService
	Command string
	Args    []string
}

type FindChildRequest struct {
	ServiceID string
	ChildName string
}

type SnapshotRequest struct {
	ServiceID   string
	Message     string
	Tag         string
	ContainerID string
}

type TagSnapshotRequest struct {
	SnapshotID string
	TagName    string
}

type SnapshotByTagRequest struct {
	ServiceID string
	TagName   string
}

type RollbackRequest struct {
	SnapshotID   string
	ForceRestart bool
}

type MetricRequest struct {
	StartTime time.Time
	HostID    string
	ServiceID string
	Instances []metrics.ServiceInstance
}

// The ControlPlane interface is the API for a serviced master.
type ControlPlane interface {

	//---------------------------------------------------------------------------
	// Service CRUD

	//for a service, get it's tenant Id
	GetTenantId(serviceId string, tenantId *string) error

	// Add a new service
	AddService(service service.Service, serviceId *string) error

	// Clones a new service
	CloneService(request ServiceCloneRequest, serviceId *string) error

	// Deploy a new service
	DeployService(service ServiceDeploymentRequest, serviceId *string) error

	// Update an existing service
	UpdateService(service service.Service, unused *int) error

	// Migrate a service definition
	RunMigrationScript(request RunMigrationScriptRequest, unused *int) error

	// Migrate a service definition
	MigrateServices(request ServiceMigrationRequest, unused *int) error

	// Remove a service definition
	RemoveService(serviceId string, unused *int) error

	// Get a service from serviced
	GetService(serviceId string, service *service.Service) error

	// Get a list of services from serviced
	GetServices(request ServiceRequest, services *[]service.Service) error

	// Find a child service with the given name
	FindChildService(request FindChildRequest, service *service.Service) error

	// Get services with the given tag(s)
	GetTaggedServices(request ServiceRequest, services *[]service.Service) error

	// Assign IP addresses to all services at and below the provided service
	AssignIPs(assignmentRequest AssignmentRequest, _ *int) (err error)

	// Get the IP addresses assigned to an service
	GetServiceAddressAssignments(serviceID string, addresses *[]addressassignment.AddressAssignment) error

	//---------------------------------------------------------------------------
	//ServiceState CRUD

	// Schedule the given service to start
	StartService(request ScheduleServiceRequest, affected *int) error

	// Schedule the given service to restart
	RestartService(request ScheduleServiceRequest, affected *int) error

	// Schedule the given service to stop
	StopService(request ScheduleServiceRequest, affected *int) error

	// Stop a running instance of a service
	StopRunningInstance(request HostServiceRequest, unused *int) error

	// Wait for a particular service state
	WaitService(request WaitServiceRequest, _ *int) error

	// Update the service state
	UpdateServiceState(state servicestate.ServiceState, unused *int) error

	// Computes the status of the service based on its service instances
	GetServiceStatus(serviceID string, statusmap *map[string]ServiceStatus) error

	// Get the services instances for a given service
	GetServiceStates(serviceId string, states *[]servicestate.ServiceState) error

	// Get logs for the given app
	GetServiceLogs(serviceId string, logs *string) error

	// Get logs for the given app
	GetServiceStateLogs(request ServiceStateRequest, logs *string) error

	// Get all running services
	GetRunningServices(request EntityRequest, runningServices *[]RunningService) error

	// Get the services instances for a given service
	GetRunningServicesForHost(hostId string, runningServices *[]RunningService) error

	// Get the service instances for a given service
	GetRunningServicesForService(serviceId string, runningServices *[]RunningService) error

	// Attach to a running container with a predefined action
	Action(request AttachRequest, unused *int) error

	// ------------------------------------------------------------------------
	// Metrics

	// Get service memory stats for a particular host
	GetHostMemoryStats(req MetricRequest, stats *metrics.MemoryUsageStats) error

	// Get service memory stats for a particular service
	GetServiceMemoryStats(req MetricRequest, stats *metrics.MemoryUsageStats) error

	// Get service memory stats for a particular service instance
	GetInstanceMemoryStats(req MetricRequest, stats *[]metrics.MemoryUsageStats) error

	//---------------------------------------------------------------------------
	// ServiceTemplate CRUD

	// Deploy an application template in to production
	DeployTemplate(request ServiceTemplateDeploymentRequest, tenantIDs *[]string) error

	// Add a new service Template
	AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, templateId *string) error

	// Update a new service Template
	UpdateServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, unused *int) error

	// Update a new service Template
	RemoveServiceTemplate(serviceTemplateID string, unused *int) error

	// Get a list of ServiceTemplates
	GetServiceTemplates(unused int, serviceTemplates *map[string]servicetemplate.ServiceTemplate) error

	//---------------------------------------------------------------------------
	// Service CRUD

	//GetSystemUser retrieves the credentials for the system_user account
	GetSystemUser(unused int, user *user.User) error

	//ValidateCredentials verifies if the passed in user has the correct username and password
	ValidateCredentials(user user.User, result *bool) error

	// Register a health check result
	LogHealthCheck(result domain.HealthCheckResult, unused *int) error

	// Check the health of control center
	ServicedHealthCheck(IServiceNames []string, results *[]IServiceHealthResult) error

	// -----------------------------------------------------------------------
	// Filesystem CRUD

	// Backup captures the state of the application stack and writes the output
	// to disk.
	Backup(dirpath string, filename *string) (err error)

	// AsyncBackup is the same as backup but asynchronous
	AsyncBackup(dirpath string, filename *string) (err error)

	// Restore reverts the full application stack from a backup file
	Restore(filename string, _ *int) (err error)

	// AsyncRestore is the same as restore but asynchronous
	AsyncRestore(filename string, _ *int) (err error)

	// Adds 1 or more tags to an existing snapshot
	TagSnapshot(request TagSnapshotRequest, _ *int) error

	// Removes a specific tag from an existing snapshot
	RemoveSnapshotTag(request SnapshotByTagRequest, snapshotID *string) error

	// Gets the snapshot from a specific service with a specific tag
	GetSnapshotByServiceIDAndTag(request SnapshotByTagRequest, snapshot *SnapshotInfo) error

	// ListBackups returns the list of backups
	ListBackups(dirpath string, files *[]BackupFile) (err error)

	// BackupStatus returns the current status of a running backup or restore
	BackupStatus(_ EntityRequest, status *string) (err error)

	// Snapshot captures the state of a single application
	Snapshot(req SnapshotRequest, snapshotID *string) (err error)

	// Rollback reverts a single application to the state of a snapshot
	Rollback(req RollbackRequest, _ *int) (err error)

	// DeleteSnapshot deletes a single snapshot
	DeleteSnapshot(snapshotID string, _ *int) (err error)

	// DeleteSnapshots deletes all snapshots for a service
	DeleteSnapshots(serviceID string, _ *int) (err error)

	// ListSnapshots returns a list of all snapshots for a service
	ListSnapshots(serviceID string, snapshots *[]SnapshotInfo) (err error)

	// ResetRegistry prompts all images to be re-pushed into the docker
	// registry.
	ResetRegistry(_ EntityRequest, _ *int) (err error)

	// RepairRegistry will try to recover the latest image of all service
	// images from the docker registry and save it to the index.
	RepairRegistry(_ EntityRequest, _ *int) (err error)

	// ReadyDFS waits for the DFS to be idle when creating a service shell.
	ReadyDFS(serviceID string, _ *int) (err error)
}
