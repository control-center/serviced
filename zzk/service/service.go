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
	"fmt"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
)

// ServiceError manages service errors
type ServiceError struct {
	Action    string
	ServiceID string
	Message   string
}

func (err ServiceError) Error() string {
	return fmt.Sprintf("could not %s service %s: %s", err.Action, err.ServiceID, err.Message)
}

// ServiceNode is the storage object for service data
type ServiceNode struct {
	//*service.Service
	ID            string
	Name          string
	DesiredState  int
	Instances     int
	RAMCommitment utils.EngNotation
	ChangeOptions []string
	//non-service fields
	Locked  bool
	version interface{}
}

func NewServiceNodeFromService(s *service.Service) *ServiceNode {
	return &ServiceNode{
		ID:            s.ID,
		Name:          s.Name,
		DesiredState:  s.DesiredState,
		Instances:     s.Instances,
		RAMCommitment: s.RAMCommitment,
		ChangeOptions: s.ChangeOptions,
	}
}

func (s *ServiceNode) AsService() *service.Service {
	return &service.Service{
		ID:            s.ID,
		Name:          s.Name,
		DesiredState:  s.DesiredState,
		Instances:     s.Instances,
		RAMCommitment: s.RAMCommitment,
		ChangeOptions: s.ChangeOptions,
	}
}

// Version implements client.Node
func (s *ServiceNode) Version() interface{} {
	return s.version
}

// SetVersion implements client.Node
func (s *ServiceNode) SetVersion(version interface{}) {
	s.version = version
}

// UpdateService creates the service if it doesn't exist or updates it if it
// does exist. (uses a pool-based connection)
func UpdateService(conn client.Connection, svc service.Service, setLockOnCreate, setLockOnUpdate bool) error {
	pth := path.Join("/services", svc.ID)

	logger := plog.WithFields(log.Fields{
		"poolid":    svc.PoolID,
		"serviceid": svc.ID,
		"zkpath":    pth,
	})

	// create the /services path if it doesn't exist
	if err := conn.CreateIfExists("/services", &client.Dir{}); err != nil && err != client.ErrNodeExists {
		logger.WithError(err).Debug("Could not initialize services path in zookeeper")
		return &ServiceError{
			Action:    "update",
			ServiceID: svc.ID,
			Message:   "could not initialize services path in zookeeper",
		}
	}

	// create the service if it doesn't exist
	// setLockOnCreate sets the lock as the node is created
	sn := NewServiceNodeFromService(&svc)
	sn.Locked = setLockOnCreate
	if err := conn.CreateIfExists(pth, sn); err == client.ErrNodeExists {

		// the node exists so get it and update it
		node := &ServiceNode{}
		if err := conn.Get(pth, node); err != nil && err != client.ErrEmptyNode {

			logger.WithError(err).Debug("Could not get service entry from zookeeper")
			return &ServiceError{
				Action:    "update",
				ServiceID: svc.ID,
				Message:   "could not get service for update",
			}
		}

		// setLockOnUpdate sets the lock to true if enabled, otherwise it uses
		// whatever existing value was previously set on the node.
		if setLockOnUpdate {
			node.Locked = true
		}
		sn.SetVersion(node.Version())
		if err := conn.Set(pth, sn); err != nil {

			logger.WithError(err).Debug("Could not update service entry in zookeeper")
			return &ServiceError{
				Action:    "update",
				ServiceID: svc.ID,
				Message:   "could not update service",
			}
		}

		logger.Debug("Updated entry for service in zookeeper")
		return nil
	} else if err != nil {

		logger.WithError(err).Debug("Could not create service entry in zookeeper")
		return &ServiceError{
			Action:    "update",
			ServiceID: svc.ID,
			Message:   "could not create service",
		}
	}

	logger.Debug("Created entry for service in zookeeper")
	return nil
}

// getLiteService prepares a service object as it is written into zookeeper
func getLiteService(svc service.Service) *service.Service {
	return &service.Service{
		ID:              svc.ID,
		Name:            svc.Name,
		Startup:         svc.Startup,
		Environment:     svc.Environment,
		Instances:       svc.Instances,
		ChangeOptions:   svc.ChangeOptions,
		ImageID:         svc.ImageID,
		LogConfigs:      svc.LogConfigs,
		DesiredState:    svc.DesiredState,
		HostPolicy:      svc.HostPolicy,
		Privileged:      svc.Privileged,
		Endpoints:       svc.Endpoints,
		Volumes:         svc.Volumes,
		Snapshot:        svc.Snapshot,
		RAMCommitment:   svc.RAMCommitment,
		CPUCommitment:   svc.CPUCommitment,
		HealthChecks:    svc.HealthChecks,
		MemoryLimit:     svc.MemoryLimit,
		CPUShares:       svc.CPUShares,
		ParentServiceID: svc.ParentServiceID,
		Hostname:        svc.Hostname,
	}
}

