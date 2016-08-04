// Copyright 2016 The Serviced Authors.
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
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
)

// StateError describes an error from a state CRUD operation
type StateError struct {
	Request   StateRequest
	Operation string
	Message   string
}

func (err StateError) Error() string {
	return fmt.Sprintf("could not %s instance %d from service %s on host %s: %s", err.Operation, err.Request.InstanceID, err.Request.ServiceID, err.Request.ServiceID, err.Message)
}

// ErrInvalidStateID is an error that is returned when a state id value is not
// parseable.
var ErrInvalidStateID = errors.New("invalid state id")

// ServiceState provides information for a particular instance of a service
type ServiceState struct {
	DockerID   string
	ImageID    string
	Paused     bool
	Started    time.Time
	Terminated time.Time
	version    interface{}
}

// Version implements client.Node
func (s *ServiceState) Version() interface{} {
	return s.version
}

// SetVersion implements client.Node
func (s *ServiceState) SetVersion(version interface{}) {
	s.version = version
}

// HostState2 provides information for a particular instance on host for a
// service.
// TODO: update name when the calls are swapped on the listeners
type HostState2 struct {
	DesiredState service.DesiredState
	Scheduled    time.Time
	version      interface{}
}

// Version implements client.Node
func (s *HostState2) Version() interface{} {
	return s.version
}

// SetVersion implements client.Node
func (s *HostState2) SetVersion(version interface{}) {
	s.version = version
}

// State is a concatenation of the HostState and ServiceState objects
type State struct {
	HostState2
	ServiceState
	HostID     string
	ServiceID  string
	InstanceID int
}

// StateRequest provides information for service instance CRUD
type StateRequest struct {
	PoolID     string
	HostID     string
	ServiceID  string
	InstanceID int
}

func (req StateRequest) StateID() string {
	return fmt.Sprintf("%s-%s-%d", req.HostID, req.ServiceID, req.InstanceID)
}

// ParseStateID returns the host, service, and instance id from the given state
// id
func ParseStateID(stateID string) (string, string, int, error) {
	parts := strings.SplitN(stateID, "-", 3)
	if len(parts) != 3 {
		return "", "", 0, ErrInvalidStateID
	}
	instanceID, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", "", 0, ErrInvalidStateID
	}
	return parts[0], parts[1], instanceID, nil
}

// GetState returns the service state and host state for a particular instance.
func GetState(conn client.Connection, req StateRequest) (*State, error) {
	// set up logging
	logger := log.WithFields(log.Fields{
		"HostID":     req.HostID,
		"ServiceID":  req.ServiceID,
		"InstanceID": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	// Get the current host state
	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", req.StateID())
	hsdat := &HostState2{}
	if err := conn.Get(hspth, hsdat); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up host state")

		return nil, &StateError{
			Request:   req,
			Operation: "get",
			Message:   "could not look up host state",
		}
	}
	logger.Debug("Found the host state")

	// Get the current service state
	sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())
	ssdat := &ServiceState{}
	if err := conn.Get(sspth, ssdat); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up service state")

		return nil, &StateError{
			Request:   req,
			Operation: "get",
			Message:   "could not look up service state",
		}
	}
	logger.Debug("Found the service state")

	return &State{
		HostState2:   *hsdat,
		ServiceState: *ssdat,
		HostID:       req.HostID,
		ServiceID:    req.ServiceID,
		InstanceID:   req.InstanceID,
	}, nil
}

// GetServiceStates2 returns the states of a running service
// TODO: update name when calls are swapped
func GetServiceStates2(conn client.Connection, poolID, serviceID string) ([]State, error) {
	// set up logging
	logger := log.WithFields(log.Fields{
		"ServiceID": serviceID,
	})

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {

		// TODO: wrap the error?
		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up instances on service")

		return nil, err
	}

	states := make([]State, len(ch))
	for i, stateID := range ch {
		hostID, _, instanceID, err := ParseStateID(stateID)
		if err != nil {

			logger.WithFields(log.Fields{
				"StateID": stateID,
				"Error":   err,
			}).Debug("Could not parse state")

			return nil, err
		}

		req := StateRequest{
			PoolID:     poolID,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}

		state, err := GetState(conn, req)
		if err != nil {
			return nil, err
		}

		states[i] = *state
	}

	logger.WithFields(log.Fields{
		"Count": len(states),
	}).Debug("Loaded states")

	return states, nil
}

// GetHostStates returns the states running on a host
func GetHostStates(conn client.Connection, poolID, hostID string) ([]State, error) {
	logger := log.WithFields(log.Fields{
		"HostID": hostID,
	})

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {

		// TODO: wrap the error?
		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up instances on host")

		return nil, err
	}

	states := make([]State, len(ch))
	for i, stateID := range ch {
		_, serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {

			logger.WithFields(log.Fields{
				"StateID": stateID,
				"Error":   err,
			}).Debug("Could not parse state")

			return nil, err
		}

		req := StateRequest{
			PoolID:     poolID,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}

		state, err := GetState(conn, req)
		if err != nil {
			return nil, err
		}
		states[i] = *state
	}

	logger.WithFields(log.Fields{
		"Count": len(states),
	}).Debug("Loaded states")

	return states, nil
}

