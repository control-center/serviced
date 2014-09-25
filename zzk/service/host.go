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
	"fmt"
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/zenoss/glog"
)

const (
	zkHost = "/hosts"
)

func hostpath(nodes ...string) string {
	p := append([]string{zkHost}, nodes...)
	return path.Join(p...)
}

// HostState is the zookeeper node for storing service instance information
// per host
type HostState struct {
	HostID         string
	ServiceID      string
	ServiceStateID string
	DesiredState   int
	version        interface{}
}

// NewHostState instantiates a new HostState node for client.Node
func NewHostState(state *servicestate.ServiceState) *HostState {
	return &HostState{
		HostID:         state.HostID,
		ServiceID:      state.ServiceID,
		ServiceStateID: state.ID,
		DesiredState:   service.SVCRun,
	}
}

// Version inplements client.Node
func (node *HostState) Version() interface{} {
	return node.version
}

// SetVersion implements client.Node
func (node *HostState) SetVersion(version interface{}) {
	node.version = version
}

// HostHandler is the handler for running the HostListener
type HostStateHandler interface {
	GetHost(string) (*host.Host, error)
	AttachService(chan<- interface{}, *service.Service, *servicestate.ServiceState) error
	StartService(chan<- interface{}, *service.Service, *servicestate.ServiceState) error
	PauseService(*service.Service, *servicestate.ServiceState) error
	ResumeService(*service.Service, *servicestate.ServiceState) error
	StopService(*servicestate.ServiceState) error
}

// HostStateListener is the listener for monitoring service instances
type HostStateListener struct {
	conn     client.Connection
	handler  HostStateHandler
	hostID   string
	registry string
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
	return hostpath(append([]string{l.hostID}, nodes...)...)
}

// Ready adds an ephemeral node to the host registry
func (l *HostStateListener) Ready() error {
	host, err := l.handler.GetHost(l.hostID)
	if err != nil {
		return err
	} else if host == nil {
		return ErrHostInvalid
	}

	// Create an ephemeral node at /registry/host
	// What you would expect to see from epath is /registry/host/EHOSTID, but
	// CreateEphemeral returns the full path from the root.  Since these are
	// pool-based connections, the path from the root is actually
	// /pools/POOLID/registry/host/EHOSTID
	epath, err := l.conn.CreateEphemeral(hostregpath(l.hostID), &HostNode{Host: host})
	if err != nil {
		return err
	}

	// Parse the ephemeral path to get the relative path from the connection
	// base. In other words, get the base (EHOSTID) and set the path starting
	// from /registry/host, instead of from /pools/POOLID/.../EHOSTID
	l.registry = hostregpath(path.Base(epath))
	return nil
}

// Done removes the ephemeral node from the host registry
func (l *HostStateListener) Done() {
	if err := l.conn.Delete(l.registry); err != nil {
		glog.Warningf("Could not unregister host %s: %s", l.hostID, err)
	}
}

// PostProcess implements zzk.Listener
func (l *HostStateListener) PostProcess(p map[string]struct{}) {}

// Spawn listens for changes in the host state and manages running instances
func (l *HostStateListener) Spawn(shutdown <-chan interface{}, stateID string) {
	var (
		processDone <-chan interface{}
		state       *servicestate.ServiceState
	)

	hpath := l.GetPath(stateID)

	defer func() {
		if state != nil {
			glog.V(0).Infof("Stopping service instance: %s", state.ID)
			l.stopInstance(processDone, state)
		} else if err := l.conn.Delete(hpath); err != nil {
			glog.Errorf("Could not delete state %s for host %s: %s", stateID, l.hostID, err)
		}
	}()

	for {
		var hs HostState
		event, err := l.conn.GetW(hpath, &hs)
		if err != nil {
			glog.Errorf("Could not load host instance %s: %s", stateID, err)
			return
		}

		if hs.ServiceID == "" || hs.ServiceStateID == "" {
			glog.Error("Invalid host state instance: ", hpath)
			return
		}

		var s servicestate.ServiceState
		if err := l.conn.Get(servicepath(hs.ServiceID, hs.ServiceStateID), &ServiceStateNode{ServiceState: &s}); err != nil {
			glog.Error("Could not find service instance: ", hs.ServiceStateID)
			return
		}
		state = &s

		var svc service.Service
		if err := l.conn.Get(servicepath(hs.ServiceID), &ServiceNode{Service: &svc}); err != nil {
			glog.Error("Could not find service: ", hs.ServiceID)
			return
		}

		glog.V(2).Infof("Processing %s (%s); Desired State: %d", svc.Name, svc.ID, hs.DesiredState)
		switch hs.DesiredState {
		case service.SVCRun:
			var err error
			if !state.IsRunning() {
				// process has stopped
				processDone, err = l.startInstance(&svc, state)
			} else if state.IsPaused() {
				// process has paused
				err = l.resumeInstance(&svc, state)
			} else if processDone == nil {
				// process is running superficially
				processDone, err = l.attachInstance(&svc, state)
			}
			if err != nil {
				glog.Errorf("Error trying to start or attach to service instance %s: %s", state.ID, err)
				return
			}
		case service.SVCPause:
			if state.IsPaused() {
				// service instance is not running, pass
			} else if err := l.pauseInstance(&svc, state); err != nil {
				glog.Errorf("Could not pause service instance %s; stopping: %s", state.ID, err)
				return
			}
		case service.SVCStop:
			return
		default:
			glog.V(2).Infof("Unhandled service %s (%s)", svc.Name, svc.ID)
		}

		select {
		case <-processDone:
			glog.V(2).Infof("Process ended for instance: ", hs.ServiceStateID)
		case e := <-event:
			glog.V(3).Info("Receieved event: ", e)
			if e.Type == client.EventNodeDeleted {
				return
			}
		case <-shutdown:
			glog.V(2).Infof("Service %s Host instance %s receieved signal to shutdown", hs.ServiceID, hs.ServiceStateID)
			return
		}
	}
}

