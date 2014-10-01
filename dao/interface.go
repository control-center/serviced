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
	"github.com/control-center/serviced/volume"
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

type ServiceStateRequest struct {
	ServiceID      string
	ServiceStateID string
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

// The ControlPlane interface is the API for a serviced master.
type ControlPlane interface {

	//---------------------------------------------------------------------------
	// Service CRUD

	//for a service, get it's tenant Id
	GetTenantId(serviceId string, tenantId *string) error

	// Add a new service
	AddService(service service.Service, serviceId *string) error

	// Deploy a new service
	DeployService(service ServiceDeploymentRequest, serviceId *string) error

	// Update an existing service
	UpdateService(service service.Service, unused *int) error

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

	// Find all service endpoint matches
	GetServiceEndpoints(serviceId string, response *map[string][]ApplicationEndpoint) error

	// Assign IP addresses to all services at and below the provided service
	AssignIPs(assignmentRequest AssignmentRequest, _ *struct{}) (err error)

	// Get the IP addresses assigned to an service
	GetServiceAddressAssignments(serviceID string, addresses *[]addressassignment.AddressAssignment) error

	//---------------------------------------------------------------------------
	//ServiceState CRUD

	// Schedule the given service to start
	StartService(serviceId string, unused *string) error

	// Schedule the given service to restart
	RestartService(serviceId string, unused *int) error

	// Schedule the given service to stop
	StopService(serviceId string, unused *int) error

	// Stop a running instance of a service
	StopRunningInstance(request HostServiceRequest, unused *int) error

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

	//---------------------------------------------------------------------------
	// ServiceTemplate CRUD

	// Deploy an application template in to production
	DeployTemplate(request ServiceTemplateDeploymentRequest, tenantId *string) error

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

	// Rollback DFS and service image
	Rollback(snapshotId string, unused *int) error

	// Commit DFS and service image
	Commit(containerId string, label *string) error

	// Performs a local snapshot from the host
	TakeSnapshot(serviceId string, label *string) error

	// Snapshots DFS and service image
	Snapshot(serviceId string, label *string) error

	// Delete a snapshot
	DeleteSnapshot(snapshotId string, unused *int) error

	// List available snapshots
	Snapshots(serviceId string, snapshotIds *[]string) error

	// Delete snapshots for a given service
	DeleteSnapshots(serviceId string, unused *int) error

	// Get the DFS volume
	GetVolume(serviceId string, theVolume *volume.Volume) error

	//GetSystemUser retrieves the credentials for the system_user account
	GetSystemUser(unused int, user *user.User) error

	//ValidateCredentials verifies if the passed in user has the correct username and password
	ValidateCredentials(user user.User, result *bool) error

	// Waits for the DFS to be ready
	ReadyDFS(bool, *int) error

	// Write a tgz file containing all templates and services
	Backup(backupDirectory string, backupFilePath *string) error

	// Restore templates and services from a tgz file (inverse of Backup)
	Restore(backupFilePath string, unused *int) error

	// Register a health check result
	LogHealthCheck(result domain.HealthCheckResult, unused *int) error

	// Return the number of layers in an image
	ImageLayerCount(imageUUID string, layers *int) error
}
