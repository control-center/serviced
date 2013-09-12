// Describes a network port, it's application and it's context. The context
// describes where the port is supplied by the service (innate), supplied by the
// parent service (inherited), or supplied by a peer/sibling service (aquired).
//
// application listening on port 5500 on all interfaces over tcp: tcp://0.0.0.0:5500
// application listening on unix socket at a given path: unix://localhost/var/lib/mysql/mysql.sock
// application listening on port 500 on all interfaces over udp: udp://0.0.0.0:5000
// application connecting to a service on host example on port 162 on over udp: udp://example.com:5000

package service

import (
	"net/url"
	"time"
)

const (
	ACQUIRED iota = -1
	INNATE
	INHERITED
)

type Connection struct {
	Uri          url.URL // Uri describing the connection: eg tcp://0.0.0.0:5500
	Application  string  // Application which this connection is applicable to.
	Relationship string  // Describes whether the connection is innate, inherited or acquired
}

type ConfigurationFile struct {
	Path           string        // Path to the file, inside the container
	UpdateInterval time.Duration // 0 value is update when the service dependancies change
}

type ServiceTemplate struct {
	Name         string            // Name of service
	Description  string            // Description of the service
	Connections  *[]Connection     // Connections for this service
	Command      *string           // A script template used to launch the service given the service context
	ImageUrl     url.URL           // Image URL; this points to a resource that describes
	MinInstances int               // Minimum number of instances to launch, 0 when using service containers
	MaxInstances int               // Maximum number of instances to launch, -1 for unbounded
	Singleton    bool              // Does
	SubServices  []ServiceTemplate // Subservice templates
	Context      string            // A JSON object that can contain arbitraty values for use in Command templates
}

type Service struct {
	Id           string
	Name         string
        Description  string
	Connections  *[]Connection
	ImageUrl      url.URL
	PoolId       string
	DesiredState int
}
