package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	zkutils "github.com/zenoss/serviced/zzk/utils"
)

type TestHostHandler struct {
	processing map[string]chan<- interface{}
	states     map[string]*servicestate.ServiceState
}

func NewTestHostHandler() *TestHostHandler {
	return new(TestHostHandler).init()
}

func (handler *TestHostHandler) init() *TestHostHandler {
	*handler = TestHostHandler{
		processing: make(map[string]chan<- interface{}),
		states:     make(map[string]*servicestate.ServiceState),
	}
	return handler
}

func (handler *TestHostHandler) AttachService(done chan<- interface{}, svc *service.Service, state *servicestate.ServiceState) error {
	if instanceC, ok := handler.processing[state.Id]; ok {
		delete(handler.processing, state.Id)
		close(instanceC)
		handler.processing[state.Id] = done
		return nil
	}

	return fmt.Errorf("instance %s not running", state.Id)
}

func (handler *TestHostHandler) StartService(done chan<- interface{}, svc *service.Service, state *servicestate.ServiceState) error {
	if _, ok := handler.processing[state.Id]; !ok {
		handler.processing[state.Id] = done
		handler.states[state.Id] = state
		state.Started = time.Now()
		return nil
	}

	return fmt.Errorf("instance %s already started", state.Id)
}

func (handler *TestHostHandler) StopService(state *servicestate.ServiceState) error {
	if instanceC, ok := handler.processing[state.Id]; ok {
		delete(handler.processing, state.Id)
		delete(handler.states, state.Id)
		close(instanceC)
	}
	return nil
}

func (handler *TestHostHandler) CheckInstance(state *servicestate.ServiceState) error {
	if s, ok := handler.states[state.Id]; ok {
		*state = *s
		return nil
	}

	return fmt.Errorf("instance %s not found", state.Id)
}

func (handler *TestHostHandler) UpdateInstance(state *servicestate.ServiceState) error {
	if _, ok := handler.states[state.Id]; ok {
		handler.states[state.Id] = state
		return nil
	}

	return fmt.Errorf("instance %s not found", state.Id)
}

