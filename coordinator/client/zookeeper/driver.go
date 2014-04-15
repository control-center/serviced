package zk_driver

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

// DNS is a Zookeeper specific struct used for connections. It can be
// serialized.
type DSN struct {
	Servers []string
	Timeout time.Duration
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

	conn, _, err := zklib.Connect(dsnVal.Servers, dsnVal.Timeout)
	if err != nil {
		return nil, err
	}
	return &Connection{
		basePath: basePath,
		conn:    conn,
		servers: dsnVal.Servers,
		timeout: dsnVal.Timeout,
	}, nil
}