// CreateState creates a new service state and host state
func CreateState(conn client.Connection, req StateRequest) error {
	// set up logging
	logger := log.WithFields(log.Fields{
		"HostID":     req.HostID,
		"ServiceID":  req.ServiceID,
		"InstanceID": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	t := conn.NewTransaction()

	// Prepare the host instance
	hpth := path.Join(basepth, "/hosts", req.HostID, "instances")
	err := conn.CreateIfExists(hpth, &client.Dir{})
	if err != nil && err != client.ErrNodeExists {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not initialize host path")

		return &StateError{
			Request:   req,
			Operation: "create",
			Message:   "could not initialize host path",
		}
	}

	hspth := path.Join(hpth, req.StateID())
	hsdat := &HostState2{
		DesiredState: service.SVCRun,
		Scheduled:    time.Now(),
	}
	t.Create(hspth, hsdat)

	// Prepare the service instance
	sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())
	ssdat := &ServiceState{}
	t.Create(sspth, ssdat)

	if err := t.Commit(); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not commit transaction")

		return &StateError{
			Request:   req,
			Operation: "create",
			Message:   fmt.Sprintf("could not commit transaction"),
		}
	}

	logger.Debug("Created state")

	return nil
}

// UpdateState updates the service state and host state
func UpdateState(conn client.Connection, req StateRequest, mutate func(*State)) error {
	// set up logging
	logger := log.WithFields(log.Fields{
		"HostID":     req.HostID,
		"ServiceID":  req.ServiceID,
		"InstanceID": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	// Get the current host state
	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", req.StateID())
	hsdat := &HostState2{}
	if err := conn.Get(hspth, hsdat); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up host state")

		return &StateError{
			Request:   req,
			Operation: "update",
			Message:   "could not look up host state",
		}
	}

	// Get the current service state
	sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())
	ssdat := &ServiceState{}
	if err := conn.Get(sspth, ssdat); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up service state")

		return &StateError{
			Request:   req,
			Operation: "update",
			Message:   "could not look up service state",
		}
	}

	// mutate the states
	hsver, ssver := hsdat.Version(), ssdat.Version()
	state := &State{
		HostState2:   *hsdat,
		ServiceState: *ssdat,
		HostID:       req.HostID,
		ServiceID:    req.ServiceID,
		InstanceID:   req.InstanceID,
	}
	mutate(state)

	// set the version object on the respective states
	*hsdat = state.HostState2
	hsdat.SetVersion(hsver)
	*ssdat = state.ServiceState
	ssdat.SetVersion(ssver)

	if err := conn.NewTransaction().Set(hspth, hsdat).Set(sspth, ssdat).Commit(); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not commit transaction")

		return &StateError{
			Request:   req,
			Operation: "update",
			Message:   fmt.Sprintf("could not commit transaction"),
		}
	}

	logger.Debug("Updated state")

	return nil
}

// DeleteState removes the service state and host state
func DeleteState(conn client.Connection, req StateRequest) error {
	// set up logging
	logger := log.WithFields(log.Fields{
		"HostID":     req.HostID,
		"ServiceID":  req.ServiceID,
		"InstanceID": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	t := conn.NewTransaction()

	// Delete the host instance
	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", req.StateID())
	if ok, err := conn.Exists(hspth); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up host state")

		return &StateError{
			Request:   req,
			Operation: "delete",
			Message:   "could not look up host state",
		}
	} else if ok {
		t.Delete(hspth)
	} else {
		logger.Debug("No state to delete on host")
	}

	// Delete the service instance
	sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())
	if ok, err := conn.Exists(sspth); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up service state")

		return &StateError{
			Request:   req,
			Operation: "delete",
			Message:   "could not look up service state",
		}
	} else if ok {
		t.Delete(sspth)
	} else {
		logger.Debug("No state to delete on service")
	}

	if err := t.Commit(); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not commit transaction")

		return &StateError{
			Request:   req,
			Operation: "delete",
			Message:   fmt.Sprintf("could not commit transaction"),
		}
	}

	logger.Debug("Deleted state")

	return nil
}

// DeleteServiceStates returns the number of states deleted from a service
func DeleteServiceStates(conn client.Connection, poolID, serviceID string) (count int) {
	// set up logging
	logger := log.WithFields(log.Fields{
		"ServiceID": serviceID,
	})

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Error("Could not look up states on service")

		return
	}

	for _, stateID := range ch {
		hostID, _, instanceID, err := ParseStateID(stateID)
		if err != nil {

			logger.WithFields(log.Fields{
				"StateID": stateID,
				"Error":   err,
			}).Warn("Could not parse state")

			continue
		}

		req := StateRequest{
			PoolID:     poolID,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}

		if err := DeleteState(conn, req); err != nil {

			logger.WithFields(log.Fields{
				"HostID":     hostID,
				"InstanceID": instanceID,
				"Error":      err,
			}).Warn("Could not delete state")

			continue
		}

		count++
	}

	logger.WithFields(log.Fields{
		"Count": count,
	}).Debug("Deleted states")

	return
}