func TestHostListener_Listen(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostHandler()
	listener := NewHostStateListener(conn, handler, &host.Host{ID: "test-host-1"})
	shutdown := make(chan interface{})
	wait := make(chan interface{})
	go func() {
		listener.Listen(shutdown)
		close(wait)
	}()

	// Create the service
	svc := &service.Service{
		Id:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.Id, err)
	}

	var states []*servicestate.ServiceState
	for i := 0; i < 3; i++ {
		// Create a service instance
		state, err := servicestate.BuildFromService(svc, listener.host.ID)
		if err != nil {
			t.Fatalf("Could not generate instance from service %s", svc.Id)
		} else if err := addInstance(conn, state); err != nil {
			t.Fatalf("Could not add instance %s from service %s", state.Id, state.ServiceID)
		}
		states = append(states, state)
	}

	// stop 1 instance and verify
	spath := servicepath(states[0].ServiceID, states[0].Id)
	var s servicestate.ServiceState
	eventC, err := conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Error retrieving watch for %s: %s", spath, err)
	}
	<-time.After(3 * time.Second)
	if err := StopServiceInstance(conn, listener.host.ID, states[0].Id); err != nil {
		t.Fatalf("Could not stop service instance %s: %s", states[0].Id, err)
	}
	<-eventC
	eventC, err = conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Error retrieving watch for %s: %s", spath, err)
	}
	<-eventC
	if exists, err := zkutils.PathExists(conn, spath); err != nil {
		t.Fatalf("Error checking the instance %s for service %s: %s", s.Id, s.ServiceID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for service %s", s.Id, s.ServiceID)
	} else if exists, err := zkutils.PathExists(conn, hostpath(listener.host.ID, s.Id)); err != nil {
		t.Fatalf("Error checking the instance %s for host %s: %s", s.Id, listener.host.ID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for host %s", s.Id, listener.host.ID)
	}

	// shutdown
	<-time.After(3 * time.Second)
	close(shutdown)
	<-wait
	hpath := hostpath(listener.host.ID)
	if children, err := conn.Children(spath); err != nil {
		t.Fatalf("Error checking children for %s: %s", spath, err)
	} else if len(children) > 0 {
		t.Errorf("Found nodes for %s: %s", spath, children)
	} else if children, err := conn.Children(hpath); err != nil {
		t.Fatalf("Error checking children for %s: %s", hpath, err)
	} else if len(children) > 0 {
		t.Errorf("Found nodes for %s: %s", hpath, children)
	}
}

func TestHostListener_listenHostState_StartAndStop(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostHandler()
	listener := NewHostStateListener(conn, handler, &host.Host{ID: "test-host-1"})

	// Create the service
	svc := &service.Service{
		Id:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.Id, err)
	}

	// Create the service instance
	state, err := servicestate.BuildFromService(svc, listener.host.ID)
	if err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.Id)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.Id, state.ServiceID)
	}

	shutdown := make(chan interface{})
	done := make(chan string)
	go listener.listenHostState(shutdown, done, state.Id)

	t.Log("Stop the instance and verify restart")
	var s servicestate.ServiceState
	spath := servicepath(state.ServiceID, state.Id)
	eventC, err := conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Could not add watch to %s: %s", spath, err)
	}

	// get the start time and stop the service
	<-eventC
	eventC, err = conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Could not add watch to %s: %s", spath, err)
	}
	startTime := s.Started
	stopTime := s.Terminated
	if err := handler.StopService(&s); err != nil {
		t.Fatalf("Could not stop instance %s: %s", s.Id, err)
	}

	// verify the instance stopped and started
	<-eventC
	eventC, err = conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Could not add watch to %s: %s", spath, err)
	}
	if stopTime.UnixNano() == s.Terminated.UnixNano() {
		t.Errorf("Service instance %s not stopped", s.Id)
	}

	<-eventC
	if err := conn.Get(spath, &ServiceStateNode{ServiceState: &s}); err != nil {
		t.Fatalf("Could not add watch to %s: %s", spath, err)
	}
	if startTime.UnixNano() == s.Started.UnixNano() {
		t.Errorf("Service instance %s not started", s.Id)
	}

	// stop the service instance and verify
	StopServiceInstance(conn, listener.host.ID, s.Id)
	if ssID := <-done; ssID != state.Id {
		t.Errorf("MISMATCH: instances do not match! (%s != %s)", state.Id, ssID)
	} else if exists, err := zkutils.PathExists(conn, spath); err != nil {
		t.Fatalf("Error checking the instance %s for service %s: %s", s.Id, s.ServiceID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for service %s", s.Id, s.ServiceID)
	} else if exists, err := zkutils.PathExists(conn, hostpath(listener.host.ID, s.Id)); err != nil {
		t.Fatalf("Error checking the instance %s for host %s: %s", s.Id, listener.host.ID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for host %s", s.Id, listener.host.ID)
	}
}

func TestHostListener_listenHostState_AttachAndDelete(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostHandler()
	listener := NewHostStateListener(conn, handler, &host.Host{ID: "test-host-1"})

	// Create the service
	svc := &service.Service{
		Id:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.Id, err)
	}

	// Create the service instance
	state, err := servicestate.BuildFromService(svc, listener.host.ID)
	if err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.Id)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.Id, state.ServiceID)
	}

	t.Log("Start the instance and verify attach")
	spath := servicepath(state.ServiceID, state.Id)
	if err := handler.StartService(make(chan interface{}), svc, state); err != nil {
		t.Fatalf("Could not start instance %s: %s", state.Id, err)
	}
	defer handler.StopService(state)
	// Update instance with the start time
	if err := conn.Set(spath, &ServiceStateNode{ServiceState: state}); err != nil {
		t.Fatalf("Could not update instance %s: %s", state.Id, err)
	}

	shutdown := make(chan interface{})
	done := make(chan string)
	go listener.listenHostState(shutdown, done, state.Id)

	// Remove the instance and verify stopped
	<-time.After(3 * time.Second)
	removeInstance(conn, state.HostID, state.Id)
	if ssID := <-done; ssID != state.Id {
		t.Errorf("MISMATCH: instances do not match! (%s != %s)", state.Id, ssID)
	} else if exists, err := zkutils.PathExists(conn, spath); err != nil {
		t.Fatalf("Error checking the instance %s for service %s: %s", state.Id, state.ServiceID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for service %s", state.Id, state.ServiceID)
	} else if exists, err := zkutils.PathExists(conn, hostpath(listener.host.ID, state.Id)); err != nil {
		t.Fatalf("Error checking the instance %s for host %s: %s", state.Id, listener.host.ID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for host %s", state.Id, listener.host.ID)
	}
}

func TestHostListener_listenHostState_Shutdown(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostHandler()
	listener := NewHostStateListener(conn, handler, &host.Host{ID: "test-host-1"})

	// Create the service
	svc := &service.Service{
		Id:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.Id, err)
	}

	// Create the service instance
	state, err := servicestate.BuildFromService(svc, listener.host.ID)
	if err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.Id)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.Id, state.ServiceID)
	}

	shutdown := make(chan interface{})
	done := make(chan string)
	go listener.listenHostState(shutdown, done, state.Id)

	// wait 3 seconds and shutdown
	t.Log("Shutdown and verify the instance")
	<-time.After(3)
	close(shutdown)
	spath := servicepath(state.ServiceID, state.Id)
	if ssID := <-done; ssID != state.Id {
		t.Errorf("MISMATCH: instances do not match! (%s != %s)", state.Id, ssID)
	} else if exists, err := zkutils.PathExists(conn, spath); err != nil {
		t.Fatalf("Error checking the instance %s for service %s: %s", state.Id, state.ServiceID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for service %s", state.Id, state.ServiceID)
	} else if exists, err := zkutils.PathExists(conn, hostpath(listener.host.ID, state.Id)); err != nil {
		t.Fatalf("Error checking the instance %s for host %s: %s", state.Id, listener.host.ID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for host %s", state.Id, listener.host.ID)
	}
}

func TestHostListener_stopInstance(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostHandler()
	listener := NewHostStateListener(conn, handler, &host.Host{ID: "test-host-1"})

	// Create the instance
	svc := &service.Service{
		Id:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}

	t.Log("Adding and stopping an instance from the connection")
	if state, err := servicestate.BuildFromService(svc, listener.host.ID); err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.Id)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.Id, state.ServiceID)
	} else if err := listener.stopInstance(state); err != nil {
		t.Fatalf("Could not stop instance %s from service %s", state.Id, state.ServiceID)
	} else if exists, err := zkutils.PathExists(conn, servicepath(state.ServiceID, state.Id)); err != nil {
		t.Fatalf("Error while checking the existance of %s from service %s", state.Id, state.ServiceID)
	} else if exists {
		t.Errorf("Failed to delete node %s from %s", state.Id, state.ServiceID)
	} else if exists, err := zkutils.PathExists(conn, hostpath(state.HostID, state.Id)); err != nil {
		t.Fatalf("Error while checking the existance of %s from host %s", state.Id, state.HostID)
	} else if exists {
		t.Errorf("Failed to delete node %s from %s", state.Id, state.HostID)
	}
}

