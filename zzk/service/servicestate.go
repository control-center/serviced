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
	"path"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
)

// ServiceHandler handles all non-zookeeper interactions required by the service
type ServiceHandler interface {
	SelectHost(*ServiceNode) (string, error)
}

// ServiceListener is the listener for /services
type ServiceListener struct {
	conn    client.Connection
	handler ServiceHandler
	poolid  string
	mu      *sync.Mutex
}

// NewServiceListener instantiates a new ServiceListener
func NewServiceListener(poolid string, handler ServiceHandler) *ServiceListener {
	return &ServiceListener{poolid: poolid, handler: handler, mu: &sync.Mutex{}}
}

// SetConnection implements zzk.Listener
func (l *ServiceListener) SetConnection(conn client.Connection) { l.conn = conn }

// GetPath implements zzk.Listener
func (l *ServiceListener) GetPath(nodes ...string) string {
	parts := append([]string{"/services"}, nodes...)
	if l.poolid != "" {
		parts = append([]string{"/pools", l.poolid}, parts...)
	}
	return path.Join(parts...)
}

// Ready implements zzk.Listener
func (l *ServiceListener) Ready() (err error) { return }

// Done implements zzk.Listener
func (l *ServiceListener) Done() { return }

// PostProcess implements zzk.Listener
func (l *ServiceListener) PostProcess(p map[string]struct{}) {}

// Spawn watches a service and syncs the number of running instances
func (l *ServiceListener) Spawn(shutdown <-chan interface{}, serviceID string) {
	logger := plog.WithField("serviceid", serviceID)

	// set up the retry timer
	timer := time.NewTimer(0)
	timer.Stop()
	defer timer.Stop()

	// set up cancellable on zookeeper watches
	done := make(chan struct{})
	defer func() { close(done) }()

	for {
		// set up a watch on the service
		sDat, err := NewServiceNodeFromService(&service.Service{})
		if err != nil {
			logger.WithError(err).Debug("Could not create service node from service")
			return
		}
		sEvt, err := l.conn.GetW(l.GetPath(serviceID), sDat, done)
		if err == client.ErrNoNode {

			logger.Debug("Service deleted, listener shutting down")
			return
		} else if err != nil {

			logger.WithError(err).Error("Could not load service")
			return
		}

		logger = logger.WithFields(log.Fields{
			"servicename":      sDat.Name,
			"desiredstate":     sDat.DesiredState,
			"desiredinstances": sDat.Instances,
		})

		// clean invalid states and set up a watch on the service's states
		if err := CleanServiceStates(l.conn, l.poolid, serviceID); err != nil {

			logger.WithError(err).Error("Could not clean up invalid states on service")
			return
		}
		stateIDs, ssEvt, err := l.conn.ChildrenW(l.GetPath(serviceID), done)
		if err == client.ErrNoNode {

			logger.Debug("Service deleted, listener shutting down")
			return
		} else if err != nil {

			logger.WithError(err).Error("Could not look up states for service")
			return
		}
		reqs, ok := l.getStateRequests(logger, stateIDs)
		if !ok {
			timer.Reset(time.Second)
		}

		// set the state of these services
		dstate := service.DesiredState(sDat.DesiredState)
		switch dstate {
		case service.SVCRun:

			// Resume paused service states
			if _, ok := l.Resume(reqs); !ok {
				timer.Reset(time.Second)
			}

			// Synchronize the number of service states
			if _, ok := l.Sync(sDat.Locked, sDat, reqs); !ok {
				timer.Reset(time.Second)
			}

		case service.SVCPause:

			// Pause running service states
			if _, ok := l.Pause(reqs); !ok {
				timer.Reset(time.Second)
			}

		case service.SVCStop:

			// Stop all service states
			if _, ok := l.Stop(reqs); !ok {
				timer.Reset(time.Second)
			}

		default:
			logger.Warn("Could not process desired state for service")
		}

		logger.Debug("Waiting for service event")
		select {
		case <-sEvt:
		case <-ssEvt:
		case <-timer.C:
		case <-shutdown:

			logger.Debug("Service listener receieved signal to shut down")
			return
		}

		close(done)
		done = make(chan struct{})
		timer.Stop()
	}
}

// getStateRequests returns a list of state requests
func (l *ServiceListener) getStateRequests(logger *log.Entry, stateIDs []string) ([]StateRequest, bool) {

	reqs := make([]StateRequest, len(stateIDs))
	ok := true

	for i, stateID := range stateIDs {
		hostID, serviceID, instanceID, err := ParseStateID(stateID)
		if err != nil {

			// This shouldn't happen
			logger.WithField("stateid", stateID).WithError(err).Warn("Unexpected error trying to parse state id")
			ok = false
			continue
		}

		reqs[i] = StateRequest{
			PoolID:     l.poolid,
			HostID:     hostID,
			ServiceID:  serviceID,
			InstanceID: instanceID,
		}
	}

	return reqs, ok
}

