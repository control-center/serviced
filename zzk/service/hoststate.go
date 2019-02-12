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

	"github.com/control-center/serviced/coordinator/client"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/domain/service"
)

// HostStateHandler is the handler for running the HostListener
type HostStateHandler interface {

	// StopsContainer stops the container if the container exists and isn't
	// already stopped.
	StopContainer(serviceID string, instanceID int) error

	// AttachContainer attaches to an existing container for the service
	// instance. Returns nil channel if the container id doesn't match or if
	// the container has stopped. Channel reports the time that the container
	// has stopped.
	AttachContainer(state *ServiceState, serviceID string, instanceID int) (<-chan time.Time, error)

	// StartContainer creates and starts a new container for the given service
	// instance.  It returns relevant information about the container and a
	// channel that triggers when the container has stopped.
	StartContainer(cancel <-chan interface{}, serviceID string, instanceID int) (*ServiceState, <-chan time.Time, error)

	// RestartContainer asynchronously prepulls the latest image before
	// stopping the container.  It only returns an error if there is a problem
	// with docker and not if the container is not running or doesn't exist.
	RestartContainer(cancel <-chan interface{}, serviceID string, instanceID int) error

	// ResumeContainer resumes a paused container.  Returns nil if the
	// container has stopped or if it doesn't exist.
	ResumeContainer(serviceID string, instanceID int) error

	// PauseContainer pauses a running container.  Returns nil if the container
	// has stopped or if it doesn't exist.
	PauseContainer(serviceID string, instanceID int) error
}

// HostStateListener is the listener for monitoring service instances
type HostStateListener struct {
	conn     client.Connection
	handler  HostStateHandler
	hostID   string
	shutdown <-chan interface{}
	mu       *sync.RWMutex
	threads  map[string]struct {
		data   *ServiceState
		exited <-chan time.Time
	}
	shutdowncomplete chan interface{}
}

// NewHostStateListener instantiates a HostStateListener object
func NewHostStateListener(handler HostStateHandler, hostID string, shutdown <-chan interface{}) *HostStateListener {
	l := &HostStateListener{
		handler:  handler,
		hostID:   hostID,
		shutdown: shutdown,
		mu:       &sync.RWMutex{},
		threads: make(map[string]struct {
			data   *ServiceState
			exited <-chan time.Time
		}),
		shutdowncomplete: make(chan interface{}),
	}
	go l.watchForShutdown()
	return l
}

// GetConnection implements zzk.Listener
func (l *HostStateListener) SetConnection(conn client.Connection) { l.conn = conn }

// GetPath implements zzk.Listener
func (l *HostStateListener) GetPath(nodes ...string) string {
	parts := append([]string{"/hosts", l.hostID, "instances"}, nodes...)
	return path.Join(parts...)
}

// Ready implements zzk.Listener
func (l *HostStateListener) Ready() error {
	return nil
}

// Done implements zzk.Listener
func (l *HostStateListener) Done() {
}

// PostProcess implements zzk.Listener
// This is always called after all threads have been spawned
func (l *HostStateListener) PostProcess(p map[string]struct{}) {
	// We are running all of the containers we are supposed to, now
	// shut down any containers we are not supposed to be running
	l.mu.Lock()
	defer l.mu.Unlock()
	stateIDs := l.getExistingThreadStateIDs()
	var orphanedStates []string
	for _, s := range stateIDs {
		if _, ok := p[s]; !ok {
			orphanedStates = append(orphanedStates, s)
			plog.WithField("stateid", s).Info("Detected orphaned container")
		}
	}

	if len(orphanedStates) > 0 {
		l.cleanUpContainers(orphanedStates, false)
		plog.WithField("count", len(orphanedStates)).Info("Cleaned up all orphaned containers")
	}
}

