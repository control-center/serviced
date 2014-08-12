// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
func GetServiceStates(conn client.Connection, serviceIDs ...string) (states []*servicestate.ServiceState, err error) {
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
			states = append(states, &state)
		}
	}
	return states, nil
}

// UpdateServiceState updates a particular service state
func UpdateServiceState(conn client.Connection, state *servicestate.ServiceState) error {
	var node ServiceStateNode
	path := servicepath(state.ServiceID, state.ID)
	if err := conn.Get(path, &node); err != nil {
		return err
	}
	node.ServiceState = state
	return conn.Set(path, &node)
}

// GetServiceStatus creates a map of service states to their corresponding status
func GetServiceStatus(conn client.Connection, serviceID string) (map[*servicestate.ServiceState]dao.Status, error) {
	states, err := GetServiceStates(conn, serviceID)
	if err != nil {
		glog.Errorf("Could not get states for service %s: %s", serviceID, err)
		return nil, err
	}

	stats := make(map[*servicestate.ServiceState]dao.Status)
	for _, state := range states {
		status, err := getStatus(conn, state)
		if err != nil {
			glog.Errorf("Error looking up status %s for service %s: %s", state.ID, serviceID, err)
			return nil, err
		}
		stats[state] = status
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