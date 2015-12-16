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

// +build integration,!quick

package service

import (
	"fmt"
	"path"

	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"

	. "gopkg.in/check.v1"
)

type action int

const (
	attach action = iota
	start
	pause
	resume
	stop
)

type request struct {
	state    *servicestate.ServiceState
	action   action
	purge    func(string)
	response chan<- error
}

type TestHostStateHandler struct {
	instances map[string]func(string)
	requestC  chan request
}

func (handler *TestHostStateHandler) init() *TestHostStateHandler {
	h := TestHostStateHandler{make(map[string]func(string)), make(chan request)}

	go func() {
		for req := range h.requestC {
			switch req.action {
			case attach:
				if _, ok := h.instances[req.state.ID]; !ok {
					req.response <- fmt.Errorf("attach: not running")
					continue
				}
				h.instances[req.state.ID] = req.purge
				req.state.Started = time.Now()
				req.response <- nil
			case start:
				if _, ok := h.instances[req.state.ID]; ok {
					req.response <- fmt.Errorf("start: running")
					continue
				}
				h.instances[req.state.ID] = req.purge
				req.state.Started = time.Now()
				req.response <- nil
			case pause:
				if _, ok := h.instances[req.state.ID]; !ok || !req.state.IsRunning() {
					req.response <- fmt.Errorf("pause: not running")
					continue
				} else if req.state.IsPaused() {
					req.response <- fmt.Errorf("pause: paused")
					continue
				}
				req.response <- nil
			case resume:
				if _, ok := h.instances[req.state.ID]; !ok || !req.state.IsRunning() {
					req.response <- fmt.Errorf("resume: not running")
					continue
				} else if !req.state.IsPaused() {
					req.response <- fmt.Errorf("resume: resumed")
					continue
				}
				req.response <- nil
			case stop:
				purge, ok := h.instances[req.state.ID]
				if !ok {
					req.response <- fmt.Errorf("stop: not running")
					continue
				}

				delete(h.instances, req.state.ID)
				purge(req.state.ID)
				req.response <- nil
			}
		}
	}()
	return &h
}

func (h *TestHostStateHandler) PullImage(cancel <-chan time.Time, imageID string) (string, error) {
	return "", nil
}

func (h *TestHostStateHandler) AttachService(svc *service.Service, state *servicestate.ServiceState, purge func(stateID string)) error {
	response := make(chan error)
	h.requestC <- request{state, attach, purge, response}
	return <-response
}

func (h *TestHostStateHandler) StartService(svc *service.Service, state *servicestate.ServiceState, purge func(stateID string)) error {
	response := make(chan error)
	h.requestC <- request{state, start, purge, response}
	return <-response
}

func (h *TestHostStateHandler) PauseService(svc *service.Service, state *servicestate.ServiceState) error {
	response := make(chan error)
	h.requestC <- request{state, pause, nil, response}
	return <-response
}

func (h *TestHostStateHandler) ResumeService(svc *service.Service, state *servicestate.ServiceState) error {
	response := make(chan error)
	h.requestC <- request{state, resume, nil, response}
	return <-response
}

func (h *TestHostStateHandler) StopService(state *servicestate.ServiceState) error {
	response := make(chan error)
	h.requestC <- request{state, stop, nil, response}
	return <-response
}

