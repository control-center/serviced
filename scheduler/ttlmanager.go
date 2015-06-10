// Copyright 2015 The Serviced Authors.
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

package scheduler

import (
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

type ttlrunner struct {
	ttl     utils.TTL
	minWait time.Duration
	maxWait time.Duration
}

// TTLManager manages all of the running ttls
type TTLManager struct {
	ttlchan chan ttlrunner
	cancel  chan struct{}
}

// NewTTLManager instantiates a new ttl manager
func NewTTLManager() *TTLManager {
	return &TTLManager{make(chan ttlrunner), make(chan struct{})}
}

// Leader returns the leader node for the manager running on the specified host
func (m *TTLManager) Leader(conn client.Connection, host *host.Host) client.Leader {
	return conn.NewLeader("/resource/ttl", &zkservice.HostNode{Host: host})
}

// Run starts the ttl manager
func (m *TTLManager) Run(cancel <-chan interface{}) error {
	glog.Infof("Starting TTL Manager")
	for {
		select {
		case t := <-m.ttlchan:
			go utils.RunTTL(t.ttl, cancel, t.minWait, t.maxWait)
		case <-cancel:
			close(m.cancel)
			glog.Infof("TTL Manager stopped")
			return nil
		}
	}
}

// StartTTL starts a TTL
func (m *TTLManager) StartTTL(ttl utils.TTL, min, max time.Duration) {
	go func() {
		select {
		case m.ttlchan <- ttlrunner{ttl, min, max}:
		case <-m.cancel:
		}
	}()
}