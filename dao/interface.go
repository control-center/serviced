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

// --------------------------------------------------------------------------------------------------
// --------------------------------------------------------------------------------------------------
//               **** USE OF THE METHODS IN THIS FILE IS DEPRECATED ****
//
// THAT MEANS DO NOT ADD MORE METHODS TO dao.ControlPlane
//
// Instead of adding new RPC calls via dao.ControlPlane, new RPCs should be added
// rpc/master.ClientInterface
// --------------------------------------------------------------------------------------------------
// --------------------------------------------------------------------------------------------------

import (
	"time"

	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/metrics"
)

// ControlPlaneError is a generic ControlPlane error
type ControlPlaneError struct {
	Msg string
}

// Implement the Error() interface for ControlPlaneError
func (s ControlPlaneError) Error() string {
	return s.Msg
}

// EntityRequest is a request for a control center object.
type EntityRequest interface{}

// ServiceRequest identifies a service plus some query parameters.
type ServiceRequest struct {
	Tags         []string
	TenantID     string
	UpdatedSince time.Duration
	NameRegex    string
}

// ServiceCloneRequest specifies a service to clone and how to modify the clone's name.
type ServiceCloneRequest struct {
	ServiceID string
	Suffix    string
}

// ServiceMigrationRequest is request to modify one or more services.
type ServiceMigrationRequest struct {
	ServiceID  string                         // The tenant service ID
	Modified   []*service.Service             // Services modified by the migration
	Added      []*service.Service             // Services added by the migration
	Deploy     []*ServiceDeploymentRequest    // ServiceDefinitions to be deployed by the migration
	LogFilters map[string]logfilter.LogFilter // LogFilters to add/replace
}

// ServiceStateRequest specifies a request for a service's service state.
type ServiceStateRequest struct {
	ServiceID      string
	ServiceStateID string
}

// ScheduleServiceRequest specifies a request to schedule a service to run.
type ScheduleServiceRequest struct {
	ServiceIDs  []string
	AutoLaunch  bool
	Synchronous bool
}

// WaitServiceRequest is a request to wait for a set of services to gain the requested status.
type WaitServiceRequest struct {
	ServiceIDs   []string             // List of service IDs to monitor
	DesiredState service.DesiredState // State which to monitor for
	Timeout      time.Duration        // Time to wait before cancelling the subprocess
	Recursive    bool                 // Recursively wait for the desired state
}

// HostServiceRequest is a request for the service state of a host.
type HostServiceRequest struct {
	HostID         string
	ServiceStateID string
}

// AttachRequest is a request to run a command in the container of a running service.
type AttachRequest struct {
	Running *RunningService
	Command string
	Args    []string
}

// FindChildRequest is a request to locate a service's child by name.
type FindChildRequest struct {
	ServiceID string
	ChildName string
}

// SnapshotRequest is a request to create a snapshot.
type SnapshotRequest struct {
	ServiceID            string
	Message              string
	Tag                  string
	ContainerID          string
	SnapshotSpacePercent int
}

// TagSnapshotRequest is a request to add a tag (label) to the specified snapshot.
type TagSnapshotRequest struct {
	SnapshotID string
	TagName    string
}

// SnapshotByTagRequest is request for the snapshot idenfified by the tag name.
type SnapshotByTagRequest struct {
	ServiceID string
	TagName   string
}

// RollbackRequest is a request to apply a snapshot to the current system.
type RollbackRequest struct {
	SnapshotID   string
	ForceRestart bool
}

// MetricRequest is a request for the metrics of the instances of a service.
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

	// Add a new service
	AddService(svc service.Service, serviceID *string) error

	// Clones a new service
	CloneService(request ServiceCloneRequest, serviceID *string) error

	// Deploy a new service
	DeployService(svc ServiceDeploymentRequest, serviceID *string) error

	// Update an existing service
	UpdateService(svc service.Service, _ *int) error

	// Migrate a service definition
	MigrateServices(request ServiceMigrationRequest, _ *int) error

	// Remove a service definition
	RemoveService(serviceID string, _ *int) error

	// Get a service from serviced
	GetService(serviceID string, svc *service.Service) error

	// Find a child service with the given name
	FindChildService(request FindChildRequest, svc *service.Service) error

	// Assign IP addresses to all services at and below the provided service
	AssignIPs(assignmentRequest addressassignment.AssignmentRequest, _ *int) (err error)

	// Get a list of tenant IDs
	GetTenantIDs(_ struct{}, tenantIDs *[]string) error

	//---------------------------------------------------------------------------
	//ServiceState CRUD

	// Schedule the given service to start
	StartService(request ScheduleServiceRequest, affected *int) error

	// Schedule the given service to restart
	RestartService(request ScheduleServiceRequest, affected *int) error

	// Schedule the given service to rebalance
	RebalanceService(request ScheduleServiceRequest, affected *int) error

	// Schedule the given service to stop
	StopService(request ScheduleServiceRequest, affected *int) error

	// Schedule the given service to pause
	PauseService(request ScheduleServiceRequest, affected *int) error

	// Stop a running instance of a service
	StopRunningInstance(request HostServiceRequest, _ *int) error

	// Wait for a particular service state
	WaitService(request WaitServiceRequest, _ *int) error

	// Computes the status of the service based on its service instances
	GetServiceStatus(serviceID string, status *[]service.Instance) error

	// Get logs for the given app
	GetServiceLogs(serviceID string, logs *string) error

	// Get logs for the given app
	GetServiceStateLogs(request ServiceStateRequest, logs *string) error

	// Get all running services
	GetRunningServices(request EntityRequest, runningServices *[]RunningService) error

	// Get the services instances for a given service
	GetRunningServicesForHost(hostID string, runningServices *[]RunningService) error

	// Get the service instances for a given service
	GetRunningServicesForService(serviceID string, runningServices *[]RunningService) error

	// Attach to a running container with a predefined action
	Action(request AttachRequest, _ *int) error

	// ------------------------------------------------------------------------
	// Metrics

	// Get service memory stats for a particular host
	GetHostMemoryStats(req MetricRequest, stats *metrics.MemoryUsageStats) error

	// Get service memory stats for a particular service
	GetServiceMemoryStats(req MetricRequest, stats *metrics.MemoryUsageStats) error

	// Get service memory stats for a particular service instance
	GetInstanceMemoryStats(req MetricRequest, stats *[]metrics.MemoryUsageStats) error

	// -----------------------------------------------------------------------
	// Filesystem CRUD

	// Backup captures the state of the application stack and writes the output
	// to disk.
	Backup(backupRequest BackupRequest, filename *string) (err error)

	// GetBackupEstimate estimates space required to take backup and space available
	GetBackupEstimate(backupRequest BackupRequest, estimate *BackupEstimate) (err error)

	// AsyncBackup is the same as backup but asynchronous
	AsyncBackup(backupRequest BackupRequest, filename *string) (err error)

	// Restore reverts the full application stack from a backup file
	Restore(restoreRequest RestoreRequest, _ *int) (err error)

	// AsyncRestore is the same as restore but asynchronous
	AsyncRestore(restoreRequest RestoreRequest, _ *int) (err error)

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