func (l *HostStateListener) startInstance(svc *service.Service, state *servicestate.ServiceState) (<-chan interface{}, error) {
	// This container may not get cleaned up if the time to start exceeds the time to stop
	done := make(chan interface{})
	if err := l.handler.StartService(done, svc, state); err != nil {
		return nil, err
	}
	return done, nil
}

func (l *HostStateListener) attachInstance(svc *service.Service, state *servicestate.ServiceState) (<-chan interface{}, error) {
	done := make(chan interface{})
	if err := l.handler.AttachService(done, svc, state); err != nil {
		return nil, err
	}
	return done, nil
}

func (l *HostStateListener) pauseInstance(svc *service.Service, state *servicestate.ServiceState) error {
	glog.Infof("Pausing service instance %s for service %s (%s)", state.ID, svc.Name, svc.ID)
	if err := l.handler.PauseService(svc, state); err != nil {
		glog.Errorf("Could not pause service instance %s: %s", state.ID, err)
		return err
	}
	return nil
}

func (l *HostStateListener) resumeInstance(svc *service.Service, state *servicestate.ServiceState) error {
	if err := l.handler.ResumeService(svc, state); err != nil {
		glog.Errorf("Could not resume service instance %s: %s", state.ID, err)
		return err
	}
	return nil
}

// stopInstance stops instance and signals done.  caller is expected to check for nil state
func (l *HostStateListener) stopInstance(done <-chan interface{}, state *servicestate.ServiceState) error {
	// TODO: may leave zombies hanging around if StopService fails...do we care?
	if err := l.handler.StopService(state); err != nil {
		glog.Errorf("Could not stop service instance %s: %s", state.ID, err)
	} else if done != nil {
		// wait for signal that the process is done
		glog.V(3).Infof("waiting for service instance %s to be updated", state.ID)
		<-done
	}

	glog.V(3).Infof("removing service state %s", state.ID)
	return removeInstance(l.conn, state)
}

func addInstance(conn client.Connection, state *servicestate.ServiceState) error {
	if state.ID == "" {
		return fmt.Errorf("missing service state id")
	} else if state.ServiceID == "" {
		return fmt.Errorf("missing service id")
	}

	var (
		spath = servicepath(state.ServiceID, state.ID)
		node  = &ServiceStateNode{ServiceState: state}
	)

	if err := conn.Create(spath, node); err != nil {
		return err
	} else if err := conn.Create(hostpath(state.HostID, state.ID), NewHostState(state)); err != nil {
		// try to clean up if create fails
		if err := conn.Delete(spath); err != nil {
			glog.Warningf("Could not remove service instance %s: %s", state.ID, err)
		}
		return err
	}
	return nil
}

func removeInstance(conn client.Connection, state *servicestate.ServiceState) error {
	if state == nil {
		return nil
	}

	if err := conn.Delete(hostpath(state.HostID, state.ID)); err != nil {
		glog.Warningf("Could not delete host state %s: %s", state.HostID, state.ID)
	}
	return conn.Delete(servicepath(state.ServiceID, state.ID))
}

// pauseInstance only updates the service instance if the instance is marked as RUN
func pauseInstance(conn client.Connection, hostID, stateID string) error {
	hpath := hostpath(hostID, stateID)
	var hs HostState
	if err := conn.Get(hpath, &hs); err != nil {
		return err
	}
	if hs.DesiredState != service.SVCRun {
		return nil
	}
	glog.V(2).Infof("Pausing service instance %s via host %s", stateID, hostID)
	hs.DesiredState = service.SVCPause
	return conn.Set(hpath, &hs)
}

// resumeInstance only updates the service instance if the instance is marked as PAUSED
func resumeInstance(conn client.Connection, hostID, stateID string) error {
	hpath := hostpath(hostID, stateID)
	var hs HostState
	if err := conn.Get(hpath, &hs); err != nil {
		return err
	}
	if hs.DesiredState != service.SVCPause {
		return nil
	}
	glog.V(2).Infof("Resuming service instance %s via host %s", stateID, hostID)
	hs.DesiredState = service.SVCRun
	return conn.Set(hpath, &hs)
}

func StopServiceInstance(conn client.Connection, hostID, stateID string) error {
	hpath := hostpath(hostID, stateID)
	var hs HostState
	if err := conn.Get(hpath, &hs); err != nil {
		return err
	}
	glog.V(2).Infof("Stopping instance %s via host %s", stateID, hostID)
	hs.DesiredState = service.SVCStop
	return conn.Set(hpath, &hs)
}