// Sync synchronizes the number of running service states and returns the delta
// of added (>0) or deleted (<0) instances.
func (l *ServiceListener) Sync(isLocked bool, sn *ServiceNode, reqs []StateRequest) (int, bool) {
	ok := true
	count := len(reqs)

	// If the service has a change option for restart all on changed, stop all
	// instances and wait for the nodes to stop.  Once all service instances
	// have been stopped (deleted), then go ahead and start the instances.
	if utils.StringInSlice("restartAllOnInstanceChanged", sn.ChangeOptions) {
		if count != 0 && count != sn.Instances {
			sn.Instances = 0 // NOTE: this will not update the node in zk or elastic
		}
	}

	// Do not create instances if service is locked
	if isLocked && count < sn.Instances {
		return 0, ok
	}

	// sort all of the requests by instance id
	SortStateRequests(reqs)

	// Start instances if there is a deficit
	delta := 0
	i := -1
	for ; count < sn.Instances; count++ {
		// Assuming that instanceIDs are gte 0 and that all instance IDs are
		// unique for a service, find the first index that does not match the
		// instanceID value.
		// Example: {0,1,4,5}, [0]=>0, [1]=>1, --> [2]=>4 <--
		for i = i + 1; i < count; i++ {
			if i < reqs[i].InstanceID {
				break
			}
		}

		// Create the instance if the service is not locked
		if !l.Start(sn, i) {
			return delta, false
		}
		delta++

		// Prepend a request object to the array. You could insert it in order,
		// but that is more expensive and doesn't matter.  What is important
		// is that the index of the mismatched value increases by one.
		// Example: Before: {0, 1, 4, 5}, [2]=>4 After: {2, 0, 1, 4, 5}, [3]=>4
		reqs = append(make([]StateRequest, 1), reqs...)

		// See https://play.golang.org/p/t8rd97z8nK if you need more proof
	}

	// Stop instances if there is a surplus
	if count > sn.Instances {
		delta, pass := l.Stop(reqs[sn.Instances:])
		return -delta, ok && pass
	}

	return delta, ok
}

// Start schedules a service instance
func (l *ServiceListener) Start(sn *ServiceNode, instanceID int) bool {

	mu.Lock()
	defer mu.Unlock()

	logger := plog.WithFields(log.Fields{
		"serviceid":                   sn.ID,
		"servicename":                 sn.Name,
		"shouldhaveaddressassignment": sn.ShouldHaveAddressAssignment,
		"instanceid":                  instanceID,
	})

	// pick a host
	hostID, err := l.handler.SelectHost(sn)
	if err != nil {

		logger.WithError(err).Warn("Could not select host")
		return false
	}

	logger.WithField("hostid", hostID).Debug("Service instance scheduled to host")
	req := StateRequest{
		PoolID:     l.poolid,
		HostID:     hostID,
		ServiceID:  sn.ID,
		InstanceID: instanceID,
	}

	// make sure the state exists on neither the service nor the host
	DeleteState(l.conn, req)

	if err := CreateState(l.conn, req); err != nil {

		logger.WithError(err).Warn("Could not schedule service instance")
		return false
	}

	return true
}

// Stop unschedules the list of service instances and returns the number of
// instances successfully stopped.
func (l *ServiceListener) Stop(reqs []StateRequest) (int, bool) {
	delta := 0
	ok := true

	for _, req := range reqs {

		logger := plog.WithFields(log.Fields{
			"hostid":     req.HostID,
			"serviceid":  req.ServiceID,
			"instanceid": req.InstanceID,
		})

		isOnline, err := IsHostOnline(l.conn, req.PoolID, req.HostID)

		if err != nil {

			logger.WithError(err).Warn("Could not verify whether host is online")
			ok = false
			continue
		}

		if isOnline {
			if err := UpdateState(l.conn, req, func(s *State) bool {
				if s.DesiredState != service.SVCStop {
					s.DesiredState = service.SVCStop
					return true
				}
				return false
			}); err != nil {

				logger.WithError(err).Warn("Could not stop service state")
				ok = false
				continue
			}

			logger.Debug("Set desired state to stop since host is online")

		} else {
			if err := DeleteState(l.conn, req); err != nil {

				logger.WithError(err).Warn("Could not remove service state")
				ok = false
				continue
			}

			logger.Debug("Removed state since host is offline")
		}
		delta++
	}

	return delta, ok
}

// Pause schedules the service instance as paused and returns the number of
// affected instances.
func (l *ServiceListener) Pause(reqs []StateRequest) (int, bool) {
	delta := 0
	ok := true

	for _, req := range reqs {

		logger := plog.WithFields(log.Fields{
			"hostid":     req.HostID,
			"serviceid":  req.ServiceID,
			"instanceid": req.InstanceID,
		})

		// ONLY pause services with a desired state of run
		if err := UpdateState(l.conn, req, func(s *State) bool {
			if s.DesiredState == service.SVCRun {
				s.DesiredState = service.SVCPause
				return true
			}
			return false
		}); err != nil {

			logger.WithError(err).Warn("Could not pause service state")
			ok = false
			continue
		}
		delta++
	}

	return delta, ok
}

// Resume schedules the service instance as run and returns the number of
// affected instances.
func (l *ServiceListener) Resume(reqs []StateRequest) (int, bool) {
	delta := 0
	ok := true

	for _, req := range reqs {

		logger := plog.WithFields(log.Fields{
			"hostid":     req.HostID,
			"serviceid":  req.ServiceID,
			"instanceid": req.InstanceID,
		})

		// ONLY resume services with a desired state of pause
		if err := UpdateState(l.conn, req, func(s *State) bool {
			if s.DesiredState == service.SVCPause {
				s.DesiredState = service.SVCRun
				return true
			}
			return false
		}); err != nil {

			logger.WithError(err).Warn("Could not resume service state")
			ok = false
			continue
		}
		delta++
	}

	return delta, ok
}
