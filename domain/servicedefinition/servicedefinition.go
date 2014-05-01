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
	Name          string                 // Name of the defined service
	Command       string                 // Command which runs the service
	Description   string                 // Description of the service
	Tags          []string               // Searchable service tags
	ImageID       string                 // Docker image hosting the service
	Instances     domain.MinMax          // Constraints on the number of instances
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
	Snapshot      SnapshotCommands              // Snapshot quiesce info for the service: Pause/Resume bash commands
	RAMCommitment uint64                        // expected RAM commitment to use for scheduling
	Runs          map[string]string             // Map of commands that can be executed with 'serviced run ...'
	Actions       map[string]string             // Map of commands that can be executed with 'serviced action ...'
	HealthChecks  map[string]domain.HealthCheck // HealthChecks for a service.
}

// SnapshotCommands commands to be called during and after a snapshot
type SnapshotCommands struct {
	Pause  string // bash command to pause the volume  (quiesce)
	Resume string // bash command to resume the volume (unquiesce)
}

// ServiceEndpoint An endpoint that a Service exposes.
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

	AddressAssignment AddressAssignment //TODO: doesn't belong in this package. addressAssignment holds the assignment when Service is started

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

//TODO: these methods don't belong here on ServiceEndpoint. Service should have a different type with these methods

//AddressAssignment is used to track Ports that have been assigned to a Service. Only exists in the context of a HostIPResource
type AddressAssignment struct {
	ID             string //Generated id
	AssignmentType string //Static or Virtual
	HostID         string //Host id if type is Static
	PoolID         string //Pool id if type is Virtual
	IPAddr         string //Used to associate to resource in Pool or Host
	Port           uint16 //Actual assigned port
	ServiceID      string //Service using this assignment
	EndpointName   string //Endpoint in the service using the assignment
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
	if se.AddressAssignment.ID == "" {
		return nil
	}
	//return reference to copy
	result := se.AddressAssignment
	return &result
}
