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
	TCP ProtocolType = "tcp"
	UDP ProtocolType = "udp"
)

// A user defined string that describes an exposed application endpoint.
type ApplicationType string

// An endpoint that a Service exposes.
type ServiceEndpoint struct {
	Protocol    ProtocolType
	PortNumber  uint16
	Application ApplicationType
}

// A Service that can run in serviced.
type Service struct {
	Id           string
	Name         string
	Startup      string
	Description  string
	Instances    int
	ImageId      string
	PoolId       string
	DesiredState int
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
	s.Id, err = newUuid()
	if err != nil {
		return s, err
	}
	return s, nil
}

// A host that runs the control plane agent.
type Host struct {
	Id             string
	Name           string
	IpAddr         string
	Cores          int
	Memory         uint64
	PrivateNetwork string
}

// Create a new host.
func NewHost() *Host {
	host := &Host{}
	return host
}

// A collection of computing resources with optional quotas.
type ResourcePool struct {
	Id          string
	Name        string
	CoreLimit   int
	MemoryLimit uint64
	Priority    int
}

// A new ResourcePool
func NewResourcePool() (pool *ResourcePool, err error) {
	pool = new(ResourcePool)
	pool.Id, err = newUuid()
	return pool, err
}

// An instantiation of a Service.
type ServiceState struct {
	Id         string
	ServiceId  string
	HostId     string
	Scheduled  time.Time
	Terminated time.Time
	Started    time.Time
	DockerId   string
	PrivateIp  string
}

// The state of a container as reported by Docker.
type ContainerState struct {
	ID      string
	Created time.Time
	Path    string
	Args    []string
	Config  struct {
		Hostname     string
		User         string
		Memory       uint64
		MemorySwap   uint64
		CpuShares    int
		AttachStdin  bool
		AttachStdout bool
		AttachStderr bool
		PortSpecs    *string
		Tty          bool
		OpenStdin    bool
		StdinOnce    bool
		Env          *string
		Cmd          []string
		Dns          []string
		Image        string
		Volumes      map[string]string
		VolumesFrom  string
		Entrypoint   []string
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
		PortMapping struct {
			Tcp map[string]string
			Udp map[string]string
		}
	}
	SysInitPath    string
	ResolvConfPath string
	Volumes        map[string]string
	VolumesRW      map[string]string
}

// A new service instance (ServiceState)
func NewServiceState(serviceId string, hostId string) (serviceState *ServiceState, err error) {
	serviceState = &ServiceState{}
	serviceState.Id, err = newUuid()
	if err != nil {
		return serviceState, err
	}
	serviceState.ServiceId = serviceId
	serviceState.HostId = hostId
	serviceState.Scheduled = time.Now()
	return serviceState, err
}

// An association between a host and a pool.
type PoolHost struct {
	HostId string
	PoolId string
}

// A request for a given application type exposed by the given service.
type ServiceEndpointRequest struct {
	ServiceId   string
	Application ApplicationType
}

// An exposed service endpoint
type ApplicationEndpoint struct {
	ServiceId string
	HostIp    string
	Port      uint16
	Protocol  ProtocolType
}

// The API for a service proxy.
type LoadBalancer interface {
	GetServiceEndpoints(serviceEndpointRequest ServiceEndpointRequest, endpoints *[]ApplicationEndpoint) error
}

// The ControlPlane interface is the API for a serviced master.
type ControlPlane interface {
	GetHosts(request EntityRequest, replyHosts *map[string]*Host) error
	AddHost(host Host, unused *int) error
	UpdateHost(host Host, ununsed *int) error
	RemoveHost(hostId string, unused *int) error

	GetServices(request EntityRequest, replyServices *[]*Service) error
	AddService(service Service, unused *int) error
	UpdateService(service Service, unused *int) error
	RemoveService(serviceId string, unused *int) error
	GetServicesForHost(hostId string, services *[]*Service) error

	GetServiceStates(serviceId string, states *[]*ServiceState) error
	StartService(serviceId string, hostId *string) error
	RestartService(serviceId string, unused *int) error
	StopService(serviceId string, unused *int) error
	UpdateServiceState(state ServiceState, unused *int) error

	GetResourcePools(request EntityRequest, pool *map[string]*ResourcePool) error
	AddResourcePool(pool ResourcePool, unused *int) error
	UpdateResourcePool(pool ResourcePool, unused *int) error
	RemoveResourcePool(poolId string, unused *int) error
	GetHostsForResourcePool(poolId string, poolHosts *[]*PoolHost) error
	AddHostToResourcePool(poolHost PoolHost, unused *int) error
	RemoveHostFromResourcePool(poolHost PoolHost, unused *int) error
}

// The Agent interface is the API for a serviced agent.
type Agent interface {
	GetInfo(unused int, host *Host) error
}
