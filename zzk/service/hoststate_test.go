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

package service_test

import (
	"errors"
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/service/mocks"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

var (
	ErrTestNoAttach = errors.New("could not attach to container")
)

const (
	servicePath string = "/services/serviceid"
	hostPath    string = "/hosts/hostid"
	hostId      string = "hostid"
	serviceId   string = "serviceid"
	serviceName string = "serviceA"
	imageId     string = "imageid"
	containerId string = "containerid"
)

// TODO: break ZZKTest suite up into hoststate, init, service, etc. and move this into SetUpTest.
func setUpServiceAndHostPaths(c *C) client.Connection {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// Basic set up
	sdat := &ServiceNode{
		ID:   serviceId,
		Name: serviceName,
	}
	err = conn.Create(servicePath, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir(hostPath)
	c.Assert(err, IsNil)

	return conn
}

// Test Case: Bad state id
func (t *ZZKTest) TestHostStateListener_Spawn_BadStateID(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	cancel := make(chan interface{})

	done := make(chan struct{})
	go func() {
		listener.Spawn(cancel, "badstateid")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		close(cancel)
		c.Fatalf("Listener did not exit")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)
	timer := time.NewTimer(time.Second)
	select {
	case <-shutdowndone:
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Missing host state
func (t *ZZKTest) TestHostStateListener_Spawn_ErrHostState(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	// delete the host state
	handler.On("StopContainer", serviceId, 1).Return(nil)
	err = conn.Delete("/hosts/hostid/instances/" + req.StateID())
	c.Assert(err, IsNil)

	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	cancel := make(chan interface{})

	done := make(chan struct{})
	ok, ev, err := conn.ExistsW("/services/serviceid/"+req.StateID(), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Logf("Listener cleaned up orphaned node")
		timer.Reset(time.Second)

		select {
		case <-done:
			c.Logf("Listener exit")
		case <-timer.C:
			c.Fatalf("Listener did not exit")
		}
	case <-done:
		c.Logf("Listener exit, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Listener did not exit")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Missing service state
func (t *ZZKTest) TestHostStateListener_Spawn_ErrServiceState(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	// delete the service state
	handler.On("StopContainer", serviceId, 1).Return(nil)
	err = conn.Delete("/services/serviceid/" + req.StateID())
	c.Assert(err, IsNil)

	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	cancel := make(chan interface{})

	done := make(chan struct{})
	ok, ev, err := conn.ExistsW("/hosts/hostid/instances/"+req.StateID(), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Logf("Listener cleaned up orphaned node")
		timer.Reset(time.Second)

		select {
		case <-done:
			c.Logf("Listener exit")
		case <-timer.C:
			c.Fatalf("Listener did not exit")
		}
	case <-done:
		c.Logf("Listener exit, checking orphaned node deletion")
		ok, err := conn.Exists("/hosts/hostid/instances/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)

	case <-timer.C:
		close(shutdown)
		c.Fatalf("Listener did not exit")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Missing service state once the hoststate listener is running
func (t *ZZKTest) TestHostStateListener_Spawn_ErrServiceState2(c *C) {
	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	// attached, run, no change
	ssdat := ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.ServiceState = ssdat
		return true
	})
	c.Assert(err, IsNil)
	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// service's stateId must exist
	done := make(chan struct{})
	sspth := "/services/serviceid/" + req.StateID()
	ok, err := conn.Exists(sspth)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// host's stateId must exist
	hspth := "/hosts/hostid/instances/" + req.StateID()
	ok, hsEvt, err := conn.ExistsW(hspth, done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// Run listener
	cancel := make(chan interface{})
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) { containerExit <- time.Now() })

	timer := time.NewTimer(1 * time.Second)
	select {
	case <-timer.C: // wait for the listener enter its infinite loop
	case <-done:
		c.Fatalf("Host StateId not cleaned up")
	case <-hsEvt:
		c.Fatalf("Host StateId not cleaned up")
	}

	// Delete service stateId
	err = conn.Delete(sspth)
	c.Assert(err, IsNil)

	timer = time.NewTimer(1 * time.Second)
	select {
	case <-done: //listener exited
		ok, err := conn.Exists(hspth)
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Timed out waiting for exit")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Error on attach
func (t *ZZKTest) TestHostStateListener_Spawn_ErrAttach(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})

	handler.On("StopContainer", serviceId, 1).Return(nil)
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(nil, ErrTestNoAttach)

	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	done := make(chan struct{})
	ok, ev, err := conn.ExistsW("/hosts/hostid/instances/"+req.StateID(), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Logf("Listener cleaned up orphaned node")
		timer.Reset(time.Second)

		select {
		case <-done:
			c.Logf("Listener exit")
		case <-timer.C:
			c.Fatalf("Listener did not exit")
		}
	case <-done:
		c.Logf("Listener exit, checking orphaned node deletion")
		ok, err := conn.Exists("/hosts/hostid/instances/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		close(cancel)
		c.Fatalf("Listener did not exit")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a running container
func (t *ZZKTest) TestHostStateListener_Spawn_AttachRun(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	// attached, run, no change
	ssdat := ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.ServiceState = ssdat
		return true
	})
	c.Assert(err, IsNil)
	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) { containerExit <- time.Now() })
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	done := make(chan struct{})
	sspth := "/services/serviceid/" + req.StateID()
	ev, err := conn.GetW(sspth, &ServiceState{}, done)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	select {
	case <-ev:
		c.Errorf("service state changed unexpectedly")
	case <-time.After(time.Second):
	}

	close(cancel)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-time.After(5 * time.Second):
		c.Fatalf("Listener did not exit")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)
	timer := time.NewTimer(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a paused running container
func (t *ZZKTest) TestHostStateListener_Spawn_AttachResume(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}

	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// set up a paused running container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      true,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCPause
		s.ServiceState = *ssdat
		return true
	})
	c.Assert(err, IsNil)

	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()

	done := make(chan struct{})
	ev, err := conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
	}

	// resume container
	handler.On("ResumeContainer", serviceId, 1).Return(nil)
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRun
		return true
	})
	c.Assert(err, IsNil)
	timer.Reset(time.Second)
	select {
	case <-ev:
		// make sure the state is paused
		ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
		c.Assert(err, IsNil)

		// may have been triggered by event to update desired state
		if ssdat.Paused {
			timer.Reset(time.Second)
			select {
			case <-ev:
				ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
				c.Assert(err, IsNil)
			case <-done:
				c.Fatalf("Listener exit")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		}

		c.Check(ssdat.Paused, Equals, false)
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// cancel
	close(cancel)
	timer.Reset(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	})
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a running container and is triggered to
// restart it.
func (t *ZZKTest) TestHostStateListener_Spawn_AttachRestart(c *C) {
	// Constants
	const ContainerId2 = "containerid2"

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}

	err = UpdateState(conn, req, func(s *State) bool {
		s.ServiceState = *ssdat
		return true
	})
	c.Assert(err, IsNil)

	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()

	done := make(chan struct{})

	ev, err := conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
	c.Assert(err, IsNil)

	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
	}

	ssdat = &ServiceState{
		ContainerID: ContainerId2,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	var retShutdown <-chan interface{} = shutdown

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(nil, nil).Once()
	handler.On("StartContainer", retShutdown, serviceId, 1).Return(ssdat, retExit, nil).Once()
	containerExit <- time.Now()
	timer = time.NewTimer(time.Second)
	select {
	case <-ev:
		// state is either terminated or overwritten
		ssdat2 := &ServiceState{}
		ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdat2, done)
		c.Assert(err, IsNil)
		if ssdat2.ContainerID == containerId {
			timer.Reset(time.Second)
			c.Check(ssdat2.Terminated.IsZero(), Equals, false)

			select {
			case <-ev:
				ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdat2, done)
				c.Assert(err, IsNil)
			case <-done:
				c.Fatalf("Listener exit")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		}
		ssdat2.SetVersion(nil)
		c.Check(ssdat2.Terminated.IsZero(), Equals, true)
		ssdat2.Terminated = ssdat.Terminated
		c.Check(ssdat2.Started.Equal(ssdat.Started), Equals, true)
		ssdat2.Started = ssdat.Started
		c.Check(ssdat2.Restarted.IsZero(), Equals, true)
		ssdat2.Restarted = ssdat.Restarted
		c.Check(ssdat2, DeepEquals, ssdat)
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	close(cancel)
	timer = time.NewTimer(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(args mock.Arguments) { containerExit <- time.Now() }).Once()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a running container and pauses the state
func (t *ZZKTest) TestHostStateListener_Spawn_AttachPause(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.ServiceState = *ssdat
		return true
	})
	c.Assert(err, IsNil)

	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()

	done := make(chan struct{})
	ev, err := conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
	c.Assert(err, IsNil)
	c.Logf("ev channel address is %v", ev)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
	}

	// pause container
	handler.On("PauseContainer", serviceId, 1).Return(nil)
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCPause
		return true
	})
	c.Assert(err, IsNil)
	timer.Reset(time.Second)
	select {
	case <-ev:
		// make sure the state is paused
		ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
		c.Assert(err, IsNil)

		// may have been triggered by event to update desired state
		if !ssdat.Paused {
			timer.Reset(time.Second)
			select {
			case <-ev:
				ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
				c.Assert(err, IsNil)
			case <-done:
				c.Fatalf("Listener exit")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		}

		c.Check(ssdat.Paused, Equals, true)
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// cancel
	close(cancel)
	timer.Reset(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}
}

// Test Case: Listener attaches to a paused running container (no change)
func (t *ZZKTest) TestHostStateListener_Spawn_AttachPausePaused(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// set up a paused running container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      true,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCPause
		s.ServiceState = *ssdat
		return true
	})
	c.Assert(err, IsNil)

	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()

	done := make(chan struct{})
	ev, err := conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
	}

	// cancel
	close(cancel)
	timer = time.NewTimer(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Listener to pause a stopped container (no change)
func (t *ZZKTest) TestHostStateListener_Spawn_DetachPause(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCPause
		return true
	})
	c.Assert(err, IsNil)

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(nil, nil).Once()

	done := make(chan struct{})
	ev, err := conn.GetW("/services/serviceid/"+req.StateID(), &ServiceState{}, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
	}

	// cancel
	close(cancel)
	timer = time.NewTimer(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	handler.On("StopContainer", serviceId, 1).Return(nil).Once()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a running container and stops
func (t *ZZKTest) TestHostStateListener_Spawn_AttachStop(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCStop
		s.ServiceState = *ssdat
		return true
	})
	c.Assert(err, IsNil)

	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()

	done := make(chan struct{})

	ok, err := conn.Exists("/services/serviceid/" + req.StateID())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Post-Process stops unnecessary containers
