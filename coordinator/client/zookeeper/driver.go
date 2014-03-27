package zk_driver

import (
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/coordinator/client"
)

type Driver struct {
	servers []string
	timeout time.Duration
}

// Assert that the Zookeeper driver meets the Driver interface
var _ client.Driver = &Driver{}

func NewDriver(servers []string, timeout time.Duration) (driver client.Driver, err error) {

	driver = &Driver{
		servers: servers,
		timeout: timeout,
	}
	return driver, nil
}

func (driver *Driver) GetConnection() (client.Connection, error) {
	conn, _, err := zklib.Connect(driver.servers, driver.timeout)
	if err != nil {
		return nil, err
	}
	return &Connection{
		conn:    conn,
		servers: driver.servers,
		timeout: driver.timeout,
	}, nil
}