// RemoveService deletes a service if the service has no running states
func RemoveService(conn client.Connection, poolID, serviceID string) error {
	basepth := ""
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}
	pth := path.Join(basepth, "/services", serviceID)

	logger := plog.WithFields(log.Fields{
		"serviceid": serviceID,
		"zkpath":    pth,
	})

	// clean any bad service states before checking children
	if err := CleanServiceStates(conn, poolID, serviceID); err != nil {
		return err
	}

	// if the service has any children, do not delete
	if ch, err := conn.Children(pth); err != nil {

		logger.WithError(err).Debug("Could not look up children of service")
		return &ServiceError{
			Action:    "delete",
			ServiceID: serviceID,
			Message:   "could not look up children of service",
		}
	} else if count := len(ch); count > 0 {

		logger.WithField("statecount", count).Debug("Cannot delete a service with running instances")
		return &ServiceError{
			Action:    "delete",
			ServiceID: serviceID,
			Message:   fmt.Sprintf("found %d running instances", count),
		}
	}

	if err := conn.Delete(pth); err != nil {

		logger.WithError(err).Debug("Could not delete service entry from zookeeper")
		return &ServiceError{
			Action:    "delete",
			ServiceID: serviceID,
			Message:   "could not delete service",
		}
	}

	logger.Debug("Deleted service entry from zookeeper")
	return nil
}

// SyncServices synchronizes the services to the provided list (uses a pool-
// based connection)
func SyncServices(conn client.Connection, svcs []service.Service) error {
	pth := path.Join("/services")

	logger := plog.WithField("zkpath", pth)

	// look up children service ids
	ch, err := conn.Children(pth)
	if err != nil && err != client.ErrNoNode {
		logger.WithError(err).Debug("Could not look up services")

		// TODO: wrap error?
		return err
	}

	// store the service ids in a hash map
	chmap := make(map[string]struct{})
	for _, serviceid := range ch {
		chmap[serviceid] = struct{}{}
	}

	// set the services
	for _, s := range svcs {
		if err := UpdateService(conn, s, false, false); err != nil {
			return err
		}

		// delete matching records
		if _, ok := chmap[s.ID]; ok {
			delete(chmap, s.ID)
		}
	}

	// remove any leftovers
	for serviceid := range chmap {
		if err := conn.Delete(path.Join(pth, serviceid)); err != nil {

			logger.WithField("serviceid", serviceid).WithError(err).Debug("Could not delete service entry from zookeeper")
			return &ServiceError{
				Action:    "sync",
				ServiceID: serviceid,
				Message:   "could not delete service",
			}
		}
	}

	return nil
}

// WaitService waits for all of a service's instances to satisfy a particular
// state.
func WaitService(cancel <-chan struct{}, conn client.Connection, poolID, serviceID string, checkCount func(count int) bool, checkState func(s *State, exists bool) bool) error {
	basepth := ""
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}
	pth := path.Join(basepth, "/services", serviceID)

	logger := plog.WithFields(log.Fields{
		"serviceid": serviceID,
		"zkpath":    pth,
	})

	done := make(chan struct{})
	defer func() { close(done) }()
	for {

		// clean any bad service states
		if err := CleanServiceStates(conn, poolID, serviceID); err != nil {
			return err
		}

		// get the list of states
		ch, ev, err := conn.ChildrenW(pth, done)
		if err != nil {

			logger.WithError(err).Debug("Could not watch states for service")

			// TODO: wrap error?
			return err
		}

		// make sure the count satisfies the requirements
		if checkCount(len(ch)) {
			ok := true

			for _, stateID := range ch {
				st8log := logger.WithField("stateid", stateID)

				// parse the state id and set up the request
				hostID, _, instanceID, err := ParseStateID(stateID)
				if err != nil {

					// This should never happen, but handle it
					st8log.WithError(err).Error("Invalid state id while monitoring service")
					return err
				}

				req := StateRequest{
					PoolID:     poolID,
					HostID:     hostID,
					ServiceID:  serviceID,
					InstanceID: instanceID,
				}

				// wait for the state to satisfy the requirements
				if _, err := MonitorState(cancel, conn, req, checkState); err != nil {
					st8log.WithError(err).Debug("Stopped monitoring state")

					// return if cancel was triggered
					select {
					case <-cancel:
						return nil
					default:
					}

					ok = false
					break
				}
			}

			// if all of the requirements are satisfied, return
			if ok {
				return nil
			}
		}

		// otherwise, wait for the number of states to change
		select {
		case <-ev:
		case <-cancel:
			return nil
		}
		close(done)
		done = make(chan struct{})
	}
}
