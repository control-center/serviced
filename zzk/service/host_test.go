// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package service

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"
)

type TestHostStateHandler struct {
	processing map[string]chan<- interface{}
	states     map[string]*servicestate.ServiceState
}

func NewTestHostStateHandler() *TestHostStateHandler {
	return new(TestHostStateHandler).init()
}

func (handler *TestHostStateHandler) init() *TestHostStateHandler {
	*handler = TestHostStateHandler{
		processing: make(map[string]chan<- interface{}),
		states:     make(map[string]*servicestate.ServiceState),
	}
	return handler
}

func (handler *TestHostStateHandler) GetHost(hostID string) (*host.Host, error) {
	return &host.Host{ID: hostID}, nil
}

func (handler *TestHostStateHandler) AttachService(done chan<- interface{}, svc *service.Service, state *servicestate.ServiceState) error {
	if instanceC, ok := handler.processing[state.ID]; ok {
		delete(handler.processing, state.ID)
		close(instanceC)
		handler.processing[state.ID] = done
		return nil
	}

	return fmt.Errorf("instance %s not running", state.ID)
}

func (handler *TestHostStateHandler) StartService(done chan<- interface{}, svc *service.Service, state *servicestate.ServiceState) error {
	if _, ok := handler.processing[state.ID]; !ok {
		handler.processing[state.ID] = done
		handler.states[state.ID] = state
		(*state).Started = time.Now()
		return nil
	}

	return fmt.Errorf("instance %s already started", state.ID)
}

func (handler *TestHostStateHandler) StopService(state *servicestate.ServiceState) error {
	if instanceC, ok := handler.processing[state.ID]; ok {
		delete(handler.processing, state.ID)
		delete(handler.states, state.ID)
		close(instanceC)
	}
	return nil
}

func (handler *TestHostStateHandler) UpdateInstance(state *servicestate.ServiceState) error {
	if _, ok := handler.states[state.ID]; ok {
		handler.states[state.ID] = state
		return nil
	}

	return fmt.Errorf("instance %s not found", state.ID)
}