// Spawn listens for changes in the host state and manages running instances
func (l *HostStateListener) Spawn(cancel <-chan interface{}, stateID string) {
	logger := plog.WithFields(log.Fields{
		"hostid":  l.hostID,
		"stateid": stateID,
	})

	// If we are shutting down, just return
	select {
	case <-l.shutdown:
		logger.Debug("Will not spawn, shutting down")
		return
	default:
	}

	// check if the state id is valid
	hostID, serviceID, instanceID, err := ParseStateID(stateID)
	if err != nil || hostID != l.hostID {

		logger.WithField("hostidmatch", hostID == l.hostID).WithError(err).Warn("Invalid state id, deleting")

		// clean up the bad node
		if err := l.conn.Delete(l.GetPath(stateID)); err != nil && err != client.ErrNoNode {
			logger.WithError(err).Error("Could not delete host state")
		}
		return
	}

	logger = logger.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	// set up the request object for updates
	req := StateRequest{
		PoolID:     "",
		HostID:     hostID,
		ServiceID:  serviceID,
		InstanceID: instanceID,
	}

	// reattach to orphaned container
	l.mu.RLock()
	ssdat, containerExit := l.getExistingThread(stateID)
	l.mu.RUnlock()

	sspth := path.Join("/services", serviceID, stateID)
	// load the service state node
	if ssdat == nil {
		ssdat = &ServiceState{}
		if err := l.conn.Get(sspth, ssdat); err == client.ErrNoNode {
			l.cleanUpContainers([]string{stateID}, true)
			return
		} else if err != nil {
			logger.WithError(err).Error("Could not load service state")
			return
		}
	} else {
		if err := l.updateServiceStateInZK(ssdat, req); err != nil {
			logger.WithError(err).Error("Could not set state for container")
			return
		}
	}

	done := make(chan struct{})
	defer func() { close(done) }()

	for {
		// set up a listener on the host state node
		hspth := l.GetPath(stateID)
		hsdat := &HostState{}
		hsevt, err := l.conn.GetW(hspth, hsdat, done)
		if err == client.ErrNoNode {
			logger.Debug("Host state was removed, exiting")
			l.cleanUpContainers([]string{stateID}, true)
			return
		} else if err != nil {
			logger.WithError(err).Error("Could not watch host state")
			return
		}

		// set up a listener on the service state node, to ensure the node's existence
		ok, ssevt, err := l.conn.ExistsW(sspth, done)
		if err != nil {
			logger.WithError(err).Error("Could not watch service state")
			return
		} else if !ok {
			logger.Debug("Service state was removed, exiting")
			l.cleanUpContainers([]string{stateID}, true)
			return
		}

		// set the state of this instance
		containerExit, ssdat, ok = l.setInstanceState(containerExit, ssdat, hsdat, stateID, serviceID, instanceID, req, logger)
		if !ok {
			return
		}

		var ipevt <-chan client.Event
		if ssdat.AssignedIP != "" && !ssdat.Static {
			req := IPRequest{
				IPAddress: ssdat.AssignedIP,
				HostID:    l.hostID,
			}

			ok, ipevt, err = l.conn.ExistsW(path.Join("/hosts", l.hostID, "ips", req.IPID()), done)
			if err != nil {
				logger.WithError(err).Error("Could not monitor ip")
				return
			}

			if !ok {
				logger.Debug("IP assignment was removed, exiting")
				l.cleanUpContainers([]string{stateID}, true)
				return
			}
		}

		logger.Debug("Waiting for event on host state")
		select {
		case hostStateEvent := <-hsevt:
			if hostStateEvent.Err != nil {
				logger.Infof("Host state listener received signal from host state connection: %s", hostStateEvent.Err.Error())
			}
		case serviceStateEvent := <-ssevt:
			if serviceStateEvent.Err != nil {
				logger.WithField("service", serviceID).Infof("Host state listener received signal from service state connection: %s", serviceStateEvent.Err.Error())
			}
		case ipEvent := <-ipevt:
			ireq := IPRequest{
				IPAddress: ssdat.AssignedIP,
				HostID:    l.hostID,
			}
			if ipEvent.Err != nil {
				logger.WithField("ip", ireq.IPID()).Infof("Host state listener received signal from host ip connection: %s", ipEvent.Err.Error())
			}
		case timeExit := <-containerExit:
			ssdat, ok = l.handleContainerExit(timeExit, ssdat, stateID, req, logger)
			if !ok {
				return
			}
			containerExit = nil
		case <-cancel:
			logger.Debug("Host state listener received signal to cancel listening")
			return
		case <-l.shutdown:
			logger.Debug("Host state listener received signal to shutdown")
			// Container shutdown will be handled by the HostStateListener for all containers during shutdown
			return
		}

		close(done)
		done = make(chan struct{})
	}
}