func (t *ZZKTest) TestHostStateListener_PostProcess(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRun
		s.ServiceState = *ssdat
		return true
	})
	c.Assert(err, IsNil)

	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()
	done := make(chan struct{})

	ok, err := conn.Exists("/services/serviceid/")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
	}

	// Cancel the listener, this won't stop the container or clean up the node
	close(cancel)
	timer.Reset(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit, checking that node still exists")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, true)
	case <-timer.C:
		c.Fatalf("Listener took too long to exit")
	}

	// Call post-process with an empty map, this WILL stop the container and clean up nodes
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()

	threadMap := make(map[string]struct{})
	ppdone := make(chan struct{})
	go func() {
		listener.PostProcess(threadMap)
		close(ppdone)
	}()

	timer.Reset(5 * time.Second)
	select {
	case <-ppdone:
		c.Logf("PostProcess done, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Post process took too long")
	}

	// Make sure StopContainer was called
	handler.AssertExpectations(c)

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}
}

// Test Case: Re-attach to existing container after cancel
func (t *ZZKTest) TestHostStateListener_Spawn_Cancel_ReSpawn(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRun
		s.ServiceState = *ssdat
		return true
	})
	c.Assert(err, IsNil)

	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()
	done := make(chan struct{})

	ok, err := conn.Exists("/services/serviceid/")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
	}

	// Cancel the listener, this won't stop the container or clean up the node
	close(cancel)
	timer.Reset(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit, checking that node still exists")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, true)
	case <-timer.C:
		c.Fatalf("Listener took too long to exit")
	}

	// Call spawn again, no need to re-attach
	cancel = make(chan interface{})
	done = make(chan struct{})

	ok, err = conn.Exists("/services/serviceid/")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	// Wait for spawn to stabilize
	timer.Reset(time.Second)
	select {
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
	}

	// Call shutdown without canceling, should still clean up
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener shut down took too long")
	}

	// Make sure spawn thread exited
	timer.Reset(time.Second)
	select {
	case <-done:
		c.Logf("Spawn thread exited")
	case <-timer.C:
		c.Fatalf("Spawn thread did not exit")
	}

	handler.AssertExpectations(c)
}