func TestHostStateListener_Listen(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostStateHandler()
	listener := NewHostStateListener(conn, handler, "test-host-1")
	RegisterHost(conn, listener.hostID)
	shutdown := make(chan interface{})
	wait := make(chan interface{})
	go func() {
		zzk.Listen(shutdown, listener)
		close(wait)
	}()

	// Create the service
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.ID, err)
	}

	var states []*servicestate.ServiceState
	for i := 0; i < 3; i++ {
		// Create a service instance
		state, err := servicestate.BuildFromService(svc, listener.hostID)
		if err != nil {
			t.Fatalf("Could not generate instance from service %s", svc.ID)
		} else if err := addInstance(conn, state); err != nil {
			t.Fatalf("Could not add instance %s from service %s", state.ID, state.ServiceID)
		}
		states = append(states, state)
	}

	spath := servicepath(states[0].ServiceID, states[0].ID)
	var s servicestate.ServiceState
	eventC, err := conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Could not get watch for service instance %s: %s", states[0].ID, err)
	}

	// verify the service has started
	<-eventC
	eventC, err = conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Could not get watch for service instance %s: %s", states[0].ID, err)
	}
	if s.Started.UnixNano() <= s.Terminated.UnixNano() {
		t.Fatalf("Service instance %s not started", s.ID)
	}

	// schedule stopping the instance
	<-time.After(3 * time.Second)
	t.Logf("Stopping service instance %s", states[0].ID)
	if err := StopServiceInstance(conn, listener.hostID, states[0].ID); err != nil {
		t.Fatalf("Could not stop service instance %s: %s", states[0].ID, err)
	}

	// verify the instance stopped
	<-eventC
	eventC, err = conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Could not get watch for service instance %s: %s", states[0].ID, err)
	}
	if s.Started.UnixNano() > s.Terminated.UnixNano() {
		t.Fatalf("Service instance %s not stopped", s.ID)
	}

	// verify the instance was removed
	if e := <-eventC; e.Type != client.EventNodeDeleted {
		t.Errorf("Service instance %s still exists for service %s", s.ID, s.ServiceID)
	} else if exists, err := zzk.PathExists(conn, hostpath(listener.hostID, s.ID)); err != nil {
		t.Fatalf("Error checking the instance %s for host %s: %s", s.ID, listener.hostID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for host %s", s.ID, listener.hostID)
	}

	// shutdown
	<-time.After(3 * time.Second)
	close(shutdown)
	<-wait
	hpath := hostpath(listener.hostID)
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

func TestHostStateListener_Spawn_StartAndStop(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostStateHandler()
	listener := NewHostStateListener(conn, handler, "test-host-1")
	RegisterHost(conn, listener.hostID)

	// Create the service
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.ID, err)
	}

	// Create the service instance
	state, err := servicestate.BuildFromService(svc, listener.hostID)
	if err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.ID)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.ID, state.ServiceID)
	}

	t.Log("Stop the instance and verify restart")
	var s servicestate.ServiceState
	spath := servicepath(state.ServiceID, state.ID)
	eventC, err := conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Could not add watch to %s: %s", spath, err)
	}

	var wg sync.WaitGroup
	shutdown := make(chan interface{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(shutdown, state.ID)
	}()

	// get the start time and stop the service
	<-eventC
	eventC, err = conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Could not add watch to %s: %s", spath, err)
	}
	startTime := s.Started
	stopTime := s.Terminated
	if err := handler.StopService(&s); err != nil {
		t.Fatalf("Could not stop instance %s: %s", s.ID, err)
	}

	// verify the instance stopped and started
	<-eventC
	eventC, err = conn.GetW(spath, &ServiceStateNode{ServiceState: &s})
	if err != nil {
		t.Fatalf("Could not add watch to %s: %s", spath, err)
	}
	if stopTime.UnixNano() == s.Terminated.UnixNano() {
		t.Errorf("Service instance %s not stopped", s.ID)
	}

	<-eventC
	if err := conn.Get(spath, &ServiceStateNode{ServiceState: &s}); err != nil {
		t.Fatalf("Could not add watch to %s: %s", spath, err)
	}
	if startTime.UnixNano() == s.Started.UnixNano() {
		t.Errorf("Service instance %s not started", s.ID)
	}

	// stop the service instance and verify
	StopServiceInstance(conn, listener.hostID, s.ID)
	wg.Wait()
	if exists, err := zzk.PathExists(conn, spath); err != nil {
		t.Fatalf("Error checking the instance %s for service %s: %s", s.ID, s.ServiceID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for service %s", s.ID, s.ServiceID)
	} else if exists, err := zzk.PathExists(conn, hostpath(listener.hostID, s.ID)); err != nil {
		t.Fatalf("Error checking the instance %s for host %s: %s", s.ID, listener.hostID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for host %s", s.ID, listener.hostID)
	}
}

func TestHostStateListener_Spawn_AttachAndDelete(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostStateHandler()
	listener := NewHostStateListener(conn, handler, "test-host-1")
	RegisterHost(conn, listener.hostID)

	// Create the service
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.ID, err)
	}

	// Create the service instance
	state, err := servicestate.BuildFromService(svc, listener.hostID)
	if err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.ID)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.ID, state.ServiceID)
	}

	t.Log("Start the instance and verify attach")
	spath := servicepath(state.ServiceID, state.ID)
	if err := handler.StartService(make(chan interface{}), svc, state); err != nil {
		t.Fatalf("Could not start instance %s: %s", state.ID, err)
	}
	defer handler.StopService(state)
	// Update instance with the start time
	if err := conn.Set(spath, &ServiceStateNode{ServiceState: state}); err != nil {
		t.Fatalf("Could not update instance %s: %s", state.ID, err)
	}

	var wg sync.WaitGroup
	shutdown := make(chan interface{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(shutdown, state.ID)
	}()

	// Remove the instance and verify stopped
	<-time.After(3 * time.Second)
	removeInstance(conn, state)
	wg.Wait()
	if exists, err := zzk.PathExists(conn, spath); err != nil {
		t.Fatalf("Error checking the instance %s for service %s: %s", state.ID, state.ServiceID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for service %s", state.ID, state.ServiceID)
	} else if exists, err := zzk.PathExists(conn, hostpath(listener.hostID, state.ID)); err != nil {
		t.Fatalf("Error checking the instance %s for host %s: %s", state.ID, listener.hostID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for host %s", state.ID, listener.hostID)
	}
}

func TestHostStateListener_Spawn_Shutdown(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostStateHandler()
	listener := NewHostStateListener(conn, handler, "test-host-1")
	RegisterHost(conn, listener.hostID)

	// Create the service
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.ID, err)
	}

	// Create the service instance
	state, err := servicestate.BuildFromService(svc, listener.hostID)
	if err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.ID)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.ID, state.ServiceID)
	}

	var wg sync.WaitGroup
	shutdown := make(chan interface{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(shutdown, state.ID)
	}()

	// wait 3 seconds and shutdown
	t.Log("Shutdown and verify the instance")
	<-time.After(3)
	close(shutdown)
	spath := servicepath(state.ServiceID, state.ID)
	wg.Wait()
	if exists, err := zzk.PathExists(conn, spath); err != nil {
		t.Fatalf("Error checking the instance %s for service %s: %s", state.ID, state.ServiceID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for service %s", state.ID, state.ServiceID)
	} else if exists, err := zzk.PathExists(conn, hostpath(listener.hostID, state.ID)); err != nil {
		t.Fatalf("Error checking the instance %s for host %s: %s", state.ID, listener.hostID, err)
	} else if exists {
		t.Errorf("Instance %s still exists for host %s", state.ID, listener.hostID)
	}
}

func TestHostStateListener_stopInstance(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostStateHandler()
	listener := NewHostStateListener(conn, handler, "test-host-1")
	RegisterHost(conn, listener.hostID)

	// Create the instance
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}

	t.Log("Adding and stopping an instance from the connection")
	if state, err := servicestate.BuildFromService(svc, listener.hostID); err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.ID)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.ID, state.ServiceID)
	} else if err := listener.stopInstance(state); err != nil {
		t.Fatalf("Could not stop instance %s from service %s", state.ID, state.ServiceID)
	} else if exists, err := zzk.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
		t.Fatalf("Error while checking the existance of %s from service %s", state.ID, state.ServiceID)
	} else if exists {
		t.Errorf("Failed to delete node %s from %s", state.ID, state.ServiceID)
	} else if exists, err := zzk.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
		t.Fatalf("Error while checking the existance of %s from host %s", state.ID, state.HostID)
	} else if exists {
		t.Errorf("Failed to delete node %s from %s", state.ID, state.HostID)
	}
}