func (t *ZZKTest) TestHostStateListener_Listen(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	defer close(shutdown)
	errC := make(chan error, 1)

	handler := new(TestHostStateHandler).init()
	listener := NewHostStateListener(handler, "test-host-1")
	go zzk.Listen(shutdown, errC, conn, listener)

	// Add a service
	svc := service.Service{ID: "test-service-1", Instances: 3}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	// Add host
	err = AddHost(conn, &host.Host{ID: "test-host-1"})
	c.Assert(err, IsNil)

	// Verify that the host is registered
	c.Logf("Waiting for 'test-host-1' to be registered")
	select {
	case err := <-errC:
		c.Assert(err, IsNil)
		c.Assert(listener.registry, Not(Equals), "")

		exists, err := conn.Exists(listener.registry)
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, true)
	case <-time.After(zzk.ZKTestTimeout):
		// NOTE: this timeout may be adjusted to satisfy race conditions
		c.Fatalf("timeout waiting for host to be ready")
	}

	// Add states
	addstates := func(hostID string, svc *service.Service, count int) []string {
		c.Logf("Adding %d service states for service %s on host %s", count, svc.ID, hostID)
		stateIDs := make([]string, count)
		for i := 0; i < count; i++ {
			state, err := servicestate.BuildFromService(svc, hostID)
			c.Assert(err, IsNil)
			c.Assert(state.IsRunning(), Equals, false)
			err = addInstance(conn, *state)
			c.Assert(err, IsNil)
			_, err = LoadRunningService(conn, state.ServiceID, state.ID)
			c.Assert(err, IsNil)
			stateIDs[i] = state.ID
		}
		return stateIDs
	}
	stateIDs := addstates("test-host-1", &svc, 3)

	wait := func(serviceID string, dState service.DesiredState) {
		errC := make(chan error)
		c.Logf("Waiting for service instances on 'test-host-1' to %s", dState)
		go func() {
			errC <- WaitService(shutdown, conn, serviceID, dState)
		}()

		// Wait on services or fail trying
		select {
		case err := <-errC:
			c.Assert(err, IsNil)
		case <-time.After(zzk.ZKTestTimeout):
			c.Fatalf("timeout waiting for instances to %s", dState)
		}
	}
	wait(svc.ID, service.SVCRun)

	// Pause states
	for _, stateID := range stateIDs {
		err = pauseInstance(conn, "test-host-1", stateID)
		c.Assert(err, IsNil)
	}
	wait(svc.ID, service.SVCPause)

	// Resume states
	for _, stateID := range stateIDs {
		err = resumeInstance(conn, "test-host-1", stateID)
		c.Assert(err, IsNil)
	}
	wait(svc.ID, service.SVCRun)

	// Stop states
	for _, stateID := range stateIDs {
		err = StopServiceInstance(conn, "test-host-1", stateID)
		c.Assert(err, IsNil)
	}
	wait(svc.ID, service.SVCStop)
}

func (t *ZZKTest) TestHostStateListener_Listen_BadState(c *C) {
	conn, err := zzk.GetLocalConnection("/base_badstate")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	defer close(shutdown)
	errC := make(chan error, 1)

	handler := new(TestHostStateHandler).init()
	listener := NewHostStateListener(handler, "test-host-1")

	// Add a service
	svc := service.Service{ID: "test-service-1", Instances: 3}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	// Add the host
	err = AddHost(conn, &host.Host{ID: "test-host-1"})
	c.Assert(err, IsNil)

	// Create a host state without a service instance (this should not spin!)
	badstate := HostState{
		HostID:         listener.hostID,
		ServiceID:      svc.ID,
		ServiceStateID: "fail123",
		DesiredState:   int(service.SVCRun),
	}
	err = conn.Create(hostpath(badstate.HostID, badstate.ServiceStateID), &badstate)
	c.Assert(err, IsNil)
	err = conn.Set(hostpath(badstate.HostID, badstate.ServiceStateID), &badstate)
	c.Assert(err, IsNil)

	// Set up a watch
	watchDone := make(chan struct{})
	defer close(watchDone)
	event, err := conn.GetW(hostpath(badstate.HostID, badstate.ServiceStateID), &HostState{}, watchDone)
	c.Assert(err, IsNil)

	// Start the listener
	go zzk.Listen(shutdown, errC, conn, listener)

	select {
	case e := <-event:
		c.Assert(e.Type, Equals, client.EventNodeDeleted)
	case <-time.After(zzk.ZKTestTimeout):
		c.Fatalf("timeout waiting for event")
	}
}

