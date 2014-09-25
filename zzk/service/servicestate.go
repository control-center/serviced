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

// UpdateServiceState updates a particular service state
func UpdateServiceState(conn client.Connection, serviceID, stateID string, f func(*servicestate.ServiceState)) error {
	path := servicepath(serviceID, stateID)
	var node ServiceStateNode
	if err := conn.Get(path, &node); err != nil {
		return err
	}
	f(node.ServiceState)
	return conn.Set(path, &node)
}

// GetServiceStatus creates a map of service states to their corresponding status
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

	if hostState.DesiredState == service.SVCStop {
		switch status {
		case dao.Running, dao.Paused:
			status = dao.Stopping
		case dao.Stopped:
			// pass
		default:
			return dao.Status{}, ErrUnknownState
		}
	} else if hostState.DesiredState == service.SVCRun {
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
	} else if hostState.DesiredState == service.SVCPause {
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
