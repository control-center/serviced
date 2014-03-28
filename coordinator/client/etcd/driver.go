// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
package etcd

import (
	"encoding/json"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/zenoss/serviced/coordinator/client"
)

// Driver is the
type Driver struct{}

type DSN struct {
	Servers []string
	Timeout time.Duration
}

func (dsn DSN) String() string {
	bytes, err := json.Marshal(dsn)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// Assert that the Ectd driver meets the Driver interface
var _ client.Driver = &Driver{}

func init() {
	client.RegisterDriver("etcd", &Driver{})
}

func ParseDSN(dsn string) (dsnVal DSN, err error) {
	err = json.Unmarshal([]byte(dsn), &dsnVal)
	return dsnVal, err
}

func (driver *Driver) GetConnection(dsn string) (client.Connection, error) {

	dsnVal, err := ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	client := etcd.NewClient(dsnVal.Servers)
	client.SetConsistency("STRONG_CONSISTENCY")

	connection := &Connection{
		client:  client,
		servers: dsnVal.Servers,
		timeout: dsnVal.Timeout,
		onClose: nil,
	}
	return connection, nil
}
