package etcd

import (
	"encoding/json"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/zenoss/serviced/coordinator/client"
)

type EtcdDriver struct {}

type DSN struct {
	Servers []string
	Timeout time.Duration
}

// Assert that the Ectd driver meets the Driver interface
var _ client.Driver = &EtcdDriver{}

func init() {
	client.RegisterDriver("etcd", &EtcdDriver{})
}

func parseDsn(dsn string) (dsnVal DSN, err error) {
	err = json.Unmarshal([]byte(dsn), &dsnVal)
	return dsnVal, err
}

func (driver *EtcdDriver) GetConnection(dsn string) (client.Connection, error) {

	dsnVal, err := parseDsn(dsn)
	if err != nil {
		return nil, err
	}

	client := etcd.NewClient(dsnVal.Servers)
	client.SetConsistency("STRONG_CONSISTENCY")

	connection := &EtcdConnection{
		client:  client,
		servers: dsnVal.Servers,
		timeout: dsnVal.Timeout,
		onClose: nil,
	}
	return connection, nil
}

