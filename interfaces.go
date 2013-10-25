/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

// Serviced is a PaaS runtime based on docker. The serviced package exposes the
// interfaces for the key parts of this runtime.
package serviced

import (
	"time"
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
type EntityRequest struct{}

// Network protocol type.
type ProtocolType string

const (
	AUTO   string = "auto"
	MANUAL string = "manual"
	TCP    string = "tcp"
	UDP    string = "udp"
)

// A user defined string that describes an exposed application endpoint.
type ApplicationType string

// An endpoint that a Service exposes.
type ServiceEndpoint struct {
	Protocol    string
	PortNumber  uint16
	Application string
	Purpose     string
}

// A Service that can run in serviced.
type Service struct {
	Id              string
	Name            string
	Startup         string
	Description     string
	Instances       int
	ImageId         string
	PoolId          string
	DesiredState    int
	Launch          string
	Endpoints       *[]ServiceEndpoint
	ParentServiceId string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ServiceTemplateWrapper struct {
	Id              string // Primary key
	Name            string // Name of top level service
	Description     string // Description
	Data            string // JSON encoded template definition
	ApiVersion      int    // Version of the ServiceTemplate API this expects
	TemplateVersion int    // Version of the template
}

type MinMax struct {
	Min int
	Max int
}

type ServiceDefinition struct {
	Name        string              // Name of the defined service
	Command     string              // Command which runs the service
	Description string              // Description of the service
	ImageId     string              // Docker image hosting the service
	Instances   MinMax              // Constraints on the number of instances
	Launch      string              // Must be "AUTO", the default, or "MANUAL"
	Endpoints   []ServiceEndpoint   // Comms endpoints used by the service
	Services    []ServiceDefinition // Supporting subservices
}

type ServiceDeployment struct {
	Id         string    // Primary key
	TemplateId string    // id of template being deployed
	ServiceId  string    // id of service created by deployment
	DeployedAt time.Time // when the template was deployed
}

// A Service Template used for
type ServiceTemplate struct {
	Name        string              // Name of service template
	Description string              // Meaningful description of service
	Services    []ServiceDefinition // Child services
}

// A request to deploy a service template
type ServiceTemplateDeploymentRequest struct {
	PoolId     string // Pool Id to deploy service into
	TemplateId string // Id of template to be deployed
}

// This is created by selecting from service_state and joining to service
type RunningService struct {
	Id              string
	ServiceId       string
	HostId          string
	StartedAt       time.Time
	Name            string
	Startup         string
	Description     string
	Instances       int
	ImageId         string
	PoolId          string
	DesiredState    int
	ParentServiceId string
}

// Desired states of services.
const (
	SVC_RUN     = 1
	SVC_STOP    = 0
	SVN_RESTART = -1
)

// Create a new Service.
func NewService() (s *Service, err error) {
	s = &Service{}
	s.Id, err = NewUuid()
	if err != nil {
		return s, err
	}
	return s, nil
}

// A host that runs the control plane agent.
type Host struct {
	Id             string // Unique identifier, default to hostid
	PoolId         string // Pool that the Host belongs to
	Name           string // A label for the host, eg hostname, role
	IpAddr         string // The IP address the host can be reached at from a serviced master
	Cores          int    // Number of cores available to serviced
	Memory         uint64 // Amount of RAM (bytes) available to serviced
	PrivateNetwork string // The private network where containers run, eg 172.16.42.0/24
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Create a new host.
func NewHost() *Host {
	host := &Host{}
	return host
}

// A collection of computing resources with optional quotas.
type ResourcePool struct {
	Id          string // Unique identifier for resource pool, eg "default"
	ParentId    string // The pool id of the parent pool, if this pool is embeded in another pool. An empty string means it is not embeded.
	CoreLimit   int    // Number of cores on the host available to serviced
	MemoryLimit uint64 // A quota on the amount (bytes) of RAM in the pool, 0 = unlimited
	Priority    int    // relative priority of resource pools, used for CPU priority
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// A new ResourcePool
func NewResourcePool(id string) (pool *ResourcePool, err error) {
	pool = new(ResourcePool)
	pool.Id = id
	return pool, err
}

func (pool *ResourcePool) MakeSubpool(id string) *ResourcePool {
	subpool := *pool
	subpool.Id = id
	subpool.ParentId = pool.Id
	subpool.Priority = 0
	return &subpool
}

// An instantiation of a Service.
type ServiceState struct {
	Id          string
	ServiceId   string
	HostId      string
	Scheduled   time.Time
	Terminated  time.Time
	Started     time.Time
	DockerId    string
	PrivateIp   string
	PortMapping map[string]map[string]string
}

// The state of a container as reported by Docker.
type ContainerState struct {
	ID      string
	Created time.Time
	Path    string
	Args    []string
	Config  struct {
		Hostname        string
		User            string
		Memory          uint64
		MemorySwap      uint64
		CpuShares       int
		AttachStdin     bool
		AttachStdout    bool
		AttachStderr    bool
		PortSpecs       []string
		Tty             bool
		OpenStdin       bool
		StdinOnce       bool
		Env             []string
		Cmd             []string
		Dns             []string
		Image           string
		Volumes         map[string]struct{}
		VolumesFrom     string
		WorkingDir      string
		Entrypoint      []string
		NetworkDisabled bool
		Privileged      bool
	}
	State struct {
		Running   bool
		Pid       int
		ExitCode  int
		StartedAt string
		Ghost     bool
	}
	Image           string
	NetworkSettings struct {
		IPAddress   string
		IPPrefixLen int
		Gateway     string
		Bridge      string
		PortMapping map[string]map[string]string
	}
	SysInitPath    string
	ResolvConfPath string
	Volumes        map[string]string
	VolumesRW      map[string]bool
}

// A new service instance (ServiceState)
func (s *Service) NewServiceState(hostId string) (serviceState *ServiceState, err error) {
	serviceState = &ServiceState{}
	serviceState.Id, err = NewUuid()
	if err != nil {
		return serviceState, err
	}
	serviceState.ServiceId = s.Id
	serviceState.HostId = hostId
	serviceState.Scheduled = time.Now()
	return serviceState, err
}

// An association between a host and a pool.
type PoolHost struct {
	HostId string
	PoolId string
}

// An exposed service endpoint
type ApplicationEndpoint struct {
	ServiceId     string
	ContainerPort uint16
	HostPort      uint16
	HostIp        string
	ContainerIp   string
	Protocol      string
}

// The API for a service proxy.
type LoadBalancer interface {
	GetServiceEndpoints(serviceId string, endpoints *map[string][]*ApplicationEndpoint) error
}

// The ControlPlane interface is the API for a serviced master.
type ControlPlane interface {

	LoadBalancer

	// Get a list of registered hosts
	GetHosts(request EntityRequest, replyHosts *map[string]*Host) error

	// Register a host with serviced
	AddHost(host Host, unused *int) error

	// Update Host information for a registered host
	UpdateHost(host Host, ununsed *int) error

	// Remove a Host from serviced
	RemoveHost(hostId string, unused *int) error

	// Get a list of services from serviced
	GetServices(request EntityRequest, replyServices *[]*Service) error

	// Add a new service
	AddService(service Service, unused *int) error

	// Update an existing service
	UpdateService(service Service, unused *int) error

	// Remove a service definition
	RemoveService(serviceId string, unused *int) error

	// Get all the services that need to be running on the given host
	GetServicesForHost(hostId string, services *[]*Service) error

	// Get logs for the given app
	GetServiceLogs(serviceId string, logs *string) error

	// Get logs for the given app
	GetServiceStateLogs(serviceStateId string, logs *string) error

	// Get the services instances for a given service
	GetRunningServicesForHost(hostId string, runningServices *[]*RunningService) error

	// Get all running services
	GetRunningServices(request EntityRequest, runningServices *[]*RunningService) error

	// Get the services instances for a given service
	GetServiceStates(serviceId string, states *[]*ServiceState) error

	// Schedule the given service to start
	StartService(serviceId string, hostId *string) error

	// Schedule the given service to restart
	RestartService(serviceId string, unused *int) error

	// Schedule the given service to stop
	StopService(serviceId string, unused *int) error

	// Update the service state
	UpdateServiceState(state ServiceState, unused *int) error

	// Get a list of all the resource pools
	GetResourcePools(request EntityRequest, pool *map[string]*ResourcePool) error

	// Add a new service pool to serviced
	AddResourcePool(pool ResourcePool, unused *int) error

	// Update a service pool definition
	UpdateResourcePool(pool ResourcePool, unused *int) error

	// Remove a service pool
	RemoveResourcePool(poolId string, unused *int) error

	// Get of a list of hosts that are in the given resource pool
	GetHostsForResourcePool(poolId string, poolHosts *[]*PoolHost) error

	// Deploy an application template in to production
	DeployTemplate(request ServiceTemplateDeploymentRequest, unused *int) error

	// Get a list of ServiceTemplates
	GetServiceTemplates(unused int, serviceTemplates *map[string]*ServiceTemplate) error

	// Add a new service Template
	AddServiceTemplate(serviceTemplate ServiceTemplate, unused *int) error

	// Update a new service Template
	UpdateServiceTemplate(serviceTemplate ServiceTemplate, unused *int) error

	// Update a new service Template
	RemoveServiceTemplate(serviceTemplateId string, unused *int) error
}

// The Agent interface is the API for a serviced agent.
type Agent interface {
	GetInfo(unused int, host *Host) error
}
