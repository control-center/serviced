package dao

import (
	"github.com/zenoss/serviced/domain/servicedefinition"

	"fmt"
	"github.com/zenoss/glog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type HostIpAndPort struct {
	HostIp   string
	HostPort string
}

type User struct {
	Name     string // the unique identifier for a user
	Password string // no requirements on passwords yet
}

// An association between a host and a pool.
type PoolHost struct {
	HostId string
	PoolId string
	HostIp string
}

//AssignmentRequest is used to couple a serviceId to an IpAddress
type AssignmentRequest struct {
	ServiceId      string
	IpAddress      string
	AutoAssignment bool
}

// Desired states of services.
const (
	SVC_RUN     = 1
	SVC_STOP    = 0
	SVN_RESTART = -1
)

// An exposed service endpoint
type ApplicationEndpoint struct {
	ServiceId     string
	ContainerPort uint16
	HostPort      uint16
	HostIp        string
	ContainerIp   string
	Protocol      string
}

// A Service that can run in serviced.
type Service struct {
	Id              string
	Name            string
	Context         string
	Startup         string
	Description     string
	Tags            []string
	ConfigFiles     map[string]servicedefinition.ConfigFile
	Instances       int
	ImageId         string
	PoolId          string
	DesiredState    int
	HostPolicy      servicedefinition.HostPolicy
	Hostname        string
	Privileged      bool
	Launch          string
	Endpoints       []servicedefinition.ServiceEndpoint
	Tasks           []servicedefinition.Task
	ParentServiceId string
	Volumes         []servicedefinition.Volume
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeploymentId    string
	DisableImage    bool
	LogConfigs      []servicedefinition.LogConfig
	Snapshot        servicedefinition.SnapshotCommands
	Runs            map[string]string
	RAMCommitment   uint64
	Actions         map[string]string
}

//export definition
type ServiceExport struct {
	Protocol    string //tcp or udp
	Application string //application type
	Internal    string //internal port number
	External    string //external port number
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
	PortMapping map[string][]HostIpAndPort // protocol -> container port (internal) -> host port (external)
	Endpoints   []servicedefinition.ServiceEndpoint
	HostIp      string
	InstanceId  int
}

type ServiceDeployment struct {
	Id         string    // Primary key
	TemplateId string    // id of template being deployed
	ServiceId  string    // id of service created by deployment
	DeployedAt time.Time // when the template was deployed
}

// A request to deploy a service template
type ServiceTemplateDeploymentRequest struct {
	PoolId       string // Pool Id to deploy service into
	TemplateId   string // Id of template to be deployed
	DeploymentId string // Unique id of the instance of this template
}

// This is created by selecting from service_state and joining to service
type RunningService struct {
	Id              string
	ServiceId       string
	HostId          string
	DockerId        string
	StartedAt       time.Time
	Name            string
	Startup         string
	Description     string
	Instances       int
	ImageId         string
	PoolId          string
	DesiredState    int
	ParentServiceId string
	InstanceId      int
}

// Create a new Service.
func NewService() (s *Service, err error) {
	s = &Service{}
	s.Id, err = NewUuid()
	return s, err
}

// A new service instance (ServiceState)
func (s *Service) NewServiceState(hostId string) (serviceState *ServiceState, err error) {
	serviceState = &ServiceState{}
	serviceState.Id, err = NewUuid()
	if err == nil {
		serviceState.ServiceId = s.Id
		serviceState.HostId = hostId
		serviceState.Scheduled = time.Now()
		serviceState.Endpoints = s.Endpoints
	}
	return serviceState, err
}

// Does the service have endpoint imports
func (s *Service) HasImports() bool {
	if s.Endpoints == nil {
		return false
	}

	for _, ep := range s.Endpoints {
		if ep.Purpose == "import" {
			return true
		}
	}
	return false
}

// GetServiceImports retrieves service endpoints whose purpose is "import"
func (s *Service) GetServiceImports() []servicedefinition.ServiceEndpoint {
	result := []servicedefinition.ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if ep.Purpose == "import" {
				result = append(result, ep)
			}
		}
	}

	return result
}

