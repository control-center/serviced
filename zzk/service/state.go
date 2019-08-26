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
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
)

var ErrInstanceNotFound = errors.New("instance is not scheduled to a host")

// StateError describes an error from a state CRUD operation
type StateError struct {
	Request   StateRequest
	Operation string
	Message   string
}

func (err StateError) Error() string {
	return fmt.Sprintf("could not %s instance %d from service %s on host %s: %s",
		err.Operation,
		err.Request.InstanceID,
		err.Request.ServiceID,
		err.Request.HostID,
		err.Message)
}

// ErrInvalidStateID is an error that is returned when a state id value is not
// parseable.
var ErrInvalidStateID = errors.New("invalid state id")

// ServiceState provides information of a service state
type ServiceState struct {
	ContainerID string
	ImageUUID   string
	Paused      bool
	PrivateIP   string
	HostIP      string
	AssignedIP  string
	Static      bool
	Imports     []ImportBinding
	Exports     []ExportBinding
	Started     time.Time
	Restarted   time.Time
	Terminated  time.Time
	version     interface{}
}

type CurrentStateContainer struct {
	Status  service.InstanceCurrentState
	version interface{}
}

func (s *CurrentStateContainer) Version() interface{} {
	return s.version
}
func (s *CurrentStateContainer) SetVersion(version interface{}) {
	s.version = version
}

// Version implements client.Node
func (s *ServiceState) Version() interface{} {
	return s.version
}

// SetVersion implements client.Node
func (s *ServiceState) SetVersion(version interface{}) {
	s.version = version
}

// HostState provides information for a particular instance on host for a
// service.
type HostState struct {
	DesiredState service.DesiredState
	Scheduled    time.Time
	version      interface{}
}

// Version implements client.Node
func (s *HostState) Version() interface{} {
	return s.version
}

// SetVersion implements client.Node
func (s *HostState) SetVersion(version interface{}) {
	s.version = version
}