func (l *HostStateListener) setInstanceState(containerExit <-chan time.Time, ssdat *ServiceState, hsdat *HostState,
	stateID, serviceID string, instanceID int, req StateRequest, logger *log.Entry) (<-chan time.Time, *ServiceState, bool) {

	var err error

	// attach to the container if not already attached
	if containerExit == nil {
		containerExit, err = l.handler.AttachContainer(ssdat, serviceID, instanceID)
		if err != nil {
			logger.WithError(err).Error("Could not attach to container")
			l.cleanUpContainers([]string{stateID}, true)
			return nil, nil, false
		}

		if !l.setExistingThreadOrShutdown(stateID, ssdat, containerExit) {
			return nil, nil, false
		}
	}

	switch hsdat.DesiredState {
	case service.SVCRun:
		if containerExit == nil {
			// container is detached because it doesn't exist
			ssdat, containerExit, err = l.handler.StartContainer(l.shutdown, serviceID, instanceID)
			if err != nil {
				logger.WithError(err).Error("Could not start container")
				l.cleanUpContainers([]string{stateID}, true)
				return nil, nil, false
			}

			// set the service state
			if !l.setExistingThreadOrShutdown(stateID, ssdat, containerExit) {
				return nil, nil, false
			}

			logger.Debug("Started container")

			if err := l.updateServiceStateInZK(ssdat, req); err != nil {
				logger.WithError(err).Error("Could not set state for started container")
				return nil, nil, false
			}

		} else if ssdat.Paused {
			// resume paused container
			if err := l.handler.ResumeContainer(serviceID, instanceID); err != nil {
				logger.WithError(err).Error("Could not resume container")
				l.cleanUpContainers([]string{stateID}, true)
				return nil, nil, false
			}

			// update the service state
			ssdat.Paused = false
			if !l.setExistingThreadOrShutdown(stateID, ssdat, containerExit) {
				return nil, nil, false
			}

			if err := l.updateServiceStateInZK(ssdat, req); err != nil {
				logger.WithError(err).Error("Could not set state for resumed container")
				return nil, nil, false
			}

			logger.Debug("Resumed paused container")
		}
	case service.SVCRestart:
		// only try to restart once if the container hasn't already been
		// restarted.
		if ssdat.Restarted.Before(ssdat.Started) {
			// RestartContainer will asynchronously pull the image and stop the
			// service.  Once the container stops, we will receive a message on the
			// containerExit channel that the container has exited which will
			// trigger the container to start again.
			if err := l.handler.RestartContainer(l.shutdown, serviceID, instanceID); err != nil {
				logger.WithError(err).Error("Could not restart container, exiting")
				l.cleanUpContainers([]string{stateID}, true)
				return nil, nil, false
			}

			// update the service state
			ssdat.Restarted = time.Now()
			if !l.setExistingThreadOrShutdown(stateID, ssdat, containerExit) {
				return nil, nil, false
			}

			// set the host state
			if err := UpdateState(l.conn, req, func(s *State) bool {
				s.ServiceState = *ssdat
				if s.DesiredState == service.SVCRestart {
					s.DesiredState = service.SVCRun
				}
				return true
			}); err != nil {
				logger.WithError(err).Error("Could not set state for restarting container")
				return nil, nil, false
			}
			logger.Debug("Initiating container restart")
		} else {
			// restart has already been triggered, so restore the state back to
			// run
			if err := UpdateState(l.conn, req, func(s *State) bool {
				if s.DesiredState == service.SVCRestart {
					s.DesiredState = service.SVCRun
					return true
				}
				return false
			}); err != nil {
				logger.WithError(err).Error("Could not update desired state for restarting container")
				return nil, nil, false
			}
		}
	case service.SVCPause:
		if containerExit != nil && !ssdat.Paused {
			// container is attached and not paused, so pause the container
			if err := l.handler.PauseContainer(serviceID, instanceID); err != nil {
				logger.WithError(err).Error("Could not pause container")
				l.handler.ResumeContainer(serviceID, instanceID)
				select{}
				//l.cleanUpContainers([]string{stateID}, true)
				//return nil, nil, false
			}

			// update the service state
			ssdat.Paused = true
			if !l.setExistingThreadOrShutdown(stateID, ssdat, containerExit) {
				return nil, nil, false
			}

			if err := l.updateServiceStateInZK(ssdat, req); err != nil {
				logger.WithError(err).Error("Could not set state for resumed container")
				return nil, nil, false
			}

			logger.Debug("Paused running container")
		}
	case service.SVCStop:
		// shut down the container and clean up nodes
		l.cleanUpContainers([]string{stateID}, true)
		return nil, nil, false
	default:
		logger.Debug("Could not process desired state for instance")
	}
	return containerExit, ssdat, true
}

func (l *HostStateListener) handleContainerExit(timeExit time.Time, ssdat *ServiceState, stateID string,
	req StateRequest, logger *log.Entry) (*ServiceState, bool) {

	l.mu.Lock()
	defer l.mu.Unlock()

	// Don't do anything if we are shutting down, the shutdown cleanup will handle it
	select {
	case <-l.shutdown:
		return nil, false
	default:
	}

	// set the service state
	ssdat.Terminated = timeExit
	l.setExistingThread(stateID, ssdat, nil)
	logger.WithField("terminated", timeExit).Warn("Container exited, restarting")

	if err := l.updateServiceStateInZK(ssdat, req); err != nil {
		logger.WithError(err).Error("Could not set state for stopped container")
		// TODO: we currently don't support containers restarting if
		// shut down during an outage, so don't bother
		return nil, false
	}
	return ssdat, true
}

