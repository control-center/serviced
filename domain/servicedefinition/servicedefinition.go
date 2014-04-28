package servicedefinition

import (
	"github.com/zenoss/serviced/utils"

	"time"
)

type ServiceDefinition struct {
	Name          string                 // Name of the defined service
	Command       string                 // Command which runs the service
	Description   string                 // Description of the service
	Tags          []string               // Searchable service tags
	ImageId       string                 // Docker image hosting the service
	Instances     MinMax                 // Constraints on the number of instances
	Launch        string                 // Must be "AUTO", the default, or "MANUAL"
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
}

// A scheduled task
type Task struct {
	Name          string
	Schedule      string
	Command       string
	LastRunAt     time.Time
	TotalRunCount int
}

// volume import defines a file system directory underneath an export directory
type Volume struct {
	Owner         string //Resource Path Owner
	Permission    string //Resource Path permissions, eg what you pass to chmod
	ResourcePath  string //Resource Pool Path, shared across all hosts in a resource pool
	ContainerPath string //Container bind-mount path
}

type ConfigFile struct {
	Filename    string // complete path of file
	Owner       string // owner of file within the container, root:root or 0:0 for root owned file, what you would pass to chown
	Permissions string // permission of file, eg 0664, what you would pass to chmod
	Content     string // content of config file
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
