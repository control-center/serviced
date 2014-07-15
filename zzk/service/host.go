// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package service

import (
	"fmt"
	"path"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
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
func NewHostStateListener(conn client.Connection, handler HostStateHandler, hostID string) *HostStateListener {
	return &HostStateListener{
		conn:    conn,
		handler: handler,
		hostID:  hostID,
	}
}

// GetConnection implements zzk.Listener
func (l *HostStateListener) GetConnection() client.Connection { return l.conn }

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
	l.registry, err = l.conn.CreateEphemeral(l.GetPath(), &HostNode{Host: host})
	return err
}

// Done removes the ephemeral node from the host registry
func (l *HostStateListener) Done() {
	if err := l.conn.Delete(l.registry); err != nil {
		glog.Warning("Could not unregister host %s: %s", l.hostID, err)
	}
}

// Spawn listens for changes in the host state and manages running instances
func (l *HostStateListener) Spawn(shutdown <-chan interface{}, stateID string) {
	var processDone <-chan interface{}
	hpath := l.GetPath(stateID)
	state := &servicestate.ServiceState{} // Just in case our state gets deleted from under us
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
			// Node doesn't exist or cannot be loaded, delete
			if err := l.conn.Delete(hpath); err != nil {
				glog.Warningf("Could not delete host state %s: %s", stateID, err)
			}
			if state == nil {
				return
			}

			if processDone != nil {
				glog.V(2).Infof("detaching from %s; %s", hs.ServiceID, hs.ServiceStateID)
				go l.detachInstance(processDone, state)
				select {
				case <-processDone:
					glog.V(2).Infof("detached from %s; %s", hs.ServiceID, hs.ServiceStateID)
				case <-time.After(45 * time.Second):
					glog.Infof("timed out detaching from %s; %s", hs.ServiceID, hs.ServiceStateID)
				}
			} else {
				glog.V(2).Infof("stopping from %s; %s", hs.ServiceID, hs.ServiceStateID)
				l.stopInstance(state)
				glog.V(2).Infof("stopped from %s; %s", hs.ServiceID, hs.ServiceStateID)

			}
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
			if state.Started.UnixNano() <= state.Terminated.UnixNano() {
				processDone, err = l.startInstance(&svc, state)
			} else if processDone == nil {
				processDone, err = l.attachInstance(&svc, state)
			}
			if err != nil {
				glog.Errorf("Error trying to start or attach to service instance %s: %s", state.ID, err)
				l.stopInstance(state)
				return
			}
		case service.SVCStop:
			if processDone != nil {
				l.detachInstance(processDone, state)
			} else {
				l.stopInstance(state)
			}
			return
		default:
			glog.V(2).Infof("Unhandled service %s (%s)", svc.Name, svc.ID)
		}

		select {
		case <-processDone:
			glog.V(2).Infof("Process ended for instance: ", hs.ServiceStateID)
			processDone = nil
		case e := <-event:
			glog.V(3).Info("Receieved event: ", e)
			if e.Type == client.EventNodeDeleted {
				if processDone != nil {
					l.detachInstance(processDone, state)
				} else {
					l.stopInstance(state)
				}
				return
			}
		case <-shutdown:
			glog.V(2).Infof("Service %s Host instance %s receieved signal to shutdown", hs.ServiceID, hs.ServiceStateID)
			if processDone != nil {
				glog.V(2).Infof("detaching from %s; %s", hs.ServiceID, hs.ServiceStateID)
				go l.detachInstance(processDone, state)
				select {
				case <-processDone:
					glog.V(2).Infof("detached from %s; %s", hs.ServiceID, hs.ServiceStateID)
				case <-time.After(45 * time.Second):
					glog.Infof("timed out detaching from %s; %s", hs.ServiceID, hs.ServiceStateID)
				}
			} else {
				glog.V(2).Infof("stopping from %s; %s", hs.ServiceID, hs.ServiceStateID)
				l.stopInstance(state)
				glog.V(2).Infof("stopped from %s; %s", hs.ServiceID, hs.ServiceStateID)

			}
			return
		}
	}
}

func (l *HostStateListener) updateInstance(done <-chan interface{}, state *servicestate.ServiceState) (<-chan interface{}, error) {
	wait := make(chan interface{})
	go func(path string) {
		defer close(wait)
		<-done
		var s servicestate.ServiceState
		if err := l.conn.Get(path, &ServiceStateNode{ServiceState: &s}); err != nil {
			glog.Warningf("Could not get service state %s: %s", state.ID, err)
			return
		}

		s.Terminated = time.Now()
		if err := updateInstance(l.conn, &s); err != nil {
			glog.Warningf("Could not update the service instance %s with the time terminated (%s): %s", s.ID, s.Terminated.UnixNano(), err)
			return
		}
	}(servicepath(state.ServiceID, state.ID))

	return wait, updateInstance(l.conn, state)
}

func (l *HostStateListener) startInstance(svc *service.Service, state *servicestate.ServiceState) (<-chan interface{}, error) {
	done := make(chan interface{})
	if err := l.handler.StartService(done, svc, state); err != nil {
		return nil, err
	}

	wait, err := l.updateInstance(done, state)
	if err != nil {
		return nil, err
	}

	return wait, nil
}

func (l *HostStateListener) attachInstance(svc *service.Service, state *servicestate.ServiceState) (<-chan interface{}, error) {
	done := make(chan interface{})
	if err := l.handler.AttachService(done, svc, state); err != nil {
		return nil, err
	}

	wait, err := l.updateInstance(done, state)
	if err != nil {
		return nil, err
	}

	return wait, nil
}

func (l *HostStateListener) stopInstance(state *servicestate.ServiceState) error {
	if err := l.handler.StopService(state); err != nil {
		return err
	}
	return removeInstance(l.conn, state)
}

func (l *HostStateListener) detachInstance(done <-chan interface{}, state *servicestate.ServiceState) error {
	if err := l.handler.StopService(state); err != nil {
		return err
	}
	<-done
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

func updateInstance(conn client.Connection, state *servicestate.ServiceState) error {
	return conn.Set(servicepath(state.ServiceID, state.ID), &ServiceStateNode{ServiceState: state})
}

func removeInstance(conn client.Connection, state *servicestate.ServiceState) error {
	if err := conn.Delete(hostpath(state.HostID, state.ID)); err != nil {
		glog.Warningf("Could not delete host state %s: %s", state.HostID, state.ID)
	}
	return conn.Delete(servicepath(state.ServiceID, state.ID))
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
