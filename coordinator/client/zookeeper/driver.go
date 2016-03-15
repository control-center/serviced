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
	"time"

	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/zenoss/glog"
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

	// wait for session event
	connected := false
	for !connected {
		select {
		case e := <-event:
			if e.State == zklib.StateHasSession {
				connected = true
				glog.V(1).Infof("zk connection has session %v", e)
			} else {
				glog.V(1).Infof("waiting for zk connection to have session %v", e)
			}
		}
	}
	go func() {
		for {
			select {
			case e, ok := <-event:
				glog.V(1).Infof("zk event %s", e)
				if !ok {
					glog.V(1).Infoln("zk eventchannel closed")
					return
				}
			}
		}
	}()
	return &Connection{
		basePath: basePath,
		conn:     conn,
	}, nil
}