func TestHostStateListener_detachInstance(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := NewTestHostStateHandler()
	listener := NewHostStateListener(conn, handler, "test-host-1")
	RegisterHost(conn, listener.hostID)

	// Create the instance
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}

	t.Log("Adding an instance from the connection and waiting to detach")
	state, err := servicestate.BuildFromService(svc, listener.hostID)
	if err != nil {
		t.Fatalf("Could not generate instance from service %s", svc.ID)
	} else if err := addInstance(conn, state); err != nil {
		t.Fatalf("Could not add instance %s from service %s", state.ID, state.ServiceID)
	}

	done := make(chan interface{})
	wait := make(chan interface{})
	go func() {
		defer close(wait)
		if err := listener.detachInstance(done, state); err != nil {
			t.Fatalf("Could not detach instance %s from service %s", state.ID, state.ServiceID)
		}
	}()

	<-time.After(time.Second)
	close(done)
	<-wait

	if exists, err := zzk.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
		t.Fatalf("Error while checking the existance of %s from service %s", state.ID, state.ServiceID)
	} else if exists {
		t.Errorf("Failed to delete node %s from %s", state.ID, state.ServiceID)
	} else if exists, err := zzk.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
		t.Fatalf("Error while checking the existance of %s from host %s", state.ID, state.HostID)
	} else if exists {
		t.Errorf("Failed to delete node %s from %s", state.ID, state.HostID)
	}
}
