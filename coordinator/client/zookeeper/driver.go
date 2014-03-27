package zk_driver

import (
	"encoding/json"
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/coordinator/client"
)

type Driver struct {}

type DSN struct {
	Servers []string
	Timeout time.Duration
}

// Assert that the Zookeeper driver meets the Driver interface
var _ client.Driver = &Driver{}

func ParseDSN(dsn string) (val DSN, err error) {
	err = json.Unmarshal([]byte(dsn), &val)
	return val, err
}

func init() {
	client.RegisterDriver("zookeeper", &Driver{})
}

func (driver *Driver) GetConnection(dsn string) (client.Connection, error) {

	dsnVal, err := ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	conn, _, err := zklib.Connect(dsnVal.Servers, dsnVal.Timeout)
	if err != nil {
		return nil, err
	}
	return &Connection{
		conn:    conn,
		servers: dsnVal.Servers,
		timeout: dsnVal.Timeout,
	}, nil
}

