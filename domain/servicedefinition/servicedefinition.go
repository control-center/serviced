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

package servicedefinition

import (
	"encoding/json"
	"errors"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/utils"
)

// initialize the package logger
var plog = logging.PackageLogger()

// ServiceDefinition is the definition of a node in a service hierarchy.
type ServiceDefinition struct {
	Name        string // Name of the defined service
	Title       string // Title is a label used when describing this service in the context of a service tree
	Version     string // Version of the defined service
	Command     string // Command which runs the service
	RunAs       string // Run command as user
	Description string // Description of the service

	Environment []string // Environment variables to be injected, of the form NAME="value"
	Tags        []string // Searchable service tags

	ImageID       string         // Docker image hosting the service
	Instances     domain.MinMax  // Constraints on the number of instances
	ChangeOptions []ChangeOption // Control options for what happens when a running service is changed
	Launch        string         // Must be "AUTO", the default, or "MANUAL"
	HostPolicy    HostPolicy     // Policy for starting up instances
	Hostname      string         // Optional hostname which should be set on run
	Privileged    bool           // Whether to run the container with extended privileges

	ConfigFiles map[string]ConfigFile  // Config file templates
	Context     map[string]interface{} // Context information for the service
	Endpoints   []EndpointDefinition   // Comms endpoints used by the service
	Services    []ServiceDefinition    // Supporting subservices
	LogFilters  map[string]string      // map of log filter name to log filter definitions

	Volumes    []Volume // list of volumes to bind into containers
	LogConfigs []LogConfig

	Snapshot      SnapshotCommands  // Snapshot quiesce info for the service: Pause/Resume bash commands
	RAMCommitment utils.EngNotation // expected RAM commitment to use for scheduling
	RAMThreshold  uint              // RAM Threshold
	CPUCommitment uint64            // expected CPU commitment (#cores) to use for scheduling
	DisableShell  bool              // disables shell commands on the service

	Runs         map[string]string             // FIXME: This field is deprecated. Remove when possible.
	Commands     map[string]domain.Command     // Map of commands that can be executed with 'serviced run ...'
	Actions      map[string]string             // Map of commands that can be executed with 'serviced action ...'
	HealthChecks map[string]health.HealthCheck // HealthChecks for a service.

	// Prereqs is a list of scripts that must run successfully before running the command in the Startup field.
	Prereqs []domain.Prereq

	MonitoringProfile domain.MonitorProfile // An optional list of queryable metrics, graphs, and thresholds
	MemoryLimit       float64
	CPUShares         int64
	OomKillDisable    bool  // Whether to disable OOM Killer for the container or not
	OomScoreAdj       int64 // Tune containers OOM preferences (-1000 to 1000)

	PIDFile string // An optional path or command to generate a path for a PID file to which signals are relayed.

	StartLevel uint // Services start in the order implied by this field (low to high) and stopped in reverse order

	EmergencyShutdownLevel uint // Services are stopped in this order during an emergency (low to high).
}

// SnapshotCommands commands to be called during and after a snapshot
type SnapshotCommands struct {
	Pause  string // bash command to pause the volume  (quiesce)
	Resume string // bash command to resume the volume (unquiesce)
}

// EndpointDefinition An endpoint that a Service exposes.
type EndpointDefinition struct {
	Name                string // Human readable name of the endpoint. Unique per service definition
	Purpose             string
	Protocol            string
	PortNumber          uint16
	PortTemplate        string // A template which, if specified, is used to calculate the port number
	VirtualAddress      string // An address by which an imported endpoint may be accessed within the container, e.g. "mysqlhost:1234"
	Application         string
	ApplicationTemplate string
	AddressConfig       AddressResourceConfig

	// VHost is used to request named vhost for this endpoint. Should be the name of a
	// subdomain, i.e "myapplication"  not "myapplication.host.com"
	VHosts []string

	VHostList []VHost // VHost is used to request named vhost(s) for this endpoint.
	PortList  []Port
}

// VHost is the configuration for an application endpoint that wants an http VHost endpoint provided by Control Center
type VHost struct {
	Name    string // name of the vhost subdomain subdomain, i.e "myapplication"  not "myapplication.host.com
	Enabled bool   // whether the vhost should be enabled or disabled.
}

// Port is the configuration for an application endpoint port.
type Port struct {
	PortAddr string // which port number to use for this endpoint
	Enabled  bool   // whether the port should be enabled or disabled.
	UseTLS   bool   // Does this port endpoint use tls.
	Protocol string // What protocol (if any) does the endpoind use.
}

// Volume import defines a file system directory underneath an export directory
type Volume struct {
	Owner              string // Resource Path Owner
	Permission         string // Resource Path permissions, eg what you pass to chmod
	ResourcePath       string // Resource Pool Path, shared across all hosts in a resource pool
	ContainerPath      string // Container bind-mount path
	Type               string // Path use, i.e. "dfs" or "tmp"
	InitContainerPath  string // Path to initialize the volume from at creation time, optional
	ExcludeFromBackups bool   // Whether to exclude this volume from backups
}

// ConfigFile config file for a service
type ConfigFile struct {
	Filename    string // complete path of file
	Owner       string // owner of file within the container, root:root or 0:0 for root owned file, what you would pass to chown
	Permissions string // permission of file, eg 0664, what you would pass to chmod
	Content     string // content of config file
}

// GetConfigFileType returns the ConfigFiles type
func GetConfigFileType() string {
	return "serviceconfigurationfile"
}

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
	IsAudit bool     // Whether to send log entries to /var/log/serviced/application-audit.log or not for each LogConfig Type
}

// LogTag  no clue what this is. Maybe someone actually reads this
type LogTag struct {
	Name  string
	Value string
}

