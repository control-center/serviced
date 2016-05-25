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

package service

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

const (
	zkServiceLock = "/locks/services"
)

// ServiceLock initializes a new lock for services
func ServiceLock(conn client.Connection) (client.Lock, error) {
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

// ServiceLock sets a lock on a group of services
type ServiceLockNode struct {
	PoolID    string
	ServiceID string
}

// Path returns the path to the service
func (l ServiceLockNode) Path() string {
	return poolpath(l.PoolID, servicepath(l.ServiceID))
}

// LockServices locks a group of services
func LockServices(conn client.Connection, svcs []ServiceLockNode) error {
	tx := conn.NewTransaction()
	for _, svc := range svcs {
		node := ServiceNode{Service: &service.Service{}}
		if err := conn.Get(svc.Path(), &node); err != nil {
			return err
		}
		node.Locked = true
		tx.Set(svc.Path(), &node)
	}
	return tx.Commit()
}

// UnlockServices unlocks a group of services
func UnlockServices(conn client.Connection, svcs []ServiceLockNode) error {
	tx := conn.NewTransaction()
	for _, svc := range svcs {
		node := ServiceNode{Service: &service.Service{}}
		if err := conn.Get(svc.Path(), &node); err != nil && err != client.ErrNoNode {
			glog.Infof("Could not get service %s in pool %s: %s", svc.ServiceID, svc.PoolID)
			return err
		}
		if node.Locked {
			node.Locked = false
			tx.Set(svc.Path(), &node)
		}
	}
	return tx.Commit()
}
