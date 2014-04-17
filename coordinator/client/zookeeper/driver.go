package zookeeper

import (
	"encoding/json"
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/coordinator/client"
)

// Driver implements a Zookeeper based client.Driver interface
type Driver struct{}

// Assert that the Zookeeper driver meets the Driver interface
var _ client.Driver = &Driver{}

func init() {
	client.RegisterDriver("zookeeper", &Driver{})
}

// DSN is a Zookeeper specific struct used for connections. It can be
// serialized.
type DSN struct {
	Servers []string
	Timeout time.Duration
}

// NewDSN returns a new DSN object from servers and timeout.
func NewDSN(servers []string, timeout time.Duration) DSN {
	dsn := DSN{
		Servers: servers,
		Timeout: timeout,
	}
	if dsn.Servers == nil || len(dsn.Servers) == 0 {
		dsn.Servers = []string{"127.0.0.1:2181"}
	}
	return dsn
}

// String creates a parsable (JSON) string represenation of this DSN.
func (dsn DSN) String() string {
	bytes, err := json.Marshal(dsn)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// ParseDSN decodes a string (JSON) represnation of a DSN object.
func ParseDSN(dsn string) (val DSN, err error) {
	err = json.Unmarshal([]byte(dsn), &val)
	return val, err
}

// GetConnection returns a Zookeeper connection given the dsn. The caller is
// responsible for closing the returned connection.
func (driver *Driver) GetConnection(dsn, basePath string) (client.Connection, error) {

	dsnVal, err := ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	conn, event, err := zklib.Connect(dsnVal.Servers, dsnVal.Timeout)
	if err != nil {
		return nil, err
	}
	<-event // wait for connected event, or anything really
	return &Connection{
		basePath: basePath,
		conn:     conn,
		servers:  dsnVal.Servers,
		timeout:  dsnVal.Timeout,
	}, nil
}