// GetServiceExports retrieves service endpoints whose purpose is "export"
func (s *Service) GetServiceExports() []servicedefinition.ServiceEndpoint {
	result := []servicedefinition.ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if ep.Purpose == "export" {
				result = append(result, ep)
			}
		}
	}

	return result
}

// GetServiceVHosts retrieves service endpoints that specify a virtual HostPort
func (s *Service) GetServiceVHosts() []servicedefinition.ServiceEndpoint {
	result := []servicedefinition.ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if len(ep.VHosts) > 0 {
				result = append(result, ep)
			}
		}
	}

	return result
}

// Add a virtual host for given service, this method avoids duplicates vhosts
func (s *Service) AddVirtualHost(application, vhostName string) error {
	if s.Endpoints != nil {

		//find the matching endpoint
		for i := 0; i < len(s.Endpoints); i += 1 {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				_vhostName := strings.ToLower(vhostName)
				vhosts := make([]string, 0)
				for _, vhost := range ep.VHosts {
					if strings.ToLower(vhost) != _vhostName {
						vhosts = append(vhosts, vhost)
					}
				}
				ep.VHosts = append(vhosts, _vhostName)
				return nil
			}
		}
	}

	return fmt.Errorf("Unable to find application %s in service: %s", application, s.Name)
}

// Remove a virtual host for given service
func (s *Service) RemoveVirtualHost(application, vhostName string) error {
	if s.Endpoints != nil {

		//find the matching endpoint
		for i := 0; i < len(s.Endpoints); i += 1 {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				if len(ep.VHosts) == 0 {
					break
				}

				_vhostName := strings.ToLower(vhostName)
				if len(ep.VHosts) == 1 && ep.VHosts[0] == _vhostName {
					return fmt.Errorf("Cannot delete last vhost: %s", _vhostName)
				}

				found := false
				vhosts := make([]string, 0)
				for _, vhost := range ep.VHosts {
					if vhost != _vhostName {
						vhosts = append(vhosts, vhost)
					} else {
						found = true
					}
				}
				//error removing an unknown vhost
				if !found {
					break
				}

				ep.VHosts = vhosts
				return nil
			}
		}
	}

	return fmt.Errorf("Unable to find application %s in service: %s", application, s.Name)
}

// Retrieve service container port info.
func (ss *ServiceState) GetHostEndpointInfo(applicationRegex *regexp.Regexp) (hostPort, containerPort uint16, protocol string, match bool) {
	for _, ep := range ss.Endpoints {
		if ep.Purpose == "export" {
			if applicationRegex.MatchString(ep.Application) {
				portS := fmt.Sprintf("%d/%s", ep.PortNumber, strings.ToLower(ep.Protocol))

				external := ss.PortMapping[portS]
				if len(external) == 0 {
					glog.Errorf("Found match for %s:%s, but no portmapping is available", applicationRegex, portS)
					break
				}

				extPort, err := strconv.ParseUint(external[0].HostPort, 10, 16)
				if err != nil {
					glog.Errorf("Portmap parsing failed for %s:%s %v", applicationRegex, portS, err)
					break
				}
				return uint16(extPort), ep.PortNumber, ep.Protocol, true
			}
		}
	}

	return 0, 0, "", false
}

// An instantiation of a Snapshot request
type SnapshotRequest struct {
	Id            string
	ServiceId     string
	SnapshotLabel string
	SnapshotError string
}

// A new snapshot request instance (SnapshotRequest)
func NewSnapshotRequest(serviceId string, snapshotLabel string) (snapshotRequest *SnapshotRequest, err error) {
	snapshotRequest = &SnapshotRequest{}
	snapshotRequest.Id, err = NewUuid()
	if err == nil {
		snapshotRequest.ServiceId = serviceId
		snapshotRequest.SnapshotLabel = snapshotLabel
		snapshotRequest.SnapshotError = ""
	}
	return snapshotRequest, err
}
