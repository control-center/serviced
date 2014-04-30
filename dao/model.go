package dao

import (
	"github.com/zenoss/glog"

	"errors"
	"fmt"
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

type MinMax struct {
	Min int
	Max int
}

type ServiceTemplateWrapper struct {
	Id              string // Primary-key
	Name            string // Name of top level service
	Description     string // Description
	Data            string // JSON encoded template definition
	ApiVersion      int    // Version of the ServiceTemplate API this expects
	TemplateVersion int    // Version of the template
}

// An association between a host and a pool.
type PoolHost struct {
	HostId string
	PoolId string
	HostIp string
}

//AssignedPort is used to track Ports that have been assigned to a Service. Only exists in the context of a HostIPResource
type AddressAssignment struct {
	Id             string //Generated id
	AssignmentType string //Static or Virtual
	HostId         string //Host id if type is Static
	PoolId         string //Pool id if type is Virtual
	IPAddr         string //Used to associate to resource in Pool or Host
	Port           uint16 //Actual assigned port
	ServiceId      string //Service using this assignment
	EndpointName   string //Endpoint in the service using the assignment
}

//AssignmentRequest is used to couple a serviceId to an IpAddress
type AssignmentRequest struct {
	ServiceId      string
	IpAddress      string
	AutoAssignment bool
}

type IPInfo struct {
	Interface string
	IP        string
	Type      string
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

// Snapshot commands
type SnapshotCommands struct {
	Pause  string // bash command to pause the volume  (quiesce)
	Resume string // bash command to resume the volume (unquiesce)
}

// A Service that can run in serviced.
type Service struct {
	Id              string
	Name            string
	Context         string
	Startup         string
	Description     string
	Tags            []string
	ConfigFiles     map[string]ConfigFile
	Instances       int
	ImageId         string
	PoolId          string
	DesiredState    int
	HostPolicy      HostPolicy
	Hostname        string
	Privileged      bool
	Launch          string
	Endpoints       []ServiceEndpoint
	Tasks           []Task
	ParentServiceId string
	Volumes         []Volume
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeploymentId    string
	DisableImage    bool
	LogConfigs      []LogConfig
	Snapshot        SnapshotCommands
	Runs            map[string]string
	RAMCommitment   uint64
	Actions         map[string]string
}

// An endpoint that a Service exposes.
type ServiceEndpoint struct {
	Name                string // Human readable name of the endpoint. Unique per service definition
	Purpose             string
	Protocol            string
	PortNumber          uint16
	Application         string
	ApplicationTemplate string
	AddressConfig       AddressResourceConfig
	VHosts              []string // VHost is used to request named vhost for this endpoint. Should be the name of a
	// subdomain, i.e "myapplication"  not "myapplication.host.com"
	AddressAssignment AddressAssignment //addressAssignment holds the assignment when Service is started
}

// A scheduled task
type Task struct {
	Name          string
	Schedule      string
	Command       string
	LastRunAt     time.Time
	TotalRunCount int
}

//export definition
type ServiceExport struct {
	Protocol    string //tcp or udp
	Application string //application type
	Internal    string //internal port number
	External    string //external port number
}

// volume import defines a file system directory underneath an export directory
type Volume struct {
	Owner         string //Resource Path Owner
	Permission    string //Resource Path permissions, eg what you pass to chmod
	ResourcePath  string //Resource Pool Path, shared across all hosts in a resource pool
	ContainerPath string //Container bind-mount path
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
	Endpoints   []ServiceEndpoint
	HostIp      string
	InstanceId  int
}

type ConfigFile struct {
	Filename    string // complete path of file
	Owner       string // owner of file within the container, root:root or 0:0 for root owned file, what you would pass to chown
	Permissions string // permission of file, eg 0664, what you would pass to chmod
	Content     string // content of config file
}

type ServiceDefinition struct {
	Name          string                 // Name of the defined service
	Command       string                 // Command which runs the service
	Description   string                 // Description of the service
	Tags          []string               // Searchable service tags
	ImageId       string                 // Docker image hosting the service
	Instances     MinMax                 // Constraints on the number of instances
	Launch        string                 // Must be "AUTO", the default, or "MANUAL"
	HostPolicy    HostPolicy             // Policy for starting up instances
	Hostname      string                 // Optional hostname which should be set on run
	Privileged    bool                   // Whether to run the container with extended privileges
	ConfigFiles   map[string]ConfigFile  // Config file templates
	Context       map[string]interface{} // Context information for the service
	Endpoints     []ServiceEndpoint      // Comms endpoints used by the service
	Services      []ServiceDefinition    // Supporting subservices
	Tasks         []Task                 // Scheduled tasks for celery to find
	LogFilters    map[string]string      // map of log filter name to log filter definitions
	Volumes       []Volume               // list of volumes to bind into containers
	LogConfigs    []LogConfig
	Snapshot      SnapshotCommands  // Snapshot quiesce info for the service: Pause/Resume bash commands
	RAMCommitment uint64            // expected RAM commitment to use for scheduling
	Runs          map[string]string // Map of commands that can be executed with 'serviced run ...'
	Actions       map[string]string // Map of commands that can be executed with 'serviced action ...'
}

// AddressResourceConfigByPort implements sort.Interface for []AddressResourceConfig based on the Port field
type AddressResourceConfigByPort []AddressResourceConfig

func (a AddressResourceConfigByPort) Len() int           { return len(a) }
func (a AddressResourceConfigByPort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a AddressResourceConfigByPort) Less(i, j int) bool { return a[i].Port < a[j].Port }

//AddressResourceConfig defines an external facing port for a service definition
type AddressResourceConfig struct {
	Port     uint16
	Protocol string
}

// LogConfig represents the configuration for a logfile for a service.
type LogConfig struct {
	Path    string   // The location on the container's filesystem of the log, can be a directory
	Type    string   // Arbitrary string that identifies the "types" of logs that come from this source. This will be
	Filters []string // A list of filters that must be contained in either the LogFilters or a parent's LogFilter,
	LogTags []LogTag // Key value pair of tags that are sent to logstash for all entries coming out of this logfile
}

type LogTag struct {
	Name  string
	Value string
}

type ServiceDeployment struct {
	Id         string    // Primary key
	TemplateId string    // id of template being deployed
	ServiceId  string    // id of service created by deployment
	DeployedAt time.Time // when the template was deployed
}

// A Service Template used for
type ServiceTemplate struct {
	Id          string                // Unique ID of this service template
	Name        string                // Name of service template
	Description string                // Meaningful description of service
	Services    []ServiceDefinition   // Child services
	ConfigFiles map[string]ConfigFile // Config file templates
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
func (s *Service) GetServiceImports() []ServiceEndpoint {
	result := []ServiceEndpoint{}

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
func (s *Service) GetServiceExports() []ServiceEndpoint {
	result := []ServiceEndpoint{}

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
func (s *Service) GetServiceVHosts() []ServiceEndpoint {
	result := []ServiceEndpoint{}

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

func (se *ServiceEndpoint) SetAssignment(aa *AddressAssignment) error {
	if se.AddressConfig.Port == 0 {
		return errors.New("Cannot assign address to endpoint without AddressResourceConfig")
	}
	se.AddressAssignment = *aa
	return nil
}

//GetAssignment Returns nil if no assignment set
func (se *ServiceEndpoint) GetAssignment() *AddressAssignment {
	if se.AddressAssignment.Id == "" {
		return nil
	}
	//return reference to copy
	result := se.AddressAssignment
	return &result
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

// HostPolicy represents the optional policy used to determine which hosts on
// which to run instances of a service. Default is to run on the available
// host with the most uncommitted RAM.
type HostPolicy string

const (
	DEFAULT          HostPolicy = ""
	LEAST_COMMITTED             = "LEAST_COMMITTED"
	PREFER_SEPARATE             = "PREFER_SEPARATE"
	REQUIRE_SEPARATE            = "REQUIRE_SEPARATE"
)

// UnmarshalText implements the encoding/TextUnmarshaler interface
func (p *HostPolicy) UnmarshalText(b []byte) error {
	s := strings.Trim(string(b), `"`)
	switch s {
	case LEAST_COMMITTED, PREFER_SEPARATE, REQUIRE_SEPARATE:
		*p = HostPolicy(s)
	case "":
		*p = DEFAULT
	default:
		return errors.New("Invalid HostPolicy: " + s)
	}
	return nil
}