// DeleteHostStates returns the number of states deleted from a host
func DeleteHostStates(conn client.Connection, poolID, hostID string) (count int) {
	// set up logging
	logger := log.WithFields(log.Fields{
		"HostID": hostID,
	})

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Error("Could not look up states on host")

		return
	}

	for _, stateID := range ch {
		_, serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {

			logger.WithFields(log.Fields{
				"StateID": stateID,
				"Error":   err,
			}).Warn("Could not parse state")

			continue
		}

		req := StateRequest{
			PoolID:     poolID,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}

		if err := DeleteState(conn, req); err != nil {

			logger.WithFields(log.Fields{
				"HostID":     hostID,
				"InstanceID": instanceID,
				"Error":      err,
			}).Warn("Could not delete state")

			continue
		}

		count++
	}

	logger.WithFields(log.Fields{
		"Count": count,
	}).Debug("Deleted states")

	return
}

// IsValidState returns true if both the service state and host state exists.
func IsValidState(conn client.Connection, req StateRequest) (bool, error) {
	// set up logging
	logger := log.WithFields(log.Fields{
		"HostID":     req.HostID,
		"ServiceID":  req.ServiceID,
		"InstanceID": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", req.StateID())
	if ok, err := conn.Exists(hspth); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up host state")

		return false, &StateError{
			Request:   req,
			Operation: "exists",
			Message:   "could not look up host state",
		}
	} else if !ok {

		logger.Debug("Host state not found")
		return false, nil
	}

	logger.Debug("Found the host state")

	sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())
	if ok, err := conn.Exists(sspth); err != nil {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up service state")

		return false, &StateError{
			Request:   req,
			Operation: "exists",
			Message:   "could not look up service state",
		}
	} else if !ok {

		logger.Debug("Service state not found")
		return false, nil
	}

	logger.Debug("Found the service state")
	return true, nil
}

// CleanHostStates deletes host states with invalid state ids or incongruent
// data.
func CleanHostStates(conn client.Connection, poolID, hostID string) error {
	// set up logging
	logger := log.WithFields(log.Fields{
		"HostID": hostID,
	})

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up states on host")

		// TODO: wrap error?
		return err
	}

	for _, stateID := range ch {

		_, serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {

			// clean up bad state id
			stateLogger := logger.WithFields(log.Fields{
				"StateID": stateID,
			})

			hspth := path.Join(hpth, stateID)
			if err := conn.Delete(hspth); err != nil && err != client.ErrNoNode {

				stateLogger.WithFields(log.Fields{
					"Error": err,
				}).Debug("Could not clean up invalid host state")

				// TODO: wrap error?
				return err
			}

			stateLogger.Warn("Cleaned up invalid host state")
			continue
		}

		stateLogger := logger.WithFields(log.Fields{
			"ServiceID":  serviceID,
			"InstanceID": instanceID,
		})

		req := StateRequest{
			PoolID:     poolID,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}

		if ok, err := IsValidState(conn, req); err != nil {
			return err
		} else if !ok {
			if err := DeleteState(conn, req); err != nil {
				return err
			}

			stateLogger.Warn("Cleaned up incongruent host state")
			continue
		}
	}

	return nil
}

// CleanServiceStates deletes service states with invalid state ids or
// incongruent data.
func CleanServiceStates(conn client.Connection, poolID, serviceID string) error {
	// set up logging
	logger := log.WithFields(log.Fields{
		"ServiceID": serviceID,
	})

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {

		logger.WithFields(log.Fields{
			"Error": err,
		}).Debug("Could not look up states on service")

		// TODO: wrap error?
		return err
	}

	for _, stateID := range ch {
		hostID, _, instanceID, err := ParseStateID(stateID)
		if err != nil {

			// clean up bad state id
			stateLogger := logger.WithFields(log.Fields{
				"StateID": stateID,
			})

			sspth := path.Join(spth, stateID)
			if err := conn.Delete(sspth); err != nil && err != client.ErrNoNode {

				stateLogger.WithFields(log.Fields{
					"Error": err,
				}).Debug("Could not clean up invalid service state")

				// TODO: wrap error?
				return err
			}

			stateLogger.Warn("Cleaned up invalid service state")
			continue
		}

		stateLogger := logger.WithFields(log.Fields{
			"HostID":     hostID,
			"InstanceID": instanceID,
		})

		req := StateRequest{
			PoolID:     poolID,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}

		if ok, err := IsValidState(conn, req); err != nil {
			return err
		} else if !ok {
			if err := DeleteState(conn, req); err != nil {
				return err
			}

			stateLogger.Warn("Cleaned up incongruent service state")
			continue
		}
	}

	return nil
}
