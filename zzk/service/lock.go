// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

const (
	zkServiceLock = "/locks/services"
)

type ServiceLockListener struct {
	conn client.Connection
}

func NewServiceLockListener() *ServiceLockListener { return &ServiceLockListener{} }

func (l *ServiceLockListener) SetConnection(conn client.Connection) {
	// requires a root-based connection
	l.conn = conn
}

func (l *ServiceLockListener) GetPath(nodes ...string) string {
	return poolpath(nodes...)
}

func (l *ServiceLockListener) Ready() error { return nil }

func (l *ServiceLockListener) Done() {}

func (l *ServiceLockListener) PostProcess(p map[string]struct{}) {}

func (l *ServiceLockListener) Spawn(shutdown <-chan interface{}, poolID string) {

	lock := l.conn.NewLock(l.GetPath(poolID, zkServiceLock))
	defer lock.Unlock()

	for {
		if err := zzk.Ready(shutdown, l.conn, zkServiceLock); err != nil {
			glog.V(2).Infof("Could not monitor lock: %s", err)
			return
		}

		nodes, lockEvent, err := l.conn.ChildrenW(zkServiceLock)
		if err != nil {
			glog.V(2).Infof("Could not monitor lock: %s", err)
			return
		}

		poolEvent, err := l.conn.GetW(l.GetPath(poolID), &PoolNode{})
		if err != nil {
			glog.V(2).Infof("Could not look up node for pool %s: %s", poolID, err)
			return
		}

		if len(nodes) > 0 {
			glog.V(3).Infof("Engaging service lock for %s", poolID)
			lock.Lock()
		} else {
			glog.V(3).Infof("Disengaging service lock for %s", poolID)
			lock.Unlock()
		}

		select {
		case <-lockEvent:
			// pass
		case e := <-poolEvent:
			if e.Type == client.EventNodeDeleted {
				return
			}
		case <-shutdown:
			return
		}
	}
}

// ServiceLock initializes a new lock for services
func ServiceLock(conn client.Connection) client.Lock {
	return conn.NewLock(zkServiceLock)
}

// IsServiceLocked verifies whether services are locked
func IsServiceLocked(conn client.Connection) (bool, error) {
	locks, err := conn.Children(zkServiceLock)
	if err == client.ErrNoNode {
		return false, nil
	}
	return len(locks) > 0, err
}
