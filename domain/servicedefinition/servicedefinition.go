// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package servicedefinition

import (
	"github.com/zenoss/serviced/domain"

	"errors"
	"strings"
	"time"
)

//ServiceDefinition is the definition of a service hierarchy.
type ServiceDefinition struct {
	Name              string                 // Name of the defined service
	Command           string                 // Command which runs the service
	Description       string                 // Description of the service
	Tags              []string               // Searchable service tags
	ImageID           string                 // Docker image hosting the service
	Instances         domain.MinMax          // Constraints on the number of instances
	Launch            string                 // Must be "AUTO", the default, or "MANUAL"
	HostPolicy        HostPolicy             // Policy for starting up instances
	Hostname          string                 // Optional hostname which should be set on run
	Privileged        bool                   // Whether to run the container with extended privileges
	ConfigFiles       map[string]ConfigFile  // Config file templates
	Context           map[string]interface{} // Context information for the service
	Endpoints         []EndpointDefinition   // Comms endpoints used by the service
	Services          []ServiceDefinition    // Supporting subservices
	Tasks             []Task                 // Scheduled tasks for celery to find
	LogFilters        map[string]string      // map of log filter name to log filter definitions
	Volumes           []Volume               // list of volumes to bind into containers
	LogConfigs        []LogConfig
	Snapshot          SnapshotCommands              // Snapshot quiesce info for the service: Pause/Resume bash commands
	RAMCommitment     uint64                        // expected RAM commitment to use for scheduling
	CPUCommitment     uint64                        // expected CPU commitment (#cores) to use for scheduling
	Runs              map[string]string             // Map of commands that can be executed with 'serviced run ...'
	Actions           map[string]string             // Map of commands that can be executed with 'serviced action ...'
	HealthChecks      map[string]domain.HealthCheck // HealthChecks for a service.
	Prereqs           []domain.Prereq               // Optional list of scripts that must be successfully run before kicking off the service command.
	MonitoringProfile domain.MonitorProfile         // An optional list of queryable metrics, graphs, and thresholds
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
	VirtualAddress      string // An address by which an imported endpoint may be accessed within the container, e.g. "mysqlhost:1234"
	Application         string
	ApplicationTemplate string
	AddressConfig       AddressResourceConfig
	VHosts              []string // VHost is used to request named vhost for this endpoint. Should be the name of a
	// subdomain, i.e "myapplication"  not "myapplication.host.com"
}

// Task A scheduled task
type Task struct {
	Name          string
	Schedule      string
	Command       string
	LastRunAt     time.Time
	TotalRunCount int
}

// Volume import defines a file system directory underneath an export directory
type Volume struct {
	Owner         string //Resource Path Owner
	Permission    string //Resource Path permissions, eg what you pass to chmod
	ResourcePath  string //Resource Pool Path, shared across all hosts in a resource pool
	ContainerPath string //Container bind-mount path
}

// ConfigFile config file for a service
type ConfigFile struct {
	Filename    string // complete path of file
	Owner       string // owner of file within the container, root:root or 0:0 for root owned file, what you would pass to chown
	Permissions string // permission of file, eg 0664, what you would pass to chmod
	Content     string // content of config file
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
	//DEFAULT policy for scheduling a service instance
	DEFAULT HostPolicy = ""
	//LeastCommitted run on host w/ least committed memory
	LeastCommitted = "LEAST_COMMITTED"
	//PreferSeparate attempt to schedule instances of a service on separate hosts
	PreferSeparate = "PREFER_SEPARATE"
	//RequireSeparate schedule instances of a service on separate hosts
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
