/*
This package defines the common structures used to describe service deployments.
*/
package service

import (
	"net/url"
	"time"
)

// Represents the connection relationship type
type ConnBond int

// Constants that describe the relationship of a connection to a sevice
const (
	CONN_CONNECTS  ConnBond = iota // A connection that is aquired from a sibling service
	CONN_LISTENS                   // A listenting connection that is supplied by its service
	CONN_INHERITED                 // A connection that is aquired from a parent service
)

// A Connection describes a network connection and its relationship to a service.
// The Uri can be used to described a tcp, udp, or unix socket connection that is
// either hosted or consumed by a service.
type Connection struct {
	Uri          url.URL  // Uri describing the connection: eg tcp://0.0.0.0:5500
	Application  string   // Application which this connection is applicable to.
	Relationship ConnBond // Describes whether the service connects to, listens on, or inherits the uri
}

// A ConfigFile represents a file within a container that needs to be written
// given the Pattern. The template is rendered using the context of the
// service and written to the Path within the container.
type ConfigFile struct {
	Path           string        // Path to the file, inside the container
	Pattern        string        // An actual Template used to generate the config file
	UpdateInterval time.Duration // 0 value is update when the service dependancies change
}

// A Template represents the basic constructs of service. It describes how a service is
// started, what image it uses, how many instances can run, it's sub services, and a Context
// that can be used as input to render the Command as a template; it is also available during
// the evaluation of the ConfiguationFile instances.
type Template struct {
	Name         string             // Name of service
	Description  string             // Description of the service
	Connections  *[]Connection      // Connections for this service
	ConfigFiles  *[]ConfigFile      // ConfigFiles that must written inside the service container
	Env          *map[string]string // A map of environment variables that the service needs. The values are evalated as templates.
	Command      *string            // A script template used to launch the service given the service context
	ImageUrl     url.URL            // Image URL; this points to a resource that describes
	MinInstances int                // Minimum number of instances to launch, 0 when using service containers
	MaxInstances int                // Maximum number of instances to launch, -1 for unbounded
	Singleton    bool               // Should this service only exist once per resource pool?
	SubServices  []Template         // Subservice templates
	Context      string             // A JSON object that can contain arbitraty values for use in Command templates
}

// A service Instance represents an instance of a service Template
type Instance struct {
	Key         string     // Unique id of a service
	PoolKey     string     // Pool which this service belongs in
	Template               // Inherit all the attributes of the serviceTemplate
	SubServices []Instance // Change the type of SubServices
}