// Test Case: shutdown and cancel at the same time
func (t *ZZKTest) TestHostStateListener_Spawn_Cancel_Shutdown(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRun
		s.ServiceState = *ssdat
		return true
	})
	c.Assert(err, IsNil)

	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(retExit, nil).Once()
	done := make(chan struct{})

	ok, err := conn.Exists("/services/serviceid/")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
	}

	// Shutdown and cancel at the same time
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()
	shutdowndone := listener.GetShutdownComplete()

	go close(shutdown)
	close(cancel)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener shut down took too long")
	}

	// Make sure spawn thread exited
	timer.Reset(time.Second)
	select {
	case <-done:
		c.Logf("Spawn thread exited")
	case <-timer.C:
		c.Fatalf("Spawn thread did not exit")
	}

	handler.AssertExpectations(c) //this makes sure StopContainer was called
}

// Test Case: spawn after shutdown does nothing
func (t *ZZKTest) TestHostStateListener_Shutdown_Spawn(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRun
		s.ServiceState = *ssdat
		return true
	})
	c.Assert(err, IsNil)

	done := make(chan struct{})
	ok, err := conn.Exists("/services/serviceid/" + req.StateID())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// Close shutdown before spawning
	shutdowndone := listener.GetShutdownComplete()
	close(shutdown)

	// Spawn should exit immediately
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long to exit")
	}

	// Wait for shutdown to complete
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down")
	case <-timer.C:
		c.Fatalf("Listener shut down took too long")
	}

	handler.AssertExpectations(c) //this makes sure StopContainer was called

	// Clean up the node we created
	err = conn.Delete("/services/serviceid/" + req.StateID())
	c.Assert(err, IsNil)
}

