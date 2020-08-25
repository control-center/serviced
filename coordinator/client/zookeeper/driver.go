// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zookeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/logging"
)

var (
	plog = logging.PackageLogger() // the standard package logger
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
	SessionTimeout      time.Duration
	ConnectTimeout      time.Duration
	PerHostConnectDelay time.Duration
	ReconnectStartDelay time.Duration
	ReconnectMaxDelay   time.Duration
}

// NewDSN returns a new DSN object from servers and timeout.
func NewDSN(servers []string,
	sessionTimeout time.Duration,
	connectTimeout      time.Duration,
	perHostConnectDelay time.Duration,
	reconnectStartDelay time.Duration,
	reconnectMaxDelay   time.Duration) DSN {
	dsn := DSN{
		Servers: servers,
		SessionTimeout: sessionTimeout,
		ConnectTimeout: connectTimeout,
		PerHostConnectDelay: perHostConnectDelay,
		ReconnectStartDelay: reconnectStartDelay,
		ReconnectMaxDelay: reconnectMaxDelay,
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

	conn, event, err := zklib.Connect(dsnVal.Servers,
		dsnVal.SessionTimeout,
		zklib.WithConnectTimeout(dsnVal.ConnectTimeout),
		zklib.WithReconnectDelay(dsnVal.ReconnectStartDelay),
		zklib.WithPerHostConnectDelay(dsnVal.PerHostConnectDelay),
		zklib.WithBackoff(&client.Backoff{
			InitialDelay: dsnVal.ReconnectStartDelay,
			MaxDelay:     dsnVal.ReconnectMaxDelay,
		}))
	if err != nil {
		return nil, err
	}

	// wait for session event
	connected := false
	for !connected {
		select {
		case e := <-event:
			if e.State == zklib.StateHasSession {
				connected = true
				plog.WithField("event", e).Debug("zk connection has session")
			} else {
				plog.WithField("event", e).Debug("waiting for zk connection to have session")
			}
		}
	}
	go func() {
		for {
			select {
			case e, ok := <-event:
				if !ok {
					plog.WithField("event", e).Debug("zk event channel closed")
					return
				} else {
					plog.WithField("event", e).Debug("zk state change event received")
				}
			}
		}
	}()

	options := config.GetOptions()
	user := options.ZkAclUser
	passwd := options.ZkAclPasswd
	var acl []zklib.ACL

	if user == "" || passwd == "" {

		user = os.Getenv("SERVICED_ZOOKEEPER_ACL_USER")
		passwd = os.Getenv("SERVICED_ZOOKEEPER_ACL_PASSWD")

		if user != "" && passwd != "" {
			acl = zklib.DigestACL(zklib.PermAll, user, passwd)
			if err := conn.AddAuth("digest", []byte(fmt.Sprintf("%s:%s", user, passwd))); err != nil {
				plog.Errorf("AddAuth returned error %+v", err)
				return nil, err
			}
		} else {
			acl = zklib.WorldACL(zklib.PermAll)
		}
	} else {
		acl = zklib.DigestACL(zklib.PermAll, user, passwd)
		if err := conn.AddAuth("digest", []byte(fmt.Sprintf("%s:%s", user, passwd))); err != nil {
			plog.Errorf("AddAuth returned error %+v", err)
			return nil, err
		}
	}

	return &Connection{
		basePath: basePath,
		conn:     conn,
		acl:      acl,
	}, nil
}