func (t *ZZKTest) TestHostStateListener_Spawn_StartAndStop(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	defer close(shutdown)
	errC := make(chan error, 1)

	handler := new(TestHostStateHandler).init()
	listener := NewHostStateListener(handler, "test-host-1")
	listener.SetConnection(conn)

	// Add a service
	svc := service.Service{ID: "test-service-1", Instances: 1}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	// Add a host
	register := func(hostID string) string {
		c.Logf("Registering host %s", hostID)
		host := host.Host{ID: hostID}
		err := AddHost(conn, &host)
		c.Assert(err, IsNil)
		p, err := conn.CreateEphemeral(hostregpath(hostID), &HostNode{Host: &host})
		c.Assert(err, IsNil)
		return path.Base(p)
	}
	register("test-host-1")

	// Add states
	addstates := func(hostID string, svc *service.Service, count int) []string {
		c.Logf("Adding %d service states for service %s on host %s", count, svc.ID, hostID)
		stateIDs := make([]string, count)
		for i := 0; i < count; i++ {
			state, err := servicestate.BuildFromService(svc, hostID)
			c.Assert(err, IsNil)
			err = addInstance(conn, *state)
			c.Assert(err, IsNil)
			_, err = LoadRunningService(conn, state.ServiceID, state.ID)
			c.Assert(err, IsNil)
			stateIDs[i] = state.ID
		}
		return stateIDs
	}
	stateIDs := addstates("test-host-1", &svc, 1)
	stateID := stateIDs[0]

	wait := func(serviceID string, dState service.DesiredState) {
		c.Logf("Waiting for service instances on 'test-host-1' to %s", dState)
		go func() {
			errC <- WaitService(shutdown, conn, serviceID, dState)
		}()

		// Wait on services or fail trying
		select {
		case err := <-errC:
			c.Assert(err, IsNil)
		case <-time.After(zzk.ZKTestTimeout):
			c.Fatalf("timeout waiting for instances to %s", dState)
		}
	}

	var node1, node2 ServiceStateNode
	listener.Ready()
	go func() {
		listener.Spawn(shutdown, stateID)
	}()

	c.Logf("Checking instance start")
	wait(svc.ID, service.SVCRun)
	err = conn.Get(servicepath(svc.ID, stateID), &node1)
	c.Assert(err, IsNil)

	c.Logf("Stopping service instance")
	err = handler.StopService(node1.ServiceState)
	wait(svc.ID, service.SVCRun)
	err = conn.Get(servicepath(svc.ID, stateID), &node2)
	c.Assert(err, IsNil)
	c.Assert(node2.Started.After(node1.Started), Equals, true)

	c.Logf("Pausing service instance")
	err = pauseInstance(conn, "test-host-1", stateID)
	wait(svc.ID, service.SVCPause)

	c.Logf("Resuming service instance")
	err = resumeInstance(conn, "test-host-1", stateID)
	c.Assert(err, IsNil)
	wait(svc.ID, service.SVCRun)
	// Verify the instance wasn't restarted
	err = conn.Get(servicepath(svc.ID, stateID), &node1)
	c.Assert(err, IsNil)
	c.Assert(node1.Started.Unix(), Equals, node2.Started.Unix())

	c.Logf("Stopping service instance")
	err = StopServiceInstance(conn, "test-host-1", stateID)
	wait(svc.ID, service.SVCStop)
}

func (t *ZZKTest) TestHostStateListener_Spawn_AttachAndDelete(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	defer close(shutdown)

	handler := new(TestHostStateHandler).init()
	listener := NewHostStateListener(handler, "test-host-1")
	listener.SetConnection(conn)

	// Add a service
	svc := service.Service{ID: "test-service-1", Instances: 1}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	// Add a host
	register := func(hostID string) string {
		c.Logf("Registering host %s", hostID)
		host := host.Host{ID: hostID}
		err := AddHost(conn, &host)
		c.Assert(err, IsNil)
		p, err := conn.CreateEphemeral(hostregpath(hostID), &HostNode{Host: &host})
		c.Assert(err, IsNil)
		return path.Base(p)
	}
	register("test-host-1")

	// Add states
	addstates := func(hostID string, svc *service.Service, count int) []string {
		c.Logf("Adding %d service states for service %s on host %s", count, svc.ID, hostID)
		stateIDs := make([]string, count)
		for i := 0; i < count; i++ {
			state, err := servicestate.BuildFromService(svc, hostID)
			c.Assert(err, IsNil)
			err = addInstance(conn, *state)
			c.Assert(err, IsNil)
			_, err = LoadRunningService(conn, state.ServiceID, state.ID)
			c.Assert(err, IsNil)
			stateIDs[i] = state.ID
		}
		return stateIDs
	}
	stateIDs := addstates("test-host-1", &svc, 1)
	stateID := stateIDs[0]

	var node ServiceStateNode
	err = conn.Get(servicepath(svc.ID, stateID), &node)
	c.Assert(err, IsNil)

	c.Logf("Starting the instance to attach")
	err = handler.StartService(&svc, node.ServiceState, func(_ string) {})
	c.Assert(err, IsNil)
	// Update the start time
	err = conn.Set(servicepath(svc.ID, stateID), &node)
	c.Assert(err, IsNil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		listener.Spawn(shutdown, stateID)
	}()

	c.Logf("Removing the instance to verify shutdown")
	time.Sleep(zzk.ZKTestTimeout)
	err = removeInstance(conn, node.ServiceState.ServiceID, node.ServiceState.HostID, node.ServiceState.ID)

	select {
	case <-done:
	case <-time.After(zzk.ZKTestTimeout):
		c.Fatalf("timeout waiting for listener to shutdown")
	}

	exists, err := conn.Exists(servicepath(svc.ID, stateID))
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
	exists, err = conn.Exists(hostpath("test-host-1", stateID))
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
}