// State is a concatenation of the HostState and ServiceState objects
type State struct {
	HostState
	ServiceState
	CurrentStateContainer
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

// StateRequests is a list of StateRequests for purposes of sorting
type StateRequests []StateRequest

func (reqs StateRequests) Len() int {
	return len(reqs)
}

func (reqs StateRequests) Less(i, j int) bool {
	return reqs[i].InstanceID < reqs[j].InstanceID
}

func (reqs StateRequests) Swap(i, j int) {
	reqs[i], reqs[j] = reqs[j], reqs[i]
}

// SortStateRequest sorts a list of state requests by instance id
func SortStateRequests(reqs []StateRequest) {
	sort.Sort(StateRequests(reqs))
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
	logger := plog.WithFields(log.Fields{
		"hostid":     req.HostID,
		"serviceid":  req.ServiceID,
		"instanceid": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	// Get the current host state
	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", req.StateID())
	hsdat := &HostState{}
	logger = logger.WithField("hspth", hspth)
	if err := conn.Get(hspth, hsdat); err != nil {

		logger.WithError(err).Debug("Could not look up host state")
		return nil, &StateError{
			Request:   req,
			Operation: "get",
			Message:   "could not look up host state",
		}
	}

	// Get the current service state
	sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())
	ssdat := &ServiceState{}
	logger = logger.WithField("sspth", sspth)
	if err := conn.Get(sspth, ssdat); err != nil {

		logger.WithError(err).Debug("Could not look up service state")
		return nil, &StateError{
			Request:   req,
			Operation: "get",
			Message:   "could not look up service state",
		}
	}

	// Get the current state (status)
	cspth := path.Join(sspth, "current")
	logger = logger.WithField("cspth", cspth)
	cstate := &CurrentStateContainer{}
	if err := conn.Get(cspth, cstate); err != nil {
		logger.WithError(err).Debug("Could not look up current state (status)")
		return nil, &StateError{
			Request:   req,
			Operation: "get",
			Message:   "could not look up current state (status)",
		}
	}

	return &State{
		HostState:             *hsdat,
		ServiceState:          *ssdat,
		CurrentStateContainer: *cstate,
		HostID:                req.HostID,
		ServiceID:             req.ServiceID,
		InstanceID:            req.InstanceID,
	}, nil
}

// GetServiceStateHostID returns the hostid of the matching service state
func GetServiceStateHostID(conn client.Connection, poolID, serviceID string, instanceID int) (string, error) {
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {

		logger.WithError(err).Debug("Could not look up states on service")

		// TODO: wrap the error?
		return "", err
	}

	// find the matching state id
	suffix := fmt.Sprintf("-%s-%d", serviceID, instanceID)
	for _, stateID := range ch {
		if strings.HasSuffix(stateID, suffix) {
			hostID, _, _, err := ParseStateID(stateID)
			if err != nil {
				// This should never happen, but handle it anyway
				logger.WithField("stateid", stateID).WithError(err).Debug("Could not parse state")
				return "", err
			}
			logger.WithField("hostid", hostID).Debug("Found state id")
			return hostID, nil
		}
	}

	return "", ErrInstanceNotFound
}

// GetServiceStateIDs returns the parsed state ids of a running service
func GetServiceStateIDs(conn client.Connection, poolID, serviceID string) ([]StateRequest, error) {
	logger := plog.WithField("serviceid", serviceID)

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {

		logger.WithError(err).Debug("Could not look up states on service")

		// TODO: wrap the error?
		return nil, err
	}

	states := make([]StateRequest, len(ch))
	for i, stateID := range ch {
		hostID, _, instanceID, err := ParseStateID(stateID)
		if err != nil {
			logger.WithField("stateid", stateID).WithError(err).Debug("Could not parse state")
			return nil, err
		}

		states[i] = StateRequest{
			PoolID:     poolID,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}
	}

	return states, nil
}

// GetServiceStates returns the states of a running service
func GetServiceStates(conn client.Connection, poolID, serviceID string) ([]State, error) {
	logger := plog.WithField("serviceid", serviceID)

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {

		logger.WithError(err).Debug("Could not look up instances on service")

		// TODO: wrap the error?
		return nil, err
	}

	states := make([]State, len(ch))
	for i, stateID := range ch {
		hostID, _, instanceID, err := ParseStateID(stateID)
		if err != nil {

			logger.WithField("stateid", stateID).WithError(err).Debug("Could not parse state")
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

	return states, nil
}

// GetHostStateIDs returns the state ids running on a host
func GetHostStateIDs(conn client.Connection, poolID, hostID string) ([]StateRequest, error) {
	logger := plog.WithField("hostid", hostID)

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {

		logger.WithError(err).Debug("Could not look up instances on host")

		// TODO: wrap the error?
		return nil, err
	}

	states := make([]StateRequest, len(ch))
	for i, stateID := range ch {
		_, serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {

			logger.WithField("stateid", stateID).WithError(err).Debug("Could not parse state id")
			return nil, err
		}

		states[i] = StateRequest{
			PoolID:     poolID,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}
	}

	logger.WithField("statecount", len(states)).Debug("Loaded states")
	return states, nil
}

// GetHostStates returns the states running on a host
func GetHostStates(conn client.Connection, poolID, hostID string) ([]State, error) {
	logger := plog.WithField("hostid", hostID)

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {

		logger.WithError(err).Debug("Could not look up instances on host")

		// TODO: wrap the error?
		return nil, err
	}

	states := make([]State, len(ch))
	for i, stateID := range ch {
		_, serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {

			logger.WithField("stateid", stateID).WithError(err).Debug("Could not parse state id")
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

	return states, nil
}

// CreateState creates a new service state and host state
func CreateState(conn client.Connection, req StateRequest) error {
	logger := plog.WithFields(log.Fields{
		"hostid":     req.HostID,
		"serviceid":  req.ServiceID,
		"instanceid": req.InstanceID,
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

		logger.WithError(err).Debug("Could not initialize host path")
		return &StateError{
			Request:   req,
			Operation: "create",
			Message:   "could not initialize host path",
		}
	}

	hspth := path.Join(hpth, req.StateID())
	hsdat := &HostState{
		DesiredState: service.SVCRun,
		Scheduled:    time.Now(),
	}
	t.Create(hspth, hsdat)

	// Prepare the service instance
	sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())
	ssdat := &ServiceState{}
	t.Create(sspth, ssdat)

	cspth := path.Join(sspth, "current")
	cstate := &CurrentStateContainer{}
	cstate.Status = service.StateStopped
	t.Create(cspth, cstate)

	if err := t.Commit(); err != nil {
		logger.WithError(err).Debug("Could not commit transaction")
		return &StateError{
			Request:   req,
			Operation: "create",
			Message:   "could not commit transaction",
		}
	}

	logger.Debug("Created state")
	return nil
}

// UpdateState updates the service state and host state
func UpdateState(conn client.Connection, req StateRequest, mutate func(*State) bool) error {
	logger := plog.WithFields(log.Fields{
		"hostid":     req.HostID,
		"serviceid":  req.ServiceID,
		"instanceid": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	// Get the current host state
	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", req.StateID())
	hsdat := &HostState{}
	if err := conn.Get(hspth, hsdat); err != nil {

		logger.WithError(err).Debug("Could not look up host state")
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

		logger.WithError(err).Debug("Could not look up service state")
		return &StateError{
			Request:   req,
			Operation: "update",
			Message:   "could not look up service state",
		}
	}

	// Get the current state (status)
	cspth := path.Join(sspth, "current")
	cstate := &CurrentStateContainer{}
	if err := conn.Get(cspth, cstate); err != nil {
		logger.WithError(err).Debug("Could not look up current state (status)")
		return &StateError{
			Request:   req,
			Operation: "update",
			Message:   "could not look up current state (status)",
		}
	}

	// mutate the states
	hsver, ssver, cstatever := hsdat.Version(), ssdat.Version(), cstate.Version()
	state := &State{
		HostState:             *hsdat,
		ServiceState:          *ssdat,
		CurrentStateContainer: *cstate,
		HostID:                req.HostID,
		ServiceID:             req.ServiceID,
		InstanceID:            req.InstanceID,
	}

	// only commit the transaction if mutate returns true
	if !mutate(state) {
		logger.Debug("Transaction aborted")
		return nil
	}

	// set the version object on the respective states
	*hsdat = state.HostState
	hsdat.SetVersion(hsver)
	*ssdat = state.ServiceState
	ssdat.SetVersion(ssver)
	*cstate = state.CurrentStateContainer
	cstate.SetVersion(cstatever)

	if err := conn.NewTransaction().Set(hspth, hsdat).Set(sspth, ssdat).Set(cspth, cstate).Commit(); err != nil {
		logger.WithError(err).Debug("Could not commit transaction")
		return &StateError{
			Request:   req,
			Operation: "update",
			Message:   "could not commit transaction",
		}
	}

	logger.Debug("Updated state")
	return nil
}

// DeleteState removes the service state and host state
func DeleteState(conn client.Connection, req StateRequest) error {
	// set up logging
	logger := plog.WithFields(log.Fields{
		"hostid":     req.HostID,
		"serviceid":  req.ServiceID,
		"instanceid": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	t := conn.NewTransaction()

	// Delete the host instance
	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", req.StateID())
	if ok, err := conn.Exists(hspth); err != nil {
		logger.WithError(err).Debug("Could not look up host state")

		// CC-2853: only wrap errors that are NOT of type client.ErrNoServer
		if err == client.ErrNoServer {
			return err
		}
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

	sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())

	// Delete the current state (status)
	cspth := path.Join(sspth, "current")
	if ok, err := conn.Exists(cspth); err != nil {
		logger.WithError(err).Debug("Could not look up current state (status)")
		return &StateError{
			Request:   req,
			Operation: "delete",
			Message:   "could not look up current state (status)",
		}
	} else if ok {
		t.Delete(cspth)
	} else {
		logger.Debug("No status to delete on service")
	}

	// Delete the service instance
	if ok, err := conn.Exists(sspth); err != nil {
		logger.WithError(err).Debug("Could not look up service state")

		// CC-2853: only wrap errors that are NOT of type client.ErrNoServer
		if err == client.ErrNoServer {
			return err
		}
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
		logger.WithError(err).Debug("Could not commit transaction")

		// CC-2853: only wrap errors that are NOT of type client.ErrNoServer
		if err == client.ErrNoServer {
			return err
		}
		return &StateError{
			Request:   req,
			Operation: "delete",
			Message:   "could not commit transaction",
		}
	}

	logger.Debug("Deleted state")
	return nil
}

// DeleteServiceStates returns the number of states deleted from a service
func DeleteServiceStates(conn client.Connection, poolID, serviceID string) (count int) {
	logger := plog.WithField("serviceid", serviceID)

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {

		logger.WithError(err).Error("Could not look up states on service")
		return
	}

	for _, stateID := range ch {
		hostID, _, instanceID, err := ParseStateID(stateID)
		if err != nil {

			logger.WithField("stateid", stateID).WithError(err).Warn("Could not parse state")
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
				"hostid":     hostID,
				"instanceid": instanceID,
			}).WithError(err).Warn("Could not delete state")
			continue
		}

		count++
	}

	logger.WithField("statecount", count).Debug("Deleted states")
	return
}

// DeleteHostStates returns the number of states deleted from a host
func DeleteHostStates(conn client.Connection, poolID, hostID string) (count int) {
	logger := plog.WithField("hostid", hostID)

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {

		logger.WithError(err).Error("Could not look up states on host")
		return
	}

	for _, stateID := range ch {
		_, serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {

			logger.WithField("stateid", stateID).WithError(err).Warn("Could not parse state")
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
				"serviceid":  serviceID,
				"instanceid": instanceID,
			}).WithError(err).Warn("Could not delete state")
			continue
		}

		count++
	}

	logger.WithField("statecount", count).Debug("Deleted states")
	return
}

// DeleteHostStatesWhen returns the number of states deleted from a host when
// the state satisfies a particular condition.
func DeleteHostStatesWhen(conn client.Connection, poolID, hostID string, when func(*State) bool) (count int) {
	logger := plog.WithField("hostid", hostID)

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	// which states are running on the host
	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {
		logger.WithError(err).Error("Could not look up states on host")
		return
	}

	for _, stateID := range ch {
		_, serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {
			logger.WithField("stateid", stateID).WithError(err).Warn("Could not parse state")
			continue
		}

		st8log := logger.WithFields(log.Fields{
			"serviceid":  serviceID,
			"instanceid": instanceID,
		})

		req := StateRequest{
			PoolID:     poolID,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}

		state, err := GetState(conn, req)
		if err != nil {
			st8log.WithError(err).Warn("Could not load state from host")
			continue
		}

		if when(state) {
			if err := DeleteState(conn, req); err != nil {
				st8log.WithError(err).Warn("Could not delete state from host")
				continue
			}
			count++
		}
	}

	logger.WithField("statecount", count).Debug("Deleted states")
	return
}

// IsValidState returns true if both the service state and host state exists.
func IsValidState(conn client.Connection, req StateRequest) (bool, error) {
	logger := plog.WithFields(log.Fields{
		"hostid":     req.HostID,
		"serviceid":  req.ServiceID,
		"instanceid": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", req.StateID())
	logger = logger.WithField("hspth", hspth)
	if ok, err := conn.Exists(hspth); err != nil {

		logger.WithError(err).Debug("Could not look up host state")
		return false, &StateError{
			Request:   req,
			Operation: "exists",
			Message:   "could not look up host state",
		}
	} else if !ok {

		logger.Debug("Host state not found")
		return false, nil
	}

	sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())
	logger = logger.WithField("sspth", sspth)
	if ok, err := conn.Exists(sspth); err != nil {
		logger.WithError(err).Debug("Could not look up service state")
		return false, &StateError{
			Request:   req,
			Operation: "exists",
			Message:   "could not look up service state",
		}
	} else if !ok {
		logger.Debug("Service state not found")
		return false, nil
	}

	cspth := path.Join(sspth, "current")
	logger = logger.WithField("cspth", cspth)
	if ok, err := conn.Exists(cspth); err != nil {
		logger.WithError(err).Debug("Could not look up current state (status)")
		return false, &StateError{
			Request:   req,
			Operation: "exists",
			Message:   "could not look up current state (status)",
		}
	} else if !ok {
		logger.Debug("Service status not found")
		return false, nil
	}

	return true, nil
}

// CleanHostStates deletes host states with invalid state ids or incongruent
// data.
func CleanHostStates(conn client.Connection, poolID, hostID string) error {
	logger := plog.WithField("hostid", hostID)

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {

		logger.WithError(err).Debug("Could not look up states on host")

		// TODO: wrap error?
		return err
	}

	for _, stateID := range ch {
		st8log := logger.WithField("stateid", stateID)

		// parse the state id
		_, serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {

			st8log.Debug("Deleting invalid host state")

			hspth := path.Join(hpth, stateID)
			if err := conn.Delete(hspth); err != nil && err != client.ErrNoNode {

				st8log.WithError(err).Debug("Could not delete invalid host state")

				// TODO: wrap error
				return err
			}

			st8log.Warn("Deleted invalid host state")
			continue
		}

		st8log = st8log.WithFields(log.Fields{
			"serviceid":  serviceID,
			"instanceid": instanceID,
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

			st8log.Warn("Deleted incongruent host state")
			continue
		}
	}

	return nil
}

// CleanServiceStates deletes service states with invalid state ids or
// incongruent data.
func CleanServiceStates(conn client.Connection, poolID, serviceID string) error {
	logger := plog.WithField("serviceid", serviceID)

	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {

		logger.WithError(err).Debug("Could not look up states on service")

		// TODO: wrap error?
		return err
	}

	for _, stateID := range ch {
		st8log := logger.WithField("stateid", stateID)

		hostID, _, instanceID, err := ParseStateID(stateID)
		if err != nil {

			st8log.Debug("Deleting invalid service state")

			sspth := path.Join(spth, stateID)
			if err := conn.Delete(sspth); err != nil && err != client.ErrNoNode {

				st8log.WithError(err).Debug("Could not clean up invalid service state")

				// TODO: wrap error?
				return err
			}

			st8log.Warn("Cleaned up invalid service state")
			continue
		}

		st8log = st8log.WithFields(log.Fields{
			"hostid":     hostID,
			"instanceid": instanceID,
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

			st8log.Warn("Deleted incongruent service state")
			continue
		}
	}

	return nil
}

// MonitorState returns the state object once it satisfies certain
// conditions.
func MonitorState(cancel <-chan struct{}, conn client.Connection, req StateRequest, check func(s *State, exists bool) bool) (*State, error) {
	logger := plog.WithFields(log.Fields{
		"hostid":     req.HostID,
		"serviceid":  req.ServiceID,
		"instanceid": req.InstanceID,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	// Monitor the state of the object
	done := make(chan struct{})
	defer func() { close(done) }()
	for {
		// Get the current host state and set the watch
		hspth := path.Join(basepth, "/hosts", req.HostID, "instances", req.StateID())
		hsok := true
		hsdat := &HostState{}
		hsev, err := conn.GetW(hspth, hsdat, done)
		if err == client.ErrNoNode {
			hsok = false
		} else if err != nil {
			logger.WithError(err).Debug("Could not watch host state")
			return nil, &StateError{
				Request:   req,
				Operation: "watch",
				Message:   "could not watch host state",
			}
		}

		// Get the current service state and set the watch
		sspth := path.Join(basepth, "/services", req.ServiceID, req.StateID())
		ssok := true
		ssdat := &ServiceState{}
		ssev, err := conn.GetW(sspth, ssdat, done)
		if err == client.ErrNoNode {
			ssok = false
		} else if err != nil {
			logger.WithError(err).Debug("Could not watch service state")
			return nil, &StateError{
				Request:   req,
				Operation: "watch",
				Message:   "could not watch service state",
			}
		}

		cspth := path.Join(sspth, "current")
		csok := true
		cstate := &CurrentStateContainer{}
		csev, err := conn.GetW(cspth, cstate, done)
		if err == client.ErrNoNode {
			csok = false
		} else if err != nil {
			logger.WithError(err).Debug("Could not watch service status")
			return nil, &StateError{
				Request:   req,
				Operation: "watch",
				Message:   "could not watch service status",
			}
		}

		// Ensure the nodes are compatible
		if hsok != ssok || ssok != csok {
			logger.WithFields(log.Fields{
				"hoststateexists":     hsok,
				"servicestateexists":  ssok,
				"servicestatusexists": csok,
			}).Debug("Incongruent state")
			return nil, &StateError{
				Request:   req,
				Operation: "watch",
				Message:   "incongruent state",
			}
		}
		exists := hsok && ssok && csok

		// Does the state statisfy the requirements?
		state := &State{
			HostState:             *hsdat,
			ServiceState:          *ssdat,
			CurrentStateContainer: *cstate,
			HostID:                req.HostID,
			ServiceID:             req.ServiceID,
			InstanceID:            req.InstanceID,
		}

		// Only return the state if the value is true.  Return an error if the
		// state has been deleted.
		if check(state, exists) {
			return state, nil
		} else if !exists {
			logger.Debug("State does not exist")
			return nil, &StateError{
				Request:   req,
				Operation: "watch",
				Message:   "state does not exist",
			}
		}

		// Wait for something to happen
		select {
		case <-hsev:
		case <-ssev:
		case <-csev:
		case <-cancel:
			logger.Debug("Aborted state monitor")
			return nil, nil
		}

		close(done)
		done = make(chan struct{})
	}
}
