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
	GetHost(string) (*host.Host, error)
	AttachService(chan<- interface{}, *service.Service, *servicestate.ServiceState) error
	StartService(chan<- interface{}, *service.Service, *servicestate.ServiceState) error
	StopService(*servicestate.ServiceState) error
}

// HostStateListener is the listener for monitoring service instances
type HostStateListener struct {
	conn    client.Connection
	handler HostStateHandler
	hostID  string
}

// NewHostListener instantiates a HostListener object
func NewHostStateListener(conn client.Connection, handler HostStateHandler, hostID string) *HostStateListener {
	return &HostStateListener{
		conn:    conn,
		handler: handler,
		hostID:  hostID,
	}
}

// Listen starts the HostListener by monitoring when new service instances are
// started, updated, or removed
func (l *HostStateListener) Listen(shutdown <-chan interface{}) {
	var (
		_shutdown  = make(chan interface{})
		done       = make(chan string)
		processing = make(map[string]interface{})
	)

	// Register the host
	regpath, err := l.register(shutdown)
	if err != nil {
		glog.Errorf("Could not register host %s: %s", l.hostID, err)
		return
	}

	// Housekeeping
	defer func() {
		glog.Infof("Agent receieved interrupt")
		if err := l.conn.Delete(regpath); err != nil {
			glog.Warning("Could not unregister host %s: %s", l.hostID, err)
		}
		close(_shutdown)
		for len(processing) > 0 {
			delete(processing, <-done)
		}
	}()

	// Monitor the instances
	hpath := hostpath(l.hostID)
	for {
		stateIDs, event, err := l.conn.ChildrenW(hpath)
		if err != nil {
			glog.Errorf("Could not watch for states on host %s: %s", l.hostID, err)
			return
		}

		for _, ssid := range stateIDs {
			if _, ok := processing[ssid]; !ok {
				glog.V(1).Info("Spawning a listener for %s", ssid)
				processing[ssid] = nil
				go l.listenHostState(_shutdown, done, ssid)
			}
		}

		select {
		case e := <-event:
			if e.Type == client.EventNodeDeleted {
				glog.Infof("Host has been removed from pool, shutting down listener")
				return
			}
			glog.V(2).Infof("Received event: %v", e)
		case ssid := <-done:
			glog.V(2).Info("Cleaning up %s", ssid)
			delete(processing, ssid)
		case <-shutdown:
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
	hpath := hostpath(l.hostID, ssID)
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

// register waits for the leader to initialize the host
func (l *HostStateListener) register(shutdown <-chan interface{}) (string, error) {
	// wait for /hosts
	for {
		exists, err := zkutils.PathExists(l.conn, hostpath())
		if err != nil {
			return "", err
		}
		if exists {
			break
		}
		_, event, err := l.conn.ChildrenW("/")
		if err != nil {
			return "", err
		}
		select {
		case <-event:
		case <-shutdown:
			return "", ErrShutdown
		}
	}

	// wait for /hosts/HOSTID
	for {
		exists, err := zkutils.PathExists(l.conn, hostpath(l.hostID))
		if err != nil {
			return "", err
		}
		if exists {
			break
		}
		_, event, err := l.conn.ChildrenW(hostpath())
		if err != nil {
			return "", err
		}
		select {
		case <-event:
		case <-shutdown:
			return "", ErrShutdown
		}
	}

	host, err := l.handler.GetHost(l.hostID)
	if err != nil {
		return "", err
	}
	return registerHost(l.conn, host)
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