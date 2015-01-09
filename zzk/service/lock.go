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

// WatchServiceLock waits for a service lock to be enabled/disabled
func WaitServiceLock(shutdown <-chan interface{}, conn client.Connection, enabled bool) error {
	for {
		// make sure the parent exists
		if err := zzk.Ready(shutdown, conn, zkServiceLock); err != nil {
			return err
		}

		// check if the lock is enabled
		nodes, event, err := conn.ChildrenW(zkServiceLock)
		if err != nil {
			return err
		}

		// check that the states match
		if locked := len(nodes) > 0; locked == enabled {
			return nil
		}

		// wait to reach the desired state or shutdown
		select {
		case <-event:
			// pass
		case <-shutdown:
			return zzk.ErrTimeout
		}
	}
}

// Attempt to acquire a service lock, ensuring that it is held until the finish channel is closed.
// Function blocks until the lock is held by someone (not necessarily by us).
func EnsureServiceLock(cancel, finish <-chan interface{}, conn client.Connection) error {
	go func() {
		lock := ServiceLock(conn)
		lock.Lock()
		<-finish
		lock.Unlock()
	}()
	return WaitServiceLock(cancel, conn, true)
}
