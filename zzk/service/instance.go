// Copyright 2015 The Serviced Authors.
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
	"errors"
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	ss "github.com/control-center/serviced/domain/servicestate"
	"github.com/zenoss/glog"
)

const (
	zkInstanceLock = "/locks/instances"
	zkStateLock    = "/locks/states" // per-service lock
)

var ErrLockNotFound = errors.New("lock not found")

// newStateLock sets up the zk state lock for a given service id
func newStateLock(conn client.Connection, serviceID string) client.Lock {
	return conn.NewLock(path.Join(zkStateLock, serviceID))
}

// rmStateLock removes a zk state lock parent
func rmStateLock(conn client.Connection, serviceID string) error {
	return conn.Delete(path.Join(zkStateLock, serviceID))
}

// newInstanceLock sets up a new zk instance lock for a given service state id
func newInstanceLock(conn client.Connection, stateID string) client.Lock {
	return conn.NewLock(path.Join(zkInstanceLock, stateID))
}

// rmInstanceLock removes a zk instance lock parent
func rmInstanceLock(conn client.Connection, stateID string) error {
	return conn.Delete(path.Join(zkInstanceLock, stateID))
}

// addInstance creates a new service state and host instance
func addInstance(conn client.Connection, state ss.ServiceState) error {
	glog.V(2).Infof("Adding instance %+v", state)
	// check the object
	if err := state.ValidEntity(); err != nil {
		glog.Errorf("Could not validate service state %+v: %s", state, err)
		return err
	}

	// CC-1050: we need to trigger the scheduler in case we only have a
	// partial create.
	svclock := newStateLock(conn, state.ServiceID)
	if err := svclock.Lock(); err != nil {
		glog.Errorf("Could not set lock on service %s: %s", state.ServiceID, err)
		return err
	}
	defer svclock.Unlock()

	lock := newInstanceLock(conn, state.ID)
	if err := lock.Lock(); err != nil {
		glog.Errorf("Could not set lock for service instance %s for service %s on host %s: %s", state.ID, state.ServiceID, state.HostID, err)
		return err
	}
	glog.V(2).Infof("Acquired lock for instance %s", state.ID)
	defer lock.Unlock()

	var err error
	defer func() {
		if err != nil {
			conn.Delete(hostpath(state.HostID, state.ID))
			conn.Delete(servicepath(state.ServiceID, state.ID))
			rmInstanceLock(conn, state.ID)
		}
	}()

	// Create node on the service
	spath := servicepath(state.ServiceID, state.ID)
	snode := &ServiceStateNode{ServiceState: &state}
	if err = conn.Create(spath, snode); err != nil {
		glog.Errorf("Could not create service state %s for service %s: %s", state.ID, state.ServiceID, err)
		return err
	} else if err = conn.Set(spath, snode); err != nil {
		glog.Errorf("Could not set service state %s for node %+v: %s", state.ID, snode, err)
		return err
	}

	// Create node on the host
	hpath := hostpath(state.HostID, state.ID)
	hnode := NewHostState(&state)
	glog.V(2).Infof("Host node: %+v", hnode)
	if err = conn.Create(hpath, hnode); err != nil {
		glog.Errorf("Could not create host state %s for host %s: %s", state.ID, state.HostID, err)
		return err
	} else if err = conn.Set(hpath, hnode); err != nil {
		glog.Errorf("Could not set host state %s for node %+v: %s", state.ID, hnode, err)
		return err
	}

	glog.V(2).Infof("Releasing lock for instance %s", state.ID)
	return nil
}

// removeInstance removes the service state and host instances
func removeInstance(conn client.Connection, serviceID, hostID, stateID string) error {
	glog.V(2).Infof("Removing instance %s", stateID)

	// CC-1050: we need to trigger the scheduler in case we only have a
	// partial delete.
	svclock := newStateLock(conn, serviceID)
	if err := svclock.Lock(); err != nil {
		glog.Errorf("Could not set lock on service %s: %s", serviceID, err)
		return err
	}
	defer svclock.Unlock()

	lock := newInstanceLock(conn, stateID)
	if err := lock.Lock(); err != nil {
		glog.Errorf("Could not set lock for service instance %s for service %s on host %s: %s", stateID, serviceID, hostID, err)
		return err
	}
	defer lock.Unlock()
	defer rmInstanceLock(conn, stateID)
	glog.V(2).Infof("Acquired lock for instance %s", stateID)
	// Remove the node on the service
	spath := servicepath(serviceID, stateID)
	if err := conn.Delete(spath); err != nil {
		if err != client.ErrNoNode {
			glog.Errorf("Could not delete service state node %s for service %s on host %s: %s", stateID, serviceID, hostID, err)
			return err
		}
	}
	// Remove the node on the host
	hpath := hostpath(hostID, stateID)
	if err := conn.Delete(hpath); err != nil {
		if err != client.ErrNoNode {
			glog.Errorf("Could not delete host state node %s for host %s: %s", stateID, hostID, err)
			return err
		}
	}
	glog.V(2).Infof("Releasing lock for instance %s", stateID)
	return nil
}

