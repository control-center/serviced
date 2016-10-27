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

// HostStateListener is the listener for monitoring service instances
type HostStateListener struct {
	conn    client.Connection
	handler HostStateHandler
	hostID  string
}

// NewHostListener instantiates a HostListener object
func NewHostStateListener(handler HostStateHandler, hostID string) *HostStateListener {
	return &HostStateListener{
		handler: handler,
		hostID:  hostID,
	}
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

// Done removes the ephemeral node from the host registry
func (l *HostStateListener) Done() {
}

// PostProcess implements zzk.Listener
func (l *HostStateListener) PostProcess(p map[string]struct{}) {}

// Spawn listens for changes in the host state and manages running instances
func (l *HostStateListener) Spawn(shutdown <-chan interface{}, stateID string) {
	logger := plog.WithFields(log.Fields{
		"hostid":  l.hostID,
		"stateid": stateID,
	})

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

	var containerExit <-chan time.Time
	defer func() {

		// stop the container
		if err := l.handler.StopContainer(serviceID, instanceID); err != nil {
			logger.WithError(err).Error("Could not stop container")
		} else if containerExit != nil {
			// wait for the container to exit
			time := <-containerExit
			logger.WithField("terminated", time).Debug("Container exited")
		}

		// delete the state from the coordinator
		if err := DeleteState(l.conn, req); err != nil {
			logger.WithError(err).Warn("Could not delete state")
		}
	}()

	// load the service state node
	sspth := path.Join("/services", serviceID, stateID)
	ssdat := &ServiceState{}
	if err := l.conn.Get(sspth, ssdat); err == client.ErrNoNode {
		return
	} else if err != nil {
		logger.WithError(err).Error("Could not load service state")
		return
	}

	done := make(chan struct{})
	defer func() {
		close(done)
	}()

	for {
		// set up a listener on the host state node
		hspth := l.GetPath(stateID)
		hsdat := &HostState{}
		hsevt, err := l.conn.GetW(hspth, hsdat, done)
		if err == client.ErrNoNode {

			logger.Debug("Host state was removed, exiting")
			return
		} else if err != nil {

			logger.WithError(err).Error("Could not watch host state")
			return
		}

		// attach to the container if not already attached
		if containerExit == nil {
			containerExit, err = l.handler.AttachContainer(ssdat, serviceID, instanceID)
			if err != nil {

				logger.WithError(err).Error("Could not attach to container")
				return
			}
		}

		// set the state of this instance
		switch hsdat.DesiredState {
		case service.SVCRun:
			if containerExit == nil {

				// container is detached because it doesn't exist
				ssdat, containerExit, err = l.handler.StartContainer(shutdown, serviceID, instanceID)
				if err != nil {

					logger.WithError(err).Error("Could not start container")
					return
				}

				// set the service state in zookeeper
				if err := UpdateState(l.conn, req, func(s *State) bool {
					s.ServiceState = *ssdat
					return true
				}); err != nil {

					logger.WithError(err).Error("Could not set state for started container")
					return
				}

				logger.Debug("Started container")
			} else if ssdat.Paused {

				// resume paused container
				if err := l.handler.ResumeContainer(serviceID, instanceID); err != nil {

					logger.WithError(err).Error("Could not resume container")
					return
				}

				// set the service state in zookeeper
				if err := UpdateState(l.conn, req, func(s *State) bool {
					s.Paused = false
					*ssdat = s.ServiceState
					return true
				}); err != nil {

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
					return
				}

				// set the service state in zookeeper
				if err := UpdateState(l.conn, req, func(s *State) bool {
					s.Paused = true
					*ssdat = s.ServiceState
					return true
				}); err != nil {

					logger.WithError(err).Error("Could not set state for paused container")
					return
				}

				logger.Debug("Paused running container")
			}
		case service.SVCStop:

			logger.Debug("Stopping running container")
			return
		default:

			logger.Debug("Could not process desired state for instance")
		}

		select {
		case <-hsevt:
		case time := <-containerExit:
			logger.WithField("terminated", time).Warn("Container exited unexpectedly, restarting")
			containerExit = nil

			func() {
				t := time.NewTicker(time.Second)
				defer t.Stop()
				for {
					if err := UpdateState(l.conn, req, func(s *State) bool {
						s.Terminated = time
						*ssdat = s.ServiceState
						return true
					}); err == client.ErrNoServer {
						logger.WithError(err).Warn("Server not found, attempting to retry updating service")
						select {
						case <-t.C:
						case <-shutdown:
							logger.Debug("Host state listener received signal to shut down")
							return
						}
					} else if err != nil {
						logger.WithError(err).Error("Could not update state for stopped container")
						return
					} else {
						break
					}
				}
			}()
		case <-shutdown:
			logger.Debug("Host state listener received signal to shut down")
			return
		}

		close(done)
		done = make(chan struct{})
	}
}