func (t *ZZKTest) TestHostStateListener_Spawn_Shutdown(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})

	handler := new(TestHostStateHandler).init()
	listener := NewHostStateListener(handler, "test-host-1")
	listener.SetConnection(conn)

	// Add a service
	svc := service.Service{ID: "test-service-1", Instances: 1}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	// Add a host
	register := func(hostID string) string {
		c.Logf("Registering host %s", hostID)
		host := host.Host{ID: hostID}
		err := AddHost(conn, &host)
		c.Assert(err, IsNil)
		p, err := conn.CreateEphemeral(hostregpath(hostID), &HostNode{Host: &host})
		c.Assert(err, IsNil)
		return path.Base(p)
	}
	register("test-host-1")

	// Add states
	addstates := func(hostID string, svc *service.Service, count int) []string {
		c.Logf("Adding %d service states for service %s on host %s", count, svc.ID, hostID)
		stateIDs := make([]string, count)
		for i := 0; i < count; i++ {
			state, err := servicestate.BuildFromService(svc, hostID)
			c.Assert(err, IsNil)
			err = addInstance(conn, *state)
			c.Assert(err, IsNil)
			_, err = LoadRunningService(conn, state.ServiceID, state.ID)
			c.Assert(err, IsNil)
			stateIDs[i] = state.ID
		}
		return stateIDs
	}
	stateIDs := addstates("test-host-1", &svc, 1)
	stateID := stateIDs[0]

	done := make(chan struct{})
	go func() {
		defer close(done)
		listener.Spawn(shutdown, stateID)
	}()

	time.Sleep(zzk.ZKTestTimeout)
	close(shutdown)

	select {
	case <-done:
	case <-time.After(zzk.ZKTestTimeout):
		c.Fatalf("timeout waiting for listener to shutdown")
	}

	exists, err := conn.Exists(servicepath(svc.ID, stateID))
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
	exists, err = conn.Exists(hostpath("test-host-1", stateID))
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
}

func (t *ZZKTest) TestHostStateListener_pauseANDresume(c *C) {
	conn, err := zzk.GetLocalConnection("/base_pauseANDresume")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)

	handler := new(TestHostStateHandler).init()
	listener := NewHostStateListener(handler, "test-host-1")
	listener.SetConnection(conn)

	// Add a service
	svc := service.Service{ID: "test-service-1", Instances: 1}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	// Add a host
	register := func(hostID string) string {
		c.Logf("Registering host %s", hostID)
		host := host.Host{ID: hostID}
		err := AddHost(conn, &host)
		c.Assert(err, IsNil)
		p, err := conn.CreateEphemeral(hostregpath(hostID), &HostNode{Host: &host})
		c.Assert(err, IsNil)
		return path.Base(p)
	}
	register("test-host-1")

	// Add states
	addstates := func(hostID string, svc *service.Service, count int) []string {
		c.Logf("Adding %d service states for service %s on host %s", count, svc.ID, hostID)
		stateIDs := make([]string, count)
		for i := 0; i < count; i++ {
			state, err := servicestate.BuildFromService(svc, hostID)
			c.Assert(err, IsNil)
			err = addInstance(conn, *state)
			c.Assert(err, IsNil)
			_, err = LoadRunningService(conn, state.ServiceID, state.ID)
			c.Assert(err, IsNil)
			stateIDs[i] = state.ID
		}
		return stateIDs
	}
	stateIDs := addstates("test-host-1", &svc, 1)
	stateID := stateIDs[0]

	var node ServiceStateNode
	err = conn.Get(servicepath(svc.ID, stateID), &node)
	c.Assert(err, IsNil)
	err = handler.StartService(&svc, node.ServiceState, func(_ string) {})
	c.Assert(err, IsNil)
	err = conn.Set(servicepath(node.ServiceID, node.ID), &node)
	c.Assert(err, IsNil)
	err = listener.pauseInstance(&svc, node.ServiceState)
	c.Assert(err, IsNil)
	err = conn.Get(servicepath(node.ServiceID, node.ID), &node)
	c.Assert(err, IsNil)
	c.Assert(node.IsPaused(), Equals, true)
	err = listener.resumeInstance(&svc, node.ServiceState)
	c.Assert(err, IsNil)
	err = conn.Get(servicepath(node.ServiceID, node.ID), &node)
	c.Assert(err, IsNil)
	c.Assert(node.IsPaused(), Equals, false)
}
