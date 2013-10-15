package dao

import "time"

type ServiceTemplateWrapper struct {
	Name            string // Name of top level service
	Description     string // Description
	Data            string // JSON encoded template definition
	ApiVersion      int    // Version of the ServiceTemplate API this expects
	TemplateVersion int    // Version of the template
}

// A Service Template used for
type ServiceTemplate struct {
	Id               string                // Unique id of service
	Name             string                // Name of service
	Startup          string                // Startup command
	Description      string                // Meaningful description of service
	MinInstances     int                   // mininum number of instances to run
	MaxInstances     int                   // maximum number of instances to run
	ImageId          string                // Docker image id
	ServiceEndpoints []ServiceEndpoint     // Ports that this service uses
	SubServices      []ServiceTemplate     // Child services
}

// A request to deploy a service template
type ServiceTemplateDeploymentRequest struct {
	PoolId   string          // Pool Id to deploy service into
	Template ServiceTemplate // ServiceTemplate to deploy
}

// An association between a host and a pool.
type PoolHost struct {
	HostId string
	PoolId string
}

// A collection of computing resources with optional quotas.
type ResourcePool struct {
	Id          string // Unique identifier for resource pool, eg "default"
	ParentId    string // The pool id of the parent pool, if this pool is embeded in another pool. An empty string means it is not embeded.
	Priority    int    // relative priority of resource pools, used for CPU priority
	CoreLimit   int    // Number of cores on the host available to serviced
	MemoryLimit uint64 // A quota on the amount (bytes) of RAM in the pool, 0 = unlimited
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// A new ResourcePool
func NewResourcePool(id string) (*ResourcePool, error) {
  pool := &ResourcePool{}
	pool.Id = id
	return pool, nil
}

func (pool *ResourcePool) MakeSubpool(id string) *ResourcePool {
	subpool := *pool
	subpool.Id = id
	subpool.ParentId = pool.Id
	subpool.Priority = 0
	return &subpool
}

// A host that runs the control plane agent.
type Host struct {
	Id             string // Unique identifier, default to hostid
	Name           string // A label for the host, eg hostname, role
	PoolId         string // Pool that the Host belongs to
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

// Desired states of services.
const (
	SVC_RUN     = 1
	SVC_STOP    = 0
	SVN_RESTART = -1
)

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
	ParentServiceId string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Endpoints       *[]ServiceEndpoint
}

// An endpoint that a Service exposes.
type ServiceEndpoint struct {
	Protocol    string
	PortNumber  uint16
	Application string
	Purpose     string
}

// An instantiation of a Service.
type ServiceState struct {
	Id          string
	ServiceId   string
	HostId      string
	DockerId    string
	PrivateIp   string
	Scheduled   time.Time
	Terminated  time.Time
	Started     time.Time
	PortMapping map[string]map[string]string
}

// Create a new Service.
func NewService() (s *Service, err error) {
	s = &Service{}
	s.Id, err = newUuid()
	return s, err
}

// A new service instance (ServiceState)
func (s *Service) NewServiceState(hostId string) (serviceState *ServiceState, err error) {
	serviceState = &ServiceState{}
	serviceState.Id, err = newUuid()
	if err == nil {
	  serviceState.ServiceId = s.Id
	  serviceState.HostId = hostId
	  serviceState.Scheduled = time.Now()
	}
	return serviceState, err
}

