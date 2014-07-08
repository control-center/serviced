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
	zkutils "github.com/zenoss/serviced/zzk/utils"
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
	AttachService(chan<- interface{}, *service.Service, *servicestate.ServiceState) error
	StartService(chan<- interface{}, *service.Service, *servicestate.ServiceState) error
	StopService(*servicestate.ServiceState) error
}

// HostStateListener is the listener for monitoring service instances
type HostStateListener struct {
	conn    client.Connection
	handler HostStateHandler
	host    *host.Host
	regpath string // path to the registered node
}

// NewHostListener instantiates a HostListener object
func NewHostStateListener(conn client.Connection, handler HostStateHandler, host *host.Host) *HostStateListener {
	return &HostStateListener{
		conn:    conn,
		handler: handler,
		host:    host,
	}
}

// Listen starts the HostListener by monitoring when new service instances are
// started, updated, or removed
func (l *HostStateListener) Listen(shutdown <-chan interface{}) {
	var (
		done       = make(chan string)
		processing = make(map[string]interface{})
	)

	// Make the path
	hpath := hostpath(l.host.ID)
	if exists, err := zkutils.PathExists(l.conn, hpath); err != nil {
		glog.Errorf("Unable to look up host path %s on zookeeper: %s", l.host.ID, err)
		return
	} else if exists {
		// pass
	} else if err := l.conn.CreateDir(hpath); err != nil {
		glog.Errorf("Unable to create host path %s: %s", hpath, err)
		return
	}

	// Housekeeping
	defer func() {
		glog.Infof("Agent receieved interrupt")
		for len(processing) > 0 {
			delete(processing, <-done)
		}
		if err := l.conn.Delete(hpath); err != nil {
			glog.Warningf("Could not clean up host %s: %s", l.host.ID, err)
		}
	}()

	// Register the host
	if err := l.register(); err != nil {
		glog.Errorf("Could not register host %s: %s", l.host.ID, err)
	}

	// Monitor the instances
	for {
		stateIDs, event, err := l.conn.ChildrenW(hpath)
		if err != nil {
			glog.Errorf("Could not watch for states on host %s: %s", l.host.ID, err)
			return
		}

		for _, ssid := range stateIDs {
			if _, ok := processing[ssid]; !ok {
				glog.V(1).Info("Spawning a listener for %s", ssid)
				processing[ssid] = nil
				go l.listenHostState(shutdown, done, ssid)
			}
		}

		select {
		case e := <-event:
			glog.V(2).Infof("Received event: %v", e)
		case ssid := <-done:
			glog.V(2).Info("Cleaning up %s", ssid)
			delete(processing, ssid)
		case <-shutdown:
			l.unregister()
			return
		}
	}
}

func (l *HostStateListener) listenHostState(shutdown <-chan interface{}, done chan<- string, ssID string) {
	defer func() {
		glog.V(2).Info("Shutting down listener for host instance ", ssID)
		done <- ssID
	}()

	var processDone <-chan interface{}
	hpath := hostpath(l.host.ID, ssID)
	for {
		var hs HostState
		event, err := l.conn.GetW(hpath, &hs)
		if err != nil {
			glog.Errorf("Could not load host instance %s: %s", ssID, err)
			return
		}

		if hs.ServiceID == "" || hs.ServiceStateID == "" {
			glog.Error("Invalid host state instance: ", hpath)
			return
		}

		var state servicestate.ServiceState
		if err := l.conn.Get(servicepath(hs.ServiceID, hs.ServiceStateID), &ServiceStateNode{ServiceState: &state}); err != nil {
			glog.Error("Could not find service instance: ", hs.ServiceStateID)
			// Node doesn't exist or cannot be loaded, delete
			if err := l.conn.Delete(hpath); err != nil {
				glog.Warningf("Could not delete host state %s: %s", ssID, err)
			}
			return
		}

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
				processDone, err = l.startInstance(&svc, &state)
			} else if processDone == nil {
				processDone, err = l.attachInstance(&svc, &state)
			}
			if err != nil {
				glog.Errorf("Error trying to start or attach to service instance %s: %s", state.ID, err)
				l.stopInstance(&state)
				return
			}
		case service.SVCStop:
			if processDone != nil {
				l.detachInstance(processDone, &state)
			} else {
				l.stopInstance(&state)
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
					l.detachInstance(processDone, &state)
				} else {
					l.stopInstance(&state)
				}
				return
			}
		case <-shutdown:
			glog.V(2).Infof("Host instance %s receieved signal to shutdown", hs.ServiceStateID)
			if processDone != nil {
				l.detachInstance(processDone, &state)
			} else {
				l.stopInstance(&state)
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
			glog.Errorf("Could not update the service instance %s with the time terminated (%s): %s", s.ID, s.Terminated.UnixNano(), err)
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
	return removeInstance(l.conn, state.HostID, state.ID)
}

func (l *HostStateListener) detachInstance(done <-chan interface{}, state *servicestate.ServiceState) error {
	if err := l.handler.StopService(state); err != nil {
		return err
	}
	<-done
	return removeInstance(l.conn, state.HostID, state.ID)
}

func (l *HostStateListener) register() (err error) {
	if l.regpath != "" {
		if ok, err := zkutils.PathExists(l.conn, l.regpath); err != nil {
			return err
		} else if ok {
			return nil
		}
	}

	l.regpath, err = registerHost(l.conn, l.host)
	return err
}

func (l *HostStateListener) unregister() {
	if err := l.conn.Delete(l.regpath); err != nil {
		glog.Warning("Could not unregister host ", l.host.ID)
	}
}

func (l *HostStateListener) Reset(conn client.Connection, host *host.Host) error {
	l.conn = conn
	l.host = host
	return l.conn.Set(l.regpath, &HostNode{Host: host})
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

func removeInstance(conn client.Connection, hostID, ssID string) error {
	var hs HostState
	if err := conn.Get(hostpath(hostID, ssID), &hs); err != nil {
		return err
	} else if err := conn.Delete(hostpath(hostID, ssID)); err != nil {
		return err
	} else if err := conn.Delete(servicepath(hs.ServiceID, hs.ServiceStateID)); err != nil {
		return err
	}
	return nil
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