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

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
)

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

func (s *ServiceState) Version() interface{} {
	return s.version
}

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

func (s *HostState2) Version() interface{} {
	return s.version
}

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

// ParseStateID returns the instance id from the given state id
func ParseStateID(stateID string) (string, int, error) {
	parts := strings.SplitN(stateID, "-", 2)
	if len(parts) != 2 {
		return "", 0, ErrInvalidStateID
	}
	instanceID, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, ErrInvalidStateID
	}
	return parts[0], instanceID, nil
}

// GetState returns the service state and host state for a particular instance.
func GetState(conn client.Connection, req StateRequest) (*State, error) {
	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	// Get the current host state
	hsval := fmt.Sprintf("%s-%d", req.ServiceID, req.InstanceID)
	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", hsval)
	hsdat := &HostState2{}
	if err := conn.Get(hspth, hsdat); err != nil {
		// TODO: error on could not get host instance
		return nil, err
	}

	// Get the current service state
	ssval := fmt.Sprintf("%s-%d", req.HostID, req.InstanceID)
	sspth := path.Join(basepth, "/services", req.ServiceID, ssval)
	ssdat := &ServiceState{}
	if err := conn.Get(sspth, ssdat); err != nil {
		// TODO: error on could not get service instance
		return nil, err
	}

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
	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	spth := path.Join(basepth, "/services", serviceID)
	ch, err := conn.Children(spth)
	if err != nil && err != client.ErrNoNode {
		// TODO: error on could not get instances of service
		return nil, err
	}

	states := make([]State, len(ch))
	for i, stateID := range ch {
		hostID, instanceID, err := ParseStateID(stateID)
		if err != nil {
			// TODO: error on invalid state
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
			// TODO: error on building state
			return nil, err
		}

		states[i] = *state
	}

	return states, nil
}

// GetHostStates returns the states running on a host
func GetHostStates(conn client.Connection, poolID, hostID string) ([]State, error) {
	basepth := "/"
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}

	hpth := path.Join(basepth, "/hosts", hostID, "instances")
	ch, err := conn.Children(hpth)
	if err != nil && err != client.ErrNoNode {
		// TODO: error on could not get instances on host
		return nil, err
	}

	states := make([]State, len(ch))
	for i, stateID := range ch {
		serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {
			// TODO: error on invalid state
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
			// TODO: error on building state
			return nil, err
		}

		states[i] = *state
	}

	return states, nil
}

// CreateState creates a new service state and host state
func CreateState(conn client.Connection, req StateRequest) error {
	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	t := conn.NewTransaction()

	// Prepare the host instance
	hpth := path.Join(basepth, "/hosts", req.HostID, "instances")
	err := conn.CreateIfExists(hpth, &client.Dir{})
	if err != nil && err != client.ErrNodeExists {
		// TODO: error on could not set up host instance
		return err
	}
	hsval := fmt.Sprintf("%s-%d", req.ServiceID, req.InstanceID)
	hspth := path.Join(hpth, hsval)
	hsdat := &HostState2{
		DesiredState: service.SVCRun,
		Scheduled:    time.Now(),
	}
	t.Create(hspth, hsdat)

	// Prepare the service instance
	ssval := fmt.Sprintf("%s-%d", req.HostID, req.InstanceID)
	sspth := path.Join(basepth, "/services", req.ServiceID, ssval)
	ssdat := &ServiceState{}
	t.Create(sspth, ssdat)

	if err := t.Commit(); err != nil {
		// TODO: error on create instance
		return err
	}
	return nil
}

// UpdateState updates the service state and host state
func UpdateState(conn client.Connection, req StateRequest, mutate func(*HostState2, *ServiceState)) error {
	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	// Get the current host state
	hsval := fmt.Sprintf("%s-%d", req.ServiceID, req.InstanceID)
	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", hsval)
	hsdat := &HostState2{}
	if err := conn.Get(hspth, hsdat); err != nil {
		// TODO: error on could not get host instance
		return err
	}

	// Get the current service state
	ssval := fmt.Sprintf("%s-%d", req.HostID, req.InstanceID)
	sspth := path.Join(basepth, "/services", req.ServiceID, ssval)
	ssdat := &ServiceState{}
	if err := conn.Get(sspth, ssdat); err != nil {
		// TODO: error on could not get service instance
		return err
	}

	// mutate the states
	mutate(hsdat, ssdat)

	if err := conn.NewTransaction().Set(hspth, hsdat).Set(sspth, ssdat).Commit(); err != nil {
		// TODO: error on update instance
		return err
	}
	return nil
}

// DeleteState removes the service state and host state
func DeleteState(conn client.Connection, req StateRequest) error {
	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	t := conn.NewTransaction()

	// Delete the host instance
	hsval := fmt.Sprintf("%s-%d", req.ServiceID, req.InstanceID)
	hspth := path.Join(basepth, "/hosts", req.HostID, "instances", hsval)
	if ok, err := conn.Exists(hspth); err != nil {
		// TODO: error on could not check host instance
		return err
	} else if ok {
		t.Delete(hspth)
	} else {
		// TODO: log on nothing to delete
	}

	// Delete the service instance
	ssval := fmt.Sprintf("%s-%d", req.HostID, req.InstanceID)
	sspth := path.Join(basepth, "/services", req.ServiceID, ssval)
	if ok, err := conn.Exists(sspth); err != nil {
		// TODO: error on could not check service instance
		return err
	} else if ok {
		t.Delete(sspth)
	} else {
		// TODO: log on nothing to delete
	}

	if err := t.Commit(); err != nil {
		// TODO: error on delete instance
		return err
	}
	return nil
}
