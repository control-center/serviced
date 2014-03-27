package etcd

import (
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/zenoss/serviced/coordinator/client"
)

type EtcdDriver struct {
	servers []string
	timeout time.Duration
}

// Assert that the Ectd driver meets the Driver interface
var _ client.Driver = &EtcdDriver{}

func NewDriver(servers []string, timeout time.Duration) (client.Driver, error) {
	return &EtcdDriver{
		servers: servers,
		timeout: timeout,
	}, nil
}

func (driver *EtcdDriver) GetConnection() (client.Connection, error) {

	client := etcd.NewClient(driver.servers)
	client.SetConsistency("STRONG_CONSISTENCY")

	connection := &EtcdConnection{
		client:  client,
		servers: driver.servers,
		timeout: driver.timeout,
		onClose: nil,
	}
	return connection, nil
}
