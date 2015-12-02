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

package service

import (
	"errors"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

var (
	ErrUnknownState = errors.New("unknown state")
)

// ServiceStateNode is the zookeeper client node for service states
type ServiceStateNode struct {
	*servicestate.ServiceState
	version interface{}
}

// Version implements client.Node
func (node *ServiceStateNode) Version() interface{} { return node.version }

// SetVersion implements client.Node
func (node *ServiceStateNode) SetVersion(version interface{}) { node.version = version }

// GetServiceState gets a service state
func GetServiceState(conn client.Connection, state *servicestate.ServiceState, serviceID string, stateID string) error {
	return conn.Get(servicepath(serviceID, stateID), &ServiceStateNode{ServiceState: state})
}

// GetServiceStates gets all service states for a particular service
func GetServiceStates(conn client.Connection, serviceIDs ...string) (states []servicestate.ServiceState, err error) {
	for _, serviceID := range serviceIDs {
		stateIDs, err := conn.Children(servicepath(serviceID))
		if err != nil {
			return nil, err
		}

		for _, stateID := range stateIDs {
			var state servicestate.ServiceState
			if err := GetServiceState(conn, &state, serviceID, stateID); err != nil {
				return nil, err
			}
			states = append(states, state)
		}
	}
	return states, nil
}

// wait waits for an individual service state to reach its desired state
func wait(shutdown <-chan interface{}, conn client.Connection, serviceID, stateID string, dstate service.DesiredState) error {
	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		var node ServiceStateNode
		event, err := conn.GetW(servicepath(serviceID, stateID), &node, done)
		if err == client.ErrNoNode {
			// if the node no longer exists, then there is nothing to watch, so we are done
			return nil
		} else if err != nil {
			glog.Errorf("Got an error while looking for %s (%s): %s", stateID, serviceID, err)
			return err
		}

		switch dstate {
		case service.SVCStop:
			// pass through, because the node needs to be deleted to be considered Stopped
		case service.SVCRun, service.SVCRestart:
			if node.IsRunning() {
				// instance reached the desired state
				return nil
			}
		case service.SVCPause:
			if node.IsPaused() {
				// instance reached the desired state
				return nil
			}
		}

		// wait for something to change on the node or shutdown
		select {
		case <-event:
		case <-shutdown:
			return zzk.ErrShutdown
		}

		close(done)
		done = make(chan struct{})
	}
}

// GetServiceStatus creates a map of service states to their corresponding status
// The ServiceStatus objects returned will NOT include healthcheck info
func GetServiceStatus(conn client.Connection, serviceID string) (map[string]dao.ServiceStatus, error) {
	states, err := GetServiceStates(conn, serviceID)
	if err != nil {
		glog.Errorf("Could not get states for service %s: %s", serviceID, err)
		return nil, err
	}

	stats := make(map[string]dao.ServiceStatus)
	for _, state := range states {
		status, err := getStatus(conn, &state)
		if err != nil {
			glog.Errorf("Error looking up status %s for service %s: %s", state.ID, serviceID, err)
			return nil, err
		}
		stats[state.ID] = dao.ServiceStatus{State: state, Status: status}
	}

	return stats, err
}

// getStatus computes the status of a service state
func getStatus(conn client.Connection, state *servicestate.ServiceState) (dao.Status, error) {
	var status dao.Status

	// Set the state based on the service state object
	if !state.IsRunning() {
		status = dao.Stopped
	} else if state.IsPaused() {
		status = dao.Paused
	} else {
		status = dao.Running
	}

	// Set the state based on the host state object
	var hostState HostState
	if err := conn.Get(hostpath(state.HostID, state.ID), &hostState); err != nil && err != client.ErrNoNode {
		return dao.Status{}, err
	}

	if hostState.DesiredState == int(service.SVCStop) {
		switch status {
		case dao.Running, dao.Paused:
			status = dao.Stopping
		case dao.Stopped:
			// pass
		default:
			return dao.Status{}, ErrUnknownState
		}
	} else if hostState.DesiredState == int(service.SVCRun) {
		switch status {
		case dao.Stopped:
			status = dao.Starting
		case dao.Paused:
			status = dao.Resuming
		case dao.Running:
			// pass
		default:
			return dao.Status{}, ErrUnknownState
		}
	} else if hostState.DesiredState == int(service.SVCPause) {
		switch status {
		case dao.Running:
			status = dao.Pausing
		case dao.Paused, dao.Stopped:
			// pass
		default:
			return dao.Status{}, ErrUnknownState
		}
	} else {
		return dao.Status{}, ErrUnknownState
	}

	return status, nil
}