func TestHostListener_detachInstance(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostHandler()
	listener := NewHostStateListener(conn, handler, &host.Host{ID: "test-host-1"})

	// Create the instance
	svc := &service.Service{
		Id:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}

	t.Log("Adding an instance from the connection and waiting to detach")
	state, err := servicestate.BuildFromService(svc, listener.host.ID)
	if err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.Id)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.Id, state.ServiceID)
	}

	done := make(chan interface{})
	wait := make(chan interface{})
	go func() {
		defer close(wait)
		if err := listener.detachInstance(done, state); err != nil {
			t.Fatalf("Could not detach instance %s from service %s", state.Id, state.ServiceID)
		}
	}()

	<-time.After(time.Second)
	close(done)
	<-wait

	if exists, err := zkutils.PathExists(conn, servicepath(state.ServiceID, state.Id)); err != nil {
		t.Fatalf("Error while checking the existance of %s from service %s", state.Id, state.ServiceID)
	} else if exists {
		t.Errorf("Failed to delete node %s from %s", state.Id, state.ServiceID)
	} else if exists, err := zkutils.PathExists(conn, hostpath(state.HostID, state.Id)); err != nil {
		t.Fatalf("Error while checking the existance of %s from host %s", state.Id, state.HostID)
	} else if exists {
		t.Errorf("Failed to delete node %s from %s", state.Id, state.HostID)
	}
}