// Test Case: Start a container, restart it
func (t *ZZKTest) TestHostStateListener_Spawn_StartRestart(c *C) {
	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// Start a container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	var retShutdown <-chan interface{} = shutdown
	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(nil, nil)
	handler.On("StartContainer", retShutdown, serviceId, 1).Return(ssdat, retExit, nil).Once()

	done := make(chan struct{})
	ssdatResult := &ServiceState{}
	ev, err := conn.GetW(path.Join(servicePath, req.StateID()), ssdatResult, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		// Make sure we set the service state appropriately
		ev, err = conn.GetW(path.Join(servicePath, req.StateID()), ssdatResult, done)
		c.Assert(err, IsNil)

		c.Assert(ssdatResult.ContainerID, Equals, ssdat.ContainerID)
		c.Assert(ssdatResult.ImageUUID, Equals, ssdat.ImageUUID)
		c.Assert(ssdatResult.Started.Unix(), Equals, ssdat.Started.Unix())
		c.Assert(ssdatResult.Paused, Equals, ssdat.Paused)
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// Restart the container
	handler.On("RestartContainer", retShutdown, serviceId, 1).Return(nil)
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRestart
		return true
	})
	c.Assert(err, IsNil)

	// wait for the state to set itself back to run
	timer = time.NewTimer(time.Second)
	hsdata := &HostState{}

	for {
		ev, err = conn.GetW(path.Join(hostPath, "instances", req.StateID()), hsdata, done)
		c.Assert(err, IsNil)
		if hsdata.DesiredState == service.SVCRun {
			break
		}

		select {
		case <-ev:
		case <-done:
			c.Fatalf("Listener exit")
		case <-timer.C:
			c.Fatalf("Listener took too long")
		}
	}

	// cancel
	close(cancel)
	timer = time.NewTimer(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: Container stop during restart
func (t *ZZKTest) TestHostStateListener_Spawn_StartRestartStop(c *C) {
	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// Start a container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	var retShutdown <-chan interface{} = shutdown
	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(nil, nil)
	handler.On("StartContainer", retShutdown, serviceId, 1).Return(ssdat, retExit, nil)
	done := make(chan struct{})

	s := &ServiceState{}
	ev, err := conn.GetW(path.Join(servicePath, req.StateID()), s, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// Restart the container
	handler.On("RestartContainer", retShutdown, serviceId, 1).Return(nil).Run(
		func(_ mock.Arguments) {
			err := UpdateState(conn, req, func(s *State) bool {
				s.DesiredState = service.SVCStop
				return true
			})
			c.Assert(err, IsNil)
		},
	)
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(
		func(_ mock.Arguments) {
			containerExit <- time.Now()
		},
	).Once()

	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRestart
		return true
	})
	c.Assert(err, IsNil)

	// wait for the listener to exit
	timer = time.NewTimer(time.Second)
	select {
	case <-done:
		ok, err := conn.Exists(path.Join(servicePath, req.StateID()))
		c.Assert(ok, Equals, false)
		c.Assert(err, IsNil)
		ok, err = conn.Exists(path.Join(hostPath, "instances", req.StateID()))
		c.Assert(ok, Equals, false)
		c.Assert(err, IsNil)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: restart during restart
func (t *ZZKTest) TestHostStateListener_Spawn_StartRestartRestart(c *C) {
	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// Start a container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	var retShutdown <-chan interface{} = shutdown
	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(nil, nil)
	handler.On("StartContainer", retShutdown, serviceId, 1).Return(ssdat, retExit, nil).Once()
	done := make(chan struct{})

	s := &ServiceState{}
	ev, err := conn.GetW(path.Join(servicePath, req.StateID()), s, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// Restart the container
	h := &HostState{}

	handler.On("RestartContainer", retShutdown, serviceId, 1).Return(nil).Once()
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRestart
		return true
	})
	c.Assert(err, IsNil)

	timer = time.NewTimer(time.Second)
	for {
		ev, err = conn.GetW(path.Join(hostPath, "instances", req.StateID()), h, done)
		c.Assert(err, IsNil)
		if h.DesiredState != service.SVCRun {
			select {
			case <-ev:
			case <-done:
				c.Fatalf("Listener exit")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		} else {
			break
		}
	}
	err = conn.Get(path.Join(servicePath, req.StateID()), s)
	c.Assert(err, IsNil)
	c.Assert(s.Restarted.After(s.Started), Equals, true)

	// Restart the container while it is restarting
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRestart
		return true
	})
	c.Assert(err, IsNil)

	timer = time.NewTimer(time.Second)
	for {
		ev, err = conn.GetW(path.Join(hostPath, "instances", req.StateID()), h, done)
		c.Assert(err, IsNil)
		if h.DesiredState != service.SVCRun {
			select {
			case <-ev:
			case <-done:
				c.Fatalf("Listener exit")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		} else {
			break
		}
	}

	// ensure restart value wasn't updated
	restarted := s.Restarted
	err = conn.Get(path.Join(servicePath, req.StateID()), s)
	c.Assert(err, IsNil)
	c.Assert(s.Restarted.Equal(restarted), Equals, true)

	// cancel
	close(cancel)
	timer = time.NewTimer(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// Test Case: restart and then reconnect
func (t *ZZKTest) TestHostStateListener_Spawn_StartRestartDetach(c *C) {
	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	// Start a container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	var retShutdown <-chan interface{} = shutdown
	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(nil, nil)
	handler.On("StartContainer", retShutdown, serviceId, 1).Return(ssdat, retExit, nil).Once()
	done := make(chan struct{})

	s := &ServiceState{}
	ev, err := conn.GetW(path.Join(servicePath, req.StateID()), s, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// Restart the container
	handler.On("RestartContainer", retShutdown, serviceId, 1).Return(nil).Run(
		func(_ mock.Arguments) {
			// close the connection to simulate a wan outage
			conn.Close()
		},
	).Once()

	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRestart
		return true
	})
	c.Assert(err, IsNil)

	select {
	case <-done:
	case <-time.After(time.Second):
		c.Fatalf("Listener did not shut down")
	}

	conn, err = zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	done = make(chan struct{})
	hsdata := &HostState{}
	ev, err = conn.GetW(path.Join(hostPath, "instances", req.StateID()), hsdata, done)
	c.Assert(err, IsNil)
	c.Assert(hsdata.DesiredState, Equals, service.SVCRestart)

	listener.SetConnection(conn)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	// Ensure the state change
	timer = time.NewTimer(time.Second)
	for hsdata.DesiredState != service.SVCRun {
		select {
		case <-ev:
			ev, err = conn.GetW(path.Join(hostPath, "instances", req.StateID()), hsdata, done)
			c.Assert(err, IsNil)
		case <-done:
			c.Fatalf("Listener exit")
		case <-timer.C:
			c.Fatalf("Listener took too long")
		}
	}

	err = conn.Get(path.Join(servicePath, req.StateID()), s)
	c.Assert(err, IsNil)
	c.Assert(s.Restarted.After(s.Started), Equals, true)

	// cancel
	close(cancel)
	timer = time.NewTimer(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}

// CC-3102
// Test Case: Start a container, pause it, then resume
func (t *ZZKTest) TestHostStateListener_Spawn_StartPauseStart(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}
	shutdown := make(chan interface{})

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	cancel := make(chan interface{})
	listener := NewHostStateListener(handler, hostId, shutdown)
	listener.SetConnection(conn)

	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRun
		return true
	})
	c.Assert(err, IsNil)

	// Start a container
	ssdat := &ServiceState{
		ContainerID: containerId,
		ImageUUID:   imageId,
		Paused:      false,
		Started:     time.Now(),
	}
	var retShutdown <-chan interface{} = shutdown
	containerExit := make(chan time.Time, 1)
	var retExit <-chan time.Time = containerExit

	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(nil, nil).Once()
	handler.On("StartContainer", retShutdown, serviceId, 1).Return(ssdat, retExit, nil).Once()

	done := make(chan struct{})
	ssdatResult := &ServiceState{}
	ev, err := conn.GetW("/services/serviceid/"+req.StateID(), ssdatResult, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(cancel, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		// Make sure we set the service state appropriately
		ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdatResult, done)
		c.Assert(err, IsNil)

		c.Assert(ssdatResult.ContainerID, Equals, ssdat.ContainerID)
		c.Assert(ssdatResult.ImageUUID, Equals, ssdat.ImageUUID)
		c.Assert(ssdatResult.Started.Unix(), Equals, ssdat.Started.Unix())
		c.Assert(ssdatResult.Paused, Equals, ssdat.Paused)
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// Pause the container
	handler.On("PauseContainer", serviceId, 1).Return(nil)
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCPause
		return true
	})
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case <-ev:
		// make sure the state is paused
		ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdatResult, done)
		c.Assert(err, IsNil)

		// may have been triggered by event to update desired state
		if !ssdatResult.Paused {
			timer.Reset(time.Second)
			select {
			case <-ev:
				ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdatResult, done)
				c.Assert(err, IsNil)
			case <-done:
				c.Fatalf("Listener exit")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		}
		// Make sure the zk data is correct
		c.Assert(ssdatResult.ContainerID, Equals, ssdat.ContainerID)
		c.Assert(ssdatResult.ImageUUID, Equals, ssdat.ImageUUID)
		c.Assert(ssdatResult.Started.Unix(), Equals, ssdat.Started.Unix())
		c.Assert(ssdatResult.Paused, Equals, true)
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// Resume the container
	handler.On("ResumeContainer", serviceId, 1).Return(nil)
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCRun
		return true
	})
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case <-ev:
		// make sure the state is NOT paused
		ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdatResult, done)
		c.Assert(err, IsNil)

		// may have been triggered by event to update desired state
		if ssdatResult.Paused {
			timer.Reset(time.Second)
			select {
			case <-ev:
				ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdatResult, done)
				c.Assert(err, IsNil)
			case <-done:
				c.Fatalf("Listener exit")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		}
		// Make sure the zk data is correct
		c.Assert(ssdatResult.ContainerID, Equals, ssdat.ContainerID)
		c.Assert(ssdatResult.ImageUUID, Equals, ssdat.ImageUUID)
		c.Assert(ssdatResult.Started.Unix(), Equals, ssdat.Started.Unix())
		c.Assert(ssdatResult.Paused, Equals, false)
	case <-done:
		c.Fatalf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// cancel
	close(cancel)
	timer = time.NewTimer(time.Second)
	select {
	case <-done:
		c.Logf("Listener exit")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	shutdowndone := listener.GetShutdownComplete()
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-shutdowndone:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.AssertExpectations(c)
}
