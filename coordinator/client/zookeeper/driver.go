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
	"math/rand"
	"time"

	zklib "github.com/control-center/go-zookeeper/zk"
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

// zkBackoff controls the exponential backoff used when connection attempts to all zookepers fail
type zkBackoff struct {
	initialDelay time.Duration	// the initial delay
	maxDelay     time.Duration	// The maximum delay
	delay        time.Duration	// the current delay
}

// GetDelay returns the amount of delay that should be used for the current connection attempt.
// It will return a randomized value of initialDelay on the first call, and will increase the delay
// randomly on each subsequent call up to maxDelay. The initial delay and each subsequent delay
// are randomized to avoid a scenario where multiple instances on the same host all start trying
// to reconnection. In scenarios like those, we don't want all instances reconnecting in lock-step
// with each other.
func (backoff *zkBackoff) GetDelay() time.Duration {
	defer func() {
		factor := 2.0
		jitter := 6.0

		backoff.delay = time.Duration(float64(backoff.delay) * factor)
		backoff.delay += time.Duration(rand.Float64() * jitter * float64(time.Second))
		if backoff.delay > backoff.maxDelay {
			backoff.delay = backoff.maxDelay
		}
	}()

	if backoff.delay == 0 {
		backoff.Reset()
	}
	return backoff.delay
}

// Reset resets the backoff delay to some random value that is btwn 80-120% of the initialDelay.
//     We want to randomize the initial delay so in cases where many instances simultaneously
//     lose all ZK connections, they will not all start trying to reconnect at the same time.
func (backoff *zkBackoff) Reset() {
	start := backoff.initialDelay.Seconds()
	minStart := 0.8 * start
	maxStart := 1.2 * start
	start = start + rand.NormFloat64()
	if start < minStart {
		start = minStart
	} else if start > maxStart {
		start = maxStart
	}
	backoff.delay = time.Duration(start * float64(time.Second))

	// never exceeed maxDelay
	if backoff.delay > backoff.maxDelay {
		backoff.delay = backoff.maxDelay
	}
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
		zklib.WithBackoff(&zkBackoff{
			initialDelay: dsnVal.ReconnectStartDelay,
			maxDelay:     dsnVal.ReconnectMaxDelay,
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
				plog.WithField("session", e).Debug("zk connection has session")
			} else {
				plog.WithField("session", e).Debug("waiting for zk connection to have session")
			}
		}
	}
	go func() {
		for {
			select {
			case _, ok := <-event:
				if !ok {
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