// updateInstance updates the service state and host instances
func updateInstance(conn client.Connection, hostID, stateID string, mutate func(*HostState, *ss.ServiceState)) error {
	glog.V(2).Infof("Updating instance %s", stateID)
	// do not lock if parent lock does not exist
	if exists, err := conn.Exists(path.Join(zkInstanceLock, stateID)); err != nil && err != client.ErrNoNode {
		glog.Errorf("Could not check for lock on instance %s: %s", stateID, err)
		return err
	} else if !exists {
		glog.Errorf("Lock not found for instance %s", stateID)
		return ErrLockNotFound
	}

	lock := newInstanceLock(conn, stateID)
	if err := lock.Lock(); err != nil {
		glog.Errorf("Could not set lock for service instance %s on host %s: %s", stateID, hostID, err)
		return err
	}
	defer lock.Unlock()
	glog.V(2).Infof("Acquired lock for instance %s", stateID)

	hpath := hostpath(hostID, stateID)
	var hsdata HostState
	if err := conn.Get(hpath, &hsdata); err != nil {
		glog.Errorf("Could not get instance %s for host %s: %s", stateID, hostID, err)
		return err
	}
	serviceID := hsdata.ServiceID
	spath := servicepath(serviceID, stateID)
	var ssnode ServiceStateNode
	if err := conn.Get(spath, &ssnode); err != nil {
		glog.Errorf("Could not get instance %s for service %s: %s", stateID, serviceID, err)
		return err
	}

	mutate(&hsdata, ssnode.ServiceState)

	if err := conn.Set(hpath, &hsdata); err != nil {
		glog.Errorf("Could not update instance %s for host %s: %s", stateID, hostID, err)
		return err
	}
	if err := conn.Set(spath, &ssnode); err != nil {
		glog.Errorf("Could not update instance %s for service %s: %s", stateID, serviceID, err)
		return err
	}
	glog.V(2).Infof("Releasing lock for instance %s", stateID)
	return nil
}

// pauseInstance pauses a running host state instance
func pauseInstance(conn client.Connection, hostID, stateID string) error {
	return updateInstance(conn, hostID, stateID, func(hsdata *HostState, _ *ss.ServiceState) {
		if hsdata.DesiredState == int(service.SVCRun) {
			glog.V(2).Infof("Pausing service instance %s via host %s", stateID, hostID)
			hsdata.DesiredState = int(service.SVCPause)
		}
	})
}

// resumeInstance resumes a paused host state instance
func resumeInstance(conn client.Connection, hostID, stateID string) error {
	return updateInstance(conn, hostID, stateID, func(hsdata *HostState, _ *ss.ServiceState) {
		if hsdata.DesiredState == int(service.SVCPause) {
			glog.V(2).Infof("Resuming service instance %s via host %s", stateID, hostID)
			hsdata.DesiredState = int(service.SVCRun)
		}
	})
}

// UpdateServiceState does a full update of a service state
func UpdateServiceState(conn client.Connection, state *ss.ServiceState) error {
	if err := state.ValidEntity(); err != nil {
		glog.Errorf("Could not validate service state %+v: %s", state, err)
		return err
	}
	return updateInstance(conn, state.HostID, state.ID, func(_ *HostState, ssdata *ss.ServiceState) {
		*ssdata = *state
	})
}

// StopServiceInstance stops a host state instance
func StopServiceInstance(conn client.Connection, hostID, stateID string) error {
	// verify that the host is active
	var isActive bool
	hostIDs, err := GetActiveHosts(conn)
	if err != nil {
		glog.Warningf("Could not verify if host %s is active: %s", hostID, err)
		isActive = false
	} else {
		for _, hid := range hostIDs {
			if isActive = hid == hostID; isActive {
				break
			}
		}
	}
	if isActive {
		// try to stop the instance nicely
		return updateInstance(conn, hostID, stateID, func(hsdata *HostState, _ *ss.ServiceState) {
			glog.V(2).Infof("Stopping service instance via %s host %s", stateID, hostID)
			hsdata.DesiredState = int(service.SVCStop)
		})
	} else {
		// if the host isn't active, then remove the instance
		var hs HostState
		if err := conn.Get(hostpath(hostID, stateID), &hs); err != nil {
			glog.Errorf("Could not look up host instance %s on host %s: %s", stateID, hostID, err)
			return err
		}
		return removeInstance(conn, hs.ServiceID, hs.HostID, hs.ServiceStateID)
	}
}
