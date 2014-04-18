package dao

import (
	"github.com/zenoss/serviced/volume"
)

// A generic ControlPlane error
type ControlPlaneError struct {
	Msg string
}

// Implement the Error() interface for ControlPlaneError
func (s ControlPlaneError) Error() string {
	return s.Msg
}

// An request for a control plane object.
type EntityRequest interface{}

type ServiceStateRequest struct {
	ServiceId      string
	ServiceStateId string
}

type HostServiceRequest struct {
	HostId         string
	ServiceStateId string
}

// The ControlPlane interface is the API for a serviced master.
type ControlPlane interface {

	//---------------------------------------------------------------------------
	// Host CRUD

	// Register a host with serviced
	AddHost(host Host, hostId *string) error

	// Update Host information for a registered host
	UpdateHost(host Host, ununsed *int) error

	// Remove a Host from serviced
	RemoveHost(hostId string, unused *int) error

	// Get Host by id
	GetHost(hostId string, host *Host) error

	// Get a list of registered hosts
	GetHosts(request EntityRequest, hosts *map[string]*Host) error

	//---------------------------------------------------------------------------
	// Service CRUD

	//for a service, get it's tenant Id
	GetTenantId(serviceId string, tenantId *string) error

	// Add a new service
	AddService(service Service, serviceId *string) error

	// Update an existing service
	UpdateService(service Service, unused *int) error

	// Remove a service definition
	RemoveService(serviceId string, unused *int) error

	// Get a service from serviced
	GetService(serviceId string, service *Service) error

	// Get a list of services from serviced
	GetServices(request EntityRequest, services *[]*Service) error

	// Get services with the given tag(s)
	GetTaggedServices(request EntityRequest, services *[]*Service) error

	// Find all service endpoint matches
	GetServiceEndpoints(serviceId string, response *map[string][]*ApplicationEndpoint) error

	// Deploy a service
	AddServiceDeployment(deployment ServiceDeployment, unused *int) (err error)

	// Add a virtual IP address to a pool
	AddVirtualIp(requestedVirtualIp VirtualIP, _ *struct{}) (err error)

	// Remove a virtual IP address from a pool
	RemoveVirtualIp(requestedVirtualIp VirtualIP, _ *struct{}) (err error)

	// Assign IP addresses to all services at and below the provided service
	AssignIPs(assignmentRequest AssignmentRequest, _ *struct{}) (err error)

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
	UpdateServiceState(state ServiceState, unused *int) error

	// Get the services instances for a given service
	GetServiceStates(serviceId string, states *[]*ServiceState) error

	// Get logs for the given app
	GetServiceLogs(serviceId string, logs *string) error

	// Get logs for the given app
	GetServiceStateLogs(request ServiceStateRequest, logs *string) error

	// Get all running services
	GetRunningServices(request EntityRequest, runningServices *[]*RunningService) error

	// Get the services instances for a given service
	GetRunningServicesForHost(hostId string, runningServices *[]*RunningService) error

	// Get the service instances for a given service
	GetRunningServicesForService(serviceId string, runningServices *[]*RunningService) error

	//---------------------------------------------------------------------------
	// ResourcePool CRUD

	// Add a new service pool to serviced
	AddResourcePool(pool ResourcePool, poolId *string) error

	// Update a service pool definition
	UpdateResourcePool(pool ResourcePool, _ *struct{}) error

	// Remove a service pool
	RemoveResourcePool(poolId string, unused *int) error

	//TODO does this belong here
	// Get a list of all the resource pools
	//GetResourcePool(poolId string, pool *ResourcePool) error

	// Get a list of all the resource pools
	GetResourcePools(request EntityRequest, pool *map[string]*ResourcePool) error

	// Get of a list of hosts that are in the given resource pool
	GetHostsForResourcePool(poolId string, poolHosts *[]*PoolHost) error

	// Get a list of IPInfo objects (includes a pool's IP address' information... both virtual and static)
	RetrievePoolIPs(poolId string, IPsInfo *[]IPInfo) error

	//---------------------------------------------------------------------------
	// ServiceTemplate CRUD

	// Deploy an application template in to production
	DeployTemplate(request ServiceTemplateDeploymentRequest, tenantId *string) error

	// Add a new service Template
	AddServiceTemplate(serviceTemplate ServiceTemplate, templateId *string) error

	// Update a new service Template
	UpdateServiceTemplate(serviceTemplate ServiceTemplate, unused *int) error

	// Update a new service Template
	RemoveServiceTemplate(serviceTemplateId string, unused *int) error

	// Get a list of ServiceTemplates
	GetServiceTemplates(unused int, serviceTemplates *map[string]*ServiceTemplate) error

	//---------------------------------------------------------------------------
	// Service CRUD

	// Start an interative shell in a service container
	StartShell(service Service, unused *int) error

	// Execute a service container shell command
	ExecuteShell(service Service, command *string) error

	// Show available commands
	ShowCommands(service Service, unused *int) error

	// Rollback DFS and service image
	Rollback(snapshotId string, unused *int) error

	// Commit DFS and service image
	Commit(containerId string, label *string) error

	// Performs a local snapshot from the host
	LocalSnapshot(serviceId string, label *string) error

	// Snapshots DFS and service image
	Snapshot(serviceId string, label *string) error

	// Delete a snapshot
	DeleteSnapshot(snapshotId string, unused *int) error

	// List available snapshots
	Snapshots(serviceId string, snapshotIds *[]string) error

	// Delete snapshots for a given service
	DeleteSnapshots(serviceId string, unused *int) error

	// Download a file from a container
	Get(service Service, file *string) error

	// Upload file(s) to a container
	Send(service Service, files *[]string) error

	// Get the DFS volume
	GetVolume(serviceId string, theVolume *volume.Volume) error

	//GetSystemUser retrieves the credentials for the system_user account
	GetSystemUser(unused int, user *User) error

	//ValidateCredentials verifies if the passed in user has the correct username and password
	ValidateCredentials(user User, result *bool) error

	// Waits for the DFS to be ready
	ReadyDFS(bool, *int) error

	// Write a tgz file containing all templates and services
	Backup(backupDirectory string, backupFilePath *string) error

	// Restore templates and services from a tgz file (inverse of Backup)
	Restore(backupFilePath string, unused *int) error
}
