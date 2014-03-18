
package client

import (
	"time"

	"github.com/coreos/go-etcd/etcd"
)


type EtcdDriver struct {
	client *etcd.Client
	servers []string
	timeout time.Duration
}


func NewEtcdDriver(servers []string, timeout time.Duration) (driver *EtcdDriver, err error) {

	client := etcd.NewClient(servers)
	client.SetConsistency("STRONG_CONSISTENCY")

	driver = &EtcdDriver{
		client: client,
		servers: servers,
		timeout: timeout,
	}
	return driver, nil
}

func (etcd *EtcdDriver) Create(path string, data []byte) error {
	_, err := etcd.client.Create(path, string(data), 1000000)
	return err
}

