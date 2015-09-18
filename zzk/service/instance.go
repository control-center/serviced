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
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	ss "github.com/control-center/serviced/domain/servicestate"
	"github.com/zenoss/glog"
)

// addInstance creates a new service state and host instance
func addInstance(conn client.Connection, state ss.ServiceState) error {
	glog.V(2).Infof("Adding instance %+v", state)
	// check the object
	if err := state.ValidEntity(); err != nil {
		glog.Errorf("Could not validate service state %+v: %s", state, err)
		return err
	}

	spath := servicepath(state.ServiceID, state.ID)
	snode := &ServiceStateNode{ServiceState: &state}
	hpath := hostpath(state.HostID, state.ID)
	hnode := NewHostState(&state)

	t := conn.NewTransaction()
	t.Create(spath, snode)
	t.Create(hpath, hnode)
	if err := t.Commit(); err != nil {
		glog.Errorf("Could not create service state nodes %s for service %s on host %s: %s", state.ID, state.ServiceID, state.HostID, err)
		return err
	}

	return nil
}

// removeInstance removes the service state and host instances
func removeInstance(conn client.Connection, serviceID, hostID, stateID string) error {
	glog.V(2).Infof("Removing instance %s", stateID)

	spath := servicepath(serviceID, stateID)
	hpath := hostpath(hostID, stateID)

	t := conn.NewTransaction()

	spresent, err := conn.Exists(spath)
	if err != nil {
		glog.Errorf("Error checking the existence of service state node %s for service %s on host %s: %s", stateID, serviceID, hostID, err)
		return err
	}

	if spresent {
		t.Delete(spath)
	}

	hpresent, err := conn.Exists(hpath)
	if err != nil {
		glog.Errorf("Error checking the existence of host state node %s for service %s on host %s: %s", stateID, serviceID, hostID, err)
		return err
	}

	if hpresent {
		t.Delete(hpath)
	}

	if err := t.Commit(); err != nil {
		glog.Errorf("Could not delete service state nodes %s for service %s on host %s: %s", stateID, serviceID, hostID, err)
		return err
	}

	return nil
}

// updateInstance updates the service state and host instances
func updateInstance(conn client.Connection, hostID, stateID string, mutate func(*HostState, *ss.ServiceState)) error {
	glog.V(2).Infof("Updating instance %s", stateID)

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

	t := conn.NewTransaction()
	t.Set(hpath, &hsdata)
	t.Set(spath, &ssnode)
	if err := t.Commit(); err != nil {
		glog.Errorf("Could not update service state %s for service %s on host %s: %s", stateID, serviceID, hostID, err)
		return err
	}

	return nil
}

// removeInstancesOnHost removes all instances for a particular host. Will not
// delete if the instance cannot be found on the host (for when you have
// incongruent data).
func removeInstancesOnHost(conn client.Connection, hostID string) {
	instances, err := conn.Children(hostpath(hostID))
	if err != nil {
		glog.Errorf("Could not look up instances on host %s: %s", hostID, err)
		return
	}
	for _, stateID := range instances {
		var hs HostState
		if err := conn.Get(hostpath(hostID, stateID), &hs); err != nil {
			glog.Warningf("Could not look up host instance %s on host %s: %s", stateID, hostID, err)
		} else if err := removeInstance(conn, hs.ServiceID, hs.HostID, hs.ServiceStateID); err != nil {
			glog.Warningf("Could not remove host instance %s on host %s for service %s: %s", hs.ServiceStateID, hs.HostID, hs.ServiceID, err)
		} else {
			glog.V(2).Infof("Removed instance %s on host %s for service %s", hs.ServiceStateID, hs.HostID, hs.ServiceID, err)
		}
	}
}

// removeInstancesOnService removes all instances for a particular service. Will
// not delete if the instance cannot be found on the service (for when you have
// incongruent data).
func removeInstancesOnService(conn client.Connection, serviceID string) {
	instances, err := conn.Children(servicepath(serviceID))
	if err != nil {
		glog.Errorf("Could not look up instances on service %s: %s", serviceID, err)
		return
	}
	for _, stateID := range instances {
		var state ss.ServiceState
		if err := conn.Get(servicepath(serviceID, stateID), &ServiceStateNode{ServiceState: &state}); err != nil {
			glog.Warningf("Could not look up service instance %s for service %s: %s", stateID, serviceID, err)
		} else if err := removeInstance(conn, state.ServiceID, state.HostID, state.ID); err != nil {
			glog.Warningf("Could not remove service instance %s for service %s on host %s: %s", state.ID, state.ServiceID, state.HostID, err)
		} else {
			glog.V(2).Infof("Removed instance %s for service %s on host %s", state.ID, state.ServiceID, state.HostID, err)
		}
	}
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