// HostPolicy represents the optional policy used to determine which hosts on
// which to run instances of a service. Default is to run on the available
// host with the most uncommitted RAM.
type HostPolicy string

const (
	// DEFAULT policy for scheduling a service instance
	DEFAULT HostPolicy = ""
	// LeastCommitted run on host w/ least committed resources
	LeastCommitted = "LEAST_COMMITTED"
	// Balance is a synonym for LeastCommitted
	Balance = "balance"
	// Pack runs instance on eligible host with most committed resources
	Pack = "pack"
	// PreferSeparate attempt to schedule instances of a service on separate hosts
	PreferSeparate = "PREFER_SEPARATE"
	// RequireSeparate schedule instances of a service on separate hosts
	RequireSeparate = "REQUIRE_SEPARATE"
)

// UnmarshalText implements the encoding/TextUnmarshaler interface
func (p *HostPolicy) UnmarshalText(b []byte) error {
	s := strings.Trim(string(b), `"`)
	switch s {
	case LeastCommitted, PreferSeparate, RequireSeparate:
		*p = HostPolicy(s)
	case "":
		*p = DEFAULT
	default:
		return errors.New("Invalid HostPolicy: " + s)
	}
	return nil
}

// ChangeOption is the policy for what happens in the scheduler Sync when the running services change
type ChangeOption string

const (
	// DefaultChangeOption is the default change option (none).
	DefaultChangeOption = ChangeOption("")

	// RestartAllOnInstanceChanged option restarts all instances when the instance count changes.
	// If the number of running instances doesn't match the requested number of instances in the service
	// definition, all instances will be restarted.  Note that this can happen due to an instance going down
	// or by the service def being modified.
	RestartAllOnInstanceChanged = ChangeOption("restartAllOnInstanceChanged")

	// RestartAllOnInstanceZeroDown option will stop and restart all running instances if there is no instance 0.
	// If the running instances doesn't contain instance 0, all services will be shut down so that when they come
	// back up we'll get a new instance 0.
	RestartAllOnInstanceZeroDown = ChangeOption("restartAllOnInstanceZeroDown")
)

// UnmarshalText implements the encoding/TextUnmarshaler interface
func (co *ChangeOption) UnmarshalText(b []byte) error {
	s := strings.ToLower(strings.Trim(string(b), `"`))
	switch s {
	case strings.ToLower(string(RestartAllOnInstanceChanged)), strings.ToLower(string(RestartAllOnInstanceZeroDown)):
		*co = ChangeOption(s)
	case "":
		*co = DefaultChangeOption
	default:
		return errors.New("Invalid ChangeOption: " + s)
	}
	return nil
}

// ChangeOptions is a list of ChangeOption objects.
type ChangeOptions []ChangeOption

// Contains returns true if the ChangeOptions object contains the ChangeOption object.
func (options ChangeOptions) Contains(co ChangeOption) bool {
	for _, option := range options {
		if strings.ToLower(string(co)) == strings.ToLower(string(option)) {
			return true
		}
	}
	return false
}

type serviceDefinition ServiceDefinition

// UnmarshalJSON loads a JSON byte stream into the ServiceDefinition object.
func (s *ServiceDefinition) UnmarshalJSON(b []byte) error {
	sd := serviceDefinition{}
	if err := json.Unmarshal(b, &sd); err == nil {
		*s = ServiceDefinition(sd)
	} else {
		return err
	}
	if len(s.Commands) > 0 {
		s.Runs = nil
		return nil
	}
	if len(s.Runs) > 0 {
		s.Commands = make(map[string]domain.Command)
		for k, v := range s.Runs {
			s.Commands[k] = domain.Command{
				Command:         v,
				CommitOnSuccess: true,
			}
		}
		s.Runs = nil
	}
	return nil
}

// private for dealing with unmarshal recursion
type endpointDefinition EndpointDefinition

// UnmarshalJSON implements the encoding/json/Unmarshaler interface used to convert deprecated vhosts list to VHostList
func (e *EndpointDefinition) UnmarshalJSON(b []byte) error {
	epd := endpointDefinition{}
	if err := json.Unmarshal(b, &epd); err == nil {
		*e = EndpointDefinition(epd)
	} else {
		return err
	}
	if len(e.VHostList) > 0 {
		//VHostList is defined, keep it and unset deprecated field if set
		e.VHosts = nil
		return nil
	}
	if len(e.VHosts) > 0 {
		// no VHostsList but vhosts is defined. Convert to VHostsList
		if log.GetLevel() == log.DebugLevel {
			plog.WithField("vhosts", e.VHosts).Warning("The field named VHosts in EndpointDefinition is deprecated, see VHostList")
		}
		for _, vhost := range e.VHosts {
			e.VHostList = append(e.VHostList, VHost{Name: vhost, Enabled: true})
		}
		plog.WithFields(log.Fields{
			"vhostlist": e.VHostList,
			"vhosts":    e.VHosts,
		}).Debug("VHostList created from VHosts")
		e.VHosts = nil
	}
	return nil
}

func (s ServiceDefinition) String() string {
	return s.Name
}

//BuildFromPath given a path will create a ServiceDefintion
func BuildFromPath(path string) (*ServiceDefinition, error) {
	sd, err := getServiceDefinition(path)
	if err != nil {
		return nil, err
	}
	return sd, sd.ValidEntity()
}

// GetType return the ServiceDefinition's type
// It returns the type as a string
func GetType() string {
	return "servicedefinition"
}

// GetType returns the ServiceDefinition instance's type
// It returns the type as a string
func (s *ServiceDefinition) GetType() string {
	return GetType()
}

// GetID return a ServiceDefinition instance's ID
// It returns the ID as a string
func (s *ServiceDefinition) GetID() string {
	return s.Name
}
