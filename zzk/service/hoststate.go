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

	// ResumeContainer resumes a paused container.  Returns nil if the
	// container has stopped or if it doesn't exist.
	ResumeContainer(serviceID string, instanceID int) error

	// PauseContainer pauses a running container.  Returns nil if the container
	// has stopped or if it doesn't exist.
	PauseContainer(serviceID string, instanceID int) error
}

type hostStateThread struct {
	data   *ServiceState
	exited <-chan time.Time
}

// HostStateListener is the listener for monitoring service instances
type HostStateListener struct {
	conn     client.Connection
	handler  HostStateHandler
	hostID   string
	shutdown <-chan interface{}
	mu       *sync.RWMutex
	threads  map[string]hostStateThread
}

// NewHostStateListener instantiates a HostStateListener object
func NewHostStateListener(handler HostStateHandler, hostID string, shutdown <-chan interface{}) *HostStateListener {
	l := &HostStateListener{
		handler:  handler,
		hostID:   hostID,
		shutdown: shutdown,
		mu:       &sync.RWMutex{},
		threads:  make(map[string]hostStateThread),
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
	stateIDs := l.getExistingStateIDs()
	for _, s := range stateIDs {
		if _, ok := p[s]; !ok {
			l.cleanUpContainer(s)
		}
	}
}

// Spawn listens for changes in the host state and manages running instances
func (l *HostStateListener) Spawn(cancel <-chan interface{}, stateID string) {
	logger := plog.WithFields(log.Fields{
		"hostID":  l.hostID,
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
	ssdat, containerExit := l.getExistingState(stateID)
	// TODO: Should we go ahead and pull the service state node from zk and make sure it matches?

	sspth := path.Join("/services", serviceID, stateID)
	// load the service state node
	if ssdat == nil {
		ssdat = &ServiceState{}
		if err := l.conn.Get(sspth, ssdat); err == client.ErrNoNode {
			// TODO: mismatched data so shut down container and clean up nodes
			//  Unless we address the TODO above, there is no container running here
			return
		} else if err != nil {
			logger.WithError(err).Error("Could not load service state")
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
			l.cleanUpContainer(stateID)
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
			l.cleanUpContainer(stateID)
			return
		}

		// attach to the container if not already attached
		if containerExit == nil {
			containerExit, err = l.handler.AttachContainer(ssdat, serviceID, instanceID)
			if err != nil {
				logger.WithError(err).Error("Could not attach to container")
				// TODO: there is a problem with docker here. How do we handle
				// this behavior?  For now just shut down and clean up nodes
				l.cleanUpContainer(stateID)
				return
			}
			l.setExistingState(stateID, ssdat, containerExit)
		}

		// set the state of this instance
		switch hsdat.DesiredState {
		case service.SVCRun:
			if containerExit == nil {
				// container is detached because it doesn't exist
				ssdat, containerExit, err = l.handler.StartContainer(l.shutdown, serviceID, instanceID)
				if err != nil {
					logger.WithError(err).Error("Could not start container")
					// TODO: there is a problem with docker here.  How do we
					// handle this behavior? For now just shut down and clean up nodes
					l.cleanUpContainer(stateID)
					return
				}

				// set the service state
				l.setExistingState(stateID, ssdat, containerExit)
				logger.Debug("Started container")

				if err := l.updateServiceStateInZK(ssdat, req); err != nil {
					logger.WithError(err).Error("Could not set state for started container")
					return
				}
			} else if ssdat.Paused {
				// resume paused container
				if err := l.handler.ResumeContainer(serviceID, instanceID); err != nil {
					logger.WithError(err).Error("Could not resume container")
					// TODO: there is a problem with docker here. How do we
					// handle this behavior? For now just shut down and clean up nodes
					l.cleanUpContainer(stateID)
					return
				}

				// update the service state
				ssdat.Paused = false
				l.setExistingState(stateID, ssdat, containerExit)

				if err := l.updateServiceStateInZK(ssdat, req); err != nil {
					logger.WithError(err).Error("Could not set state for resumed container")
					return
				}

				logger.Debug("Resumed paused container")
			}
		case service.SVCPause:
			if containerExit != nil && !ssdat.Paused {
				// container is attached and not paused, so pause the container
				if err := l.handler.PauseContainer(serviceID, instanceID); err != nil {
					logger.WithError(err).Error("Could not pause container")
					// TODO: there is a problem with docker here.  How do we
					// handle this behavior? For now just shut down and clean up nodes
					l.cleanUpContainer(stateID)
					return
				}

				// update the service state
				ssdat.Paused = true
				l.setExistingState(stateID, ssdat, containerExit)
				if err := l.updateServiceStateInZK(ssdat, req); err != nil {
					logger.WithError(err).Error("Could not set state for resumed container")
					return
				}

				logger.Debug("Paused running container")
			}
		case service.SVCStop:
			// shut down the container and clean up nodes
			l.cleanUpContainer(stateID)
			return
		default:
			logger.Debug("Could not process desired state for instance")
		}
		logger.Debug("Waiting for event on host state")
		select {
		case <-hsevt:
		case <-ssevt:
		case timeExit := <-containerExit:
			// set the service state
			ssdat.Terminated = timeExit
			containerExit = nil
			l.setExistingState(stateID, ssdat, containerExit)
			logger.WithField("terminated", timeExit).Warn("Container exited unexpectedly, restarting")

			if err := l.updateServiceStateInZK(ssdat, req); err != nil {
				logger.WithError(err).Error("Could not set state for stopped container")
				// TODO: we currently don't support containers restarting if
				// shut down during an outage, so don't bother
				return
			}
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

func (l *HostStateListener) getExistingStateIDs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	stateIds := make([]string, len(l.threads))
	i := 0
	for s := range l.threads {
		stateIds[i] = s
		i++
	}
	return stateIds
}

func (l *HostStateListener) getExistingState(stateID string) (*ServiceState, <-chan time.Time) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if thread, ok := l.threads[stateID]; ok {
		return thread.data, thread.exited
	}
	return nil, nil
}

func (l *HostStateListener) setExistingState(stateID string, data *ServiceState, containerExit <-chan time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.threads[stateID] = hostStateThread{data, containerExit}
}

func (l *HostStateListener) removeExistingState(stateID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.threads, stateID)
}

func (l *HostStateListener) updateServiceStateInZK(data *ServiceState, req StateRequest) error {
	return UpdateState(l.conn, req, func(s *State) bool {
		s.ServiceState = *data
		return true
	})
}

func (l *HostStateListener) cleanUpContainer(stateID string) {
	logger := plog.WithFields(log.Fields{
		"hostID":  l.hostID,
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

	// Get the containerExit channel from our thread map
	_, containerExit := l.getExistingState(stateID)

	// stop the container
	if err := l.handler.StopContainer(serviceID, instanceID); err != nil {
		logger.WithError(err).Error("Could not stop container")
	} else if containerExit != nil {
		// wait for the container to exit
		time := <-containerExit
		logger.WithField("terminated", time).Debug("Container exited")
	}

	// Remove the container from our thread map
	l.removeExistingState(stateID)

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
	for stateIDs := l.getExistingStateIDs(); len(stateIDs) > 0; stateIDs = l.getExistingStateIDs() {
		wg := sync.WaitGroup{}
		for _, s := range stateIDs {
			wg.Add(1)
			go func(sid string) {
				defer wg.Done()
				l.cleanUpContainer(sid)
			}(s)
		}
		wg.Wait()
	}
}
