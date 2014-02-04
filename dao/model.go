package dao

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/shell"
	"os"
	"os/exec"
	"path/filepath"

	"fmt"
	"strconv"
	"syscall"
	"time"
)

type HostIpAndPort struct {
	HostIp   string
	HostPort string
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

// HostIPs contains information about IPs on a host.
type HostIPs struct {
	Id     string
	HostId string
	PoolId string
	IPs    []HostIPResource
}

//AssignedPort is used to track Ports that have been assigned to a Service. Only exists in the context of a HostIPResource
type AssignedPort struct {
	Port      int
	ServiceId string
}

//HostIPResource contains information about a specific IP on a host. Also track spcecific ports that have been assigned
//to Services
type HostIPResource struct {
	State         string //State of the IP [valid|deleted]. deleted if IP is no longer on a Host
	IPAddress     string
	InterfaceName string
	Ports         []AssignedPort
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
	ConfigFiles     map[string]ConfigFile
	Instances       int
	ImageId         string
	PoolId          string
	DesiredState    int
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
}

// An endpoint that a Service exposes.
type ServiceEndpoint struct {
	Purpose             string
	Protocol            string
	PortNumber          uint16
	Application         string
	ApplicationTemplate string
	AddressConfig       AddressResourceConfig `TODO get json for nil`
	VHost               []string              // VHost is used to request named vhost for this endpoint. Should be the name of a
	// subdomain, i.e "myapplication"  not "myapplication.host.com"
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
}

type ConfigFile struct {
	Filename    string // complete path of file
	Owner       string // owner of file within the container, root:root or 0:0 for root owned file
	Permissions int    // permission of file, 0660 (rw owner, rw group, not world rw)
	Content     string // content of config file
}

type ServiceDefinition struct {
	Name        string                 // Name of the defined service
	Command     string                 // Command which runs the service
	Description string                 // Description of the service
	Tags        []string               // Searchable service tags
	ImageId     string                 // Docker image hosting the service
	Instances   MinMax                 // Constraints on the number of instances
	Launch      string                 // Must be "AUTO", the default, or "MANUAL"
	ConfigFiles map[string]ConfigFile  // Config file templates
	Context     map[string]interface{} // Context information for the service
	Endpoints   []ServiceEndpoint      // Comms endpoints used by the service
	Services    []ServiceDefinition    // Supporting subservices
	Tasks       []Task                 // Scheduled tasks for celery to find
	LogFilters  map[string]string      // map of log filter name to log filter definitions
	Volumes     []Volume               // list of volumes to bind into containers
	LogConfigs  []LogConfig
}

// AddressResourceConfigByPort implements sort.Interface for []AddressResourceConfig based on the Port field
type AddressResourceConfigByPort []AddressResourceConfig

func (a AddressResourceConfigByPort) Len() int           { return len(a) }
func (a AddressResourceConfigByPort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a AddressResourceConfigByPort) Less(i, j int) bool { return a[i].Port < a[j].Port }

//TCP UDP: Constants for AddressResourceConfig port protocols
const (
	TCP = "tcp"
	UDP = "udp"
)

//AddressResourceConfig defines an external facing port for a service definition
type AddressResourceConfig struct {
	Port     int
	Protocol string
}

// Defines commands to be run in an object's container
type Process struct {
	ServiceId string              // The service id of the container to start
	IsTTY     bool                // Describes the type of connection needed
	Envv      []string            // Environment variables
	Command   string              // Command to run
	Error     error               `json:"-"`
	Stdin     chan string         `json:"-"`
	Stdout    chan string         `json:"-"`
	Stderr    chan string         `json:"-"`
	Exited    chan bool           `json:"-"`
	Signal    chan syscall.Signal `json:"-"`
}

func NewProcess(serviceId, command string, envv []string, istty bool) *Process {
	return &Process{
		ServiceId: serviceId,
		IsTTY:     istty,
		Envv:      envv,
		Command:   command,
		Stdin:     make(chan string),
		Signal:    make(chan syscall.Signal),
		Exited:    make(chan bool),
	}
}

type LogConfig struct {
	Path    string   // The location on the container's filesystem of the log, can be a directory
	Type    string   // Arbitrary string that identifies the "types" of logs that come from this source. This will be
	Filters []string // A list of filters that must be contained in either the LogFilters or a parent's LogFilter
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

// Retrieve service endpoint imports
func (s *Service) GetServiceImports() (endpoints []ServiceEndpoint) {
	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if ep.Purpose == "import" {
				endpoints = append(endpoints, ep)
			}
		}
	}
	return
}

// This is wrong! I feel guilty. Avoid circular imports.
func execPath() (string, string, error) {
	path, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return "", "", err
	}
	return filepath.Dir(path), filepath.Base(path), nil
}

// Starts a container shell
func (s *Service) Exec(p *Process) error {
	var runner shell.Runner

	// Bind mount on /serviced
	dir, bin, err := execPath()
	if err != nil {
		return err
	}
	servicedVolume := fmt.Sprintf("%s:/serviced", dir)

	// Bind mount the pwd
	dir, err = os.Getwd()
	pwdVolume := fmt.Sprintf("%s:/mnt/pwd", dir)

	// Get the shell command
	var shellCmd string
	if p.Command != "" {
		shellCmd = p.Command
	} else {
		shellCmd = "su -"
	}

	// Get the proxy Command
	proxyCmd := []string{fmt.Sprintf("/serviced/%s", bin), "-logtostderr=false", "proxy", "-logstash=false", "-autorestart=false", s.Id, shellCmd}
	// Get the docker start command
	docker, err := exec.LookPath("docker")
	if err != nil {
		return err
	}
	argv := []string{"run", "-rm", "-v", servicedVolume, "-v", pwdVolume}
	argv = append(argv, p.Envv...)

	if p.IsTTY {
		argv = append(argv, "-i", "-t")
	}

	argv = append(argv, s.ImageId)
	argv = append(argv, proxyCmd...)

	runner, err = shell.CreateCommand(docker, argv)

	if err != nil {
		return err
	}

	// @see http://dave.cheney.net/tag/golang-3
	p.Stdout = runner.StdoutPipe()
	p.Stderr = runner.StderrPipe()

	go p.send(runner)
	return nil
}

func (p *Process) send(r shell.Runner) {
	exited := r.ExitedPipe()
	go r.Reader(8192)

	defer func() {
		close(p.Stdin)
		close(p.Exited)
		close(p.Signal)
	}()

	for {
		select {
		case i := <-p.Stdin:
			r.Write([]byte(i))
		case s := <-p.Signal:
			r.Signal(s)
		case m := <-exited:
			p.Error = r.Error()
			p.Exited <- m
			return
		}
	}
}

// Retrieve service container port, 0 failure
func (ss *ServiceState) GetHostPort(protocol, application string, port uint16) uint16 {
	for _, ep := range ss.Endpoints {
		if ep.PortNumber == port && ep.Application == application && ep.Protocol == protocol && ep.Purpose == "export" {
			if protocol == "Tcp" {
				protocol = "tcp"
			} else if protocol == "Udp" {
				protocol = "udp"
			}

			portS := fmt.Sprintf("%d/%s", port, protocol)
			external := ss.PortMapping[portS]
			if len(external) == 0 {
				glog.Errorf("Found match for %s, but no portmapping is available", application)
				break
			}
			glog.V(1).Infof("Found %v for %s", external, portS)
			extPort, err := strconv.Atoi(external[0].HostPort)
			if err != nil {
				glog.Errorf("Unable to convert to integer: %v", err)
				break
			}
			return uint16(extPort)
		}
	}

	return 0
}