// Gets a list of state IDs for all existing threads
//  Call l.mu.RLock() first
func (l *HostStateListener) getExistingThreadStateIDs() []string {
	stateIds := make([]string, len(l.threads))
	i := 0
	for s := range l.threads {
		stateIds[i] = s
		i++
	}
	return stateIds
}

// Gets the ServiceState for an existing thread
//  Call l.mu.RLock() first
func (l *HostStateListener) getExistingThread(stateID string) (*ServiceState, <-chan time.Time) {
	if thread, ok := l.threads[stateID]; ok {
		return thread.data, thread.exited
	}
	return nil, nil
}

// Adds a state to the internal thread list.
//  Call l.mu.Lock() first
func (l *HostStateListener) setExistingThread(stateID string, data *ServiceState, containerExit <-chan time.Time) {
	l.threads[stateID] = struct {
		data   *ServiceState
		exited <-chan time.Time
	}{data, containerExit}
}

// If we are NOT shutting down, adds a state to the internal thread list.
//  Returns true if the state was set, false if not (i.e. we are shutting down)
//  Acquires a lock, do NOT call l.mu.Lock() first
func (l *HostStateListener) setExistingThreadOrShutdown(stateID string, data *ServiceState, containerExit <-chan time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	select {
	case <-l.shutdown:
		return false
	default:
		l.setExistingThread(stateID, data, containerExit)
	}

	return true
}

// Removes a state from the internal thread list
//  Call l.mu.Lock() first
func (l *HostStateListener) removeExistingThread(stateID string) {
	delete(l.threads, stateID)
}

func (l *HostStateListener) updateServiceStateInZK(data *ServiceState, req StateRequest) error {
	return UpdateState(l.conn, req, func(s *State) bool {
		s.ServiceState = *data
		return true
	})
}

// Stops the running containers, cleans up zk nodes, and removes the threads from the thread list
//  Blocks until all containers are stopped
//  Call l.mu.Lock() first OR pass getLock=true
func (l *HostStateListener) cleanUpContainers(stateIDs []string, getLock bool) {
	// Start shutting down all of the containers in parallel
	wg := &sync.WaitGroup{}
	for _, s := range stateIDs {
		containerExit := func() <-chan time.Time {
			if getLock {
				l.mu.RLock()
				defer l.mu.RUnlock()
			}
			_, cExit := l.getExistingThread(s)
			return cExit
		}()
		wg.Add(1)
		go func(stateID string, cExit <-chan time.Time) {
			defer wg.Done()
			l.shutDownContainer(stateID, cExit)
		}(s, containerExit)
	}

	// Remove the threads from our internal thread list
	// Need to get the lock here if we don't already have it
	func() {
		if getLock {
			l.mu.Lock()
			defer l.mu.Unlock()
		}
		for _, s := range stateIDs {
			l.removeExistingThread(s)
		}
	}()

	// Wait for all containers to shut down
	wg.Wait()
}

// Shuts down a running container and removes the state from zookeeper
//  Blocks until the container is stopped
//  Does NOT require a lock.  Does NOT remove the thread from the internal thread list
func (l *HostStateListener) shutDownContainer(stateID string, containerExit <-chan time.Time) {
	logger := plog.WithFields(log.Fields{
		"hostid":  l.hostID,
		"stateid": stateID,
	})

	// Parse the stateID
	hostID, serviceID, instanceID, err := ParseStateID(stateID)
	if err != nil || hostID != l.hostID {
		logger.WithField("hostidmatch", hostID == l.hostID).WithError(err).Warn("Could not clean up container: Invalid state id")
		return
	}

	logger = logger.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	// stop the container
	if err := l.handler.StopContainer(serviceID, instanceID); err != nil {
		logger.WithError(err).Error("Could not stop container")
	} else if containerExit != nil {
		// wait for the container to exit
		time := <-containerExit
		logger.WithField("terminated", time).Debug("Container exited")
	}

	// delete the state from the coordinator
	req := StateRequest{
		PoolID:     "",
		HostID:     hostID,
		ServiceID:  serviceID,
		InstanceID: instanceID,
	}
	if err := DeleteState(l.conn, req); err != nil {
		logger.WithError(err).Warn("Could not delete state for stopped container")
		return
	}
}

func (l *HostStateListener) watchForShutdown() {
	<-l.shutdown
	plog.Info("Received shutdown")

	l.mu.Lock()
	defer l.mu.Unlock()

	stateIDs := l.getExistingThreadStateIDs()
	l.cleanUpContainers(stateIDs, false)
	close(l.shutdowncomplete)
}

// Used by tests, returns a channel that will be closed when shutdown is complete
func (l *HostStateListener) GetShutdownComplete() <-chan interface{} {
	return l.shutdowncomplete
}
