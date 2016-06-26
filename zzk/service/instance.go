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
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	ss "github.com/control-center/serviced/domain/servicestate"
	"github.com/zenoss/glog"
)

// addInstance creates a new service state and host instance
func addInstance(conn client.Connection, poolID string, state ss.ServiceState) error {
	glog.V(2).Infof("Adding instance %+v", state)
	basepth := ""
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	// check the object
	if err := state.ValidEntity(); err != nil {
		glog.Errorf("Could not validate service state %+v: %s", state, err)
		return err
	}

	t := conn.NewTransaction()

	// Prepare the host instance
	hpth := path.Join(basepth, "/hosts", state.HostID, "instances")
	err := conn.CreateIfExists(hpth, &client.Dir{})
	if err != nil && err != client.ErrNodeExists {
		glog.Errorf("Could not set up instance %s on host %s: %s", state.ID, state.HostID, err)
		return err
	}
	hpth = path.Join(hpth, state.ID)
	hdata := NewHostState(&state)
	t.Create(hpth, hdata)

	// Prepare the service instance
	spth := path.Join(basepth, "/services", state.ServiceID, state.ID)
	sdata := &ServiceStateNode{ServiceState: &state}
	t.Create(spth, sdata)

	if err := t.Commit(); err != nil {
		glog.Errorf("Could not create instance %s for service %s on host %s: %s", state.ID, state.ServiceID, state.HostID, err)
		return err
	}
	return nil
}

// removeInstance removes the service state and host instances
func removeInstance(conn client.Connection, poolID, hostID, serviceID, stateID string) error {
	glog.V(2).Infof("Removing instance %s", stateID)
	basepth := ""
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances", stateID)
	spth := path.Join(basepth, "/services", serviceID, stateID)

	t := conn.NewTransaction()

	// Delete the host instance
	if ok, err := conn.Exists(hpth); err != nil {
		glog.Errorf("Could not look up instance %s on host %s: %s", stateID, hostID, err)
		return err
	} else if ok {
		t.Delete(hpth)
	}

	// Delete the service instance
	if ok, err := conn.Exists(spth); err != nil {
		glog.Errorf("Could not look up instance %s for service %s: %s", stateID, serviceID, err)
		return err
	} else if ok {
		t.Delete(spth)
	}

	if err := t.Commit(); err != nil {
		glog.Errorf("Could not delete instance %s from service %s and host %s: %s", stateID, serviceID, hostID, err)
		return err
	}

	return nil
}

// updateInstance updates the service state and host instances
func updateInstance(conn client.Connection, poolID, hostID, stateID string, mutate func(*HostState, *ss.ServiceState)) error {
	glog.V(2).Infof("Updating instance %s", stateID)
	basepth := ""
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	// Get the current host instance
	hpth := path.Join(basepth, "/hosts", hostID, "instances", stateID)
	hdata := HostState{}
	if err := conn.Get(hpth, &hdata); err != nil {
		glog.Errorf("Could not get instance %s for host %s: %s", stateID, hostID, err)
		return err
	}

	// Get the current service instance
	serviceID := hdata.ServiceID
	spth := path.Join(basepth, "/services", serviceID, stateID)
	sdata := ServiceStateNode{ServiceState: &ss.ServiceState{}}
	if err := conn.Get(spth, &sdata); err != nil {
		glog.Errorf("Could not get instance %s from service %s: %s", stateID, serviceID, err)
		return err
	}

	// manipulate the nodes
	mutate(&hdata, sdata.ServiceState)

	if err := conn.NewTransaction().Set(hpth, &hdata).Set(spth, &sdata).Commit(); err != nil {
		glog.Errorf("Could not update instance %s from service %s on host %s: %s", stateID, serviceID, hostID, err)
		return err
	}
	return nil
}

// removeInstancesOnHost removes all instances for a particular host. Will not
// delete if the instance cannot be found on the host (for when you have
// incongruent data).
func removeInstancesOnHost(conn client.Connection, poolID, hostID string) (count int) {
	basepth := ""
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {
		glog.Errorf("Could not look up instances on host %s: %s", hostID, err)
		return
	}
	for _, stateID := range ch {
		hdata := HostState{}
		if err := conn.Get(path.Join(hpth, stateID), &hdata); err != nil {
			glog.Warningf("Could not look up instance %s on host %s: %s", stateID, hostID, err)
			continue
		}
		if err := removeInstance(conn, poolID, hostID, hdata.ServiceID, stateID); err != nil {
			glog.Warningf("Could not remove instance %s on host %s for service %s: %s", stateID, hostID, hdata.ServiceID, err)
			continue
		}
		glog.V(2).Infof("Removed instance %s on host %s for service %s", stateID, hostID, hdata.ServiceID)
		count++
	}
	return
}

// removeInstancesOnService removes all instances for a particular service. Will
// not delete if the instance cannot be found on the service (for when you have
// incongruent data).
func removeInstancesOnService(conn client.Connection, poolID, serviceID string) (count int) {
	basepth := ""
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {
		glog.Errorf("Could not look up instances from service %s: %s", serviceID, err)
		return
	}
	for _, stateID := range ch {
		sdata := ss.ServiceState{}
		if err := conn.Get(path.Join(spth, stateID), &ServiceStateNode{ServiceState: &sdata}); err != nil {
			glog.Warningf("Could not look up instance %s from service %s: %s", stateID, serviceID, err)
			continue
		}
		if err := removeInstance(conn, poolID, sdata.HostID, serviceID, stateID); err != nil {
			glog.Warningf("Could not remove instance %s from service %s: %s", stateID, serviceID, err)
			continue
		}
		glog.V(2).Infof("Removed instance %s from service %s on host %s", stateID, serviceID, sdata.HostID)
		count++
	}
	return
}

// pauseInstance pauses a running host state instance
func pauseInstance(conn client.Connection, poolID, hostID, stateID string) error {
	return updateInstance(conn, poolID, hostID, stateID, func(hsdata *HostState, _ *ss.ServiceState) {
		if hsdata.DesiredState == int(service.SVCRun) {
			glog.V(2).Infof("Pausing service instance %s via host %s", stateID, hostID)
			hsdata.DesiredState = int(service.SVCPause)
		}
	})
}

// resumeInstance resumes a paused host state instance
func resumeInstance(conn client.Connection, poolID, hostID, stateID string) error {
	return updateInstance(conn, poolID, hostID, stateID, func(hsdata *HostState, _ *ss.ServiceState) {
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
	return updateInstance(conn, "", state.HostID, state.ID, func(_ *HostState, ssdata *ss.ServiceState) {
		*ssdata = *state
	})
}

// StopServiceInstance stops a host state instance
func StopServiceInstance(conn client.Connection, poolID, hostID, stateID string) error {
	isOnline, err := IsHostOnline(conn, poolID, hostID)
	if err != nil {
		glog.Errorf("Could not verify whether host %s is online: %s", hostID, err)
		return err
	}

	// If the host is online, try to stop the instance nicely, otherwise remove
	// the instance.
	if isOnline {
		stopInstance := func(h *HostState, s *ss.ServiceState) {
			glog.V(2).Infof("Stopping instance %s on host %s", s.ID, s.HostID)
			h.DesiredState = int(service.SVCStop)
		}
		return updateInstance(conn, "", hostID, stateID, stopInstance)
	} else {
		basepth := ""
		if poolID != "" {
			basepth = path.Join("/pools", poolID)
		}
		hpth := path.Join(basepth, "/hosts", hostID, "instances")
		hdata := HostState{}
		if err := conn.Get(hpth, &hdata); err != nil {
			glog.Errorf("Could not look up instance %s on host %s: %s", stateID, hostID, err)
			return err
		}
		return removeInstance(conn, poolID, hostID, hdata.ServiceID, stateID)
	}
}
