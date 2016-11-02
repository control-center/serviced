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

	listener := NewHostStateListener(handler, hostId)
	listener.SetConnection(conn)

	shutdown := make(chan interface{})

	done := make(chan struct{})
	go func() {
		listener.Spawn(shutdown, "badstateid")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		close(shutdown)
		c.Fatalf("Listener did not shut down")
	}
	handler.AssertExpectations(c)
}

// Test Case: Missing host state
func (t *ZZKTest) TestHostStateListener_Spawn_ErrHostState(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

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

	listener := NewHostStateListener(handler, hostId)
	listener.SetConnection(conn)

	shutdown := make(chan interface{})

	done := make(chan struct{})
	ok, ev, err := conn.ExistsW("/services/serviceid/"+req.StateID(), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	go func() {
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Logf("Listener cleaned up orphaned node")
		timer.Reset(time.Second)

		select {
		case <-done:
			c.Logf("Listener shut down")
		case <-timer.C:
			c.Fatalf("Listener did not shut down")
		}
	case <-done:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Listener did not shut down")
	}
	handler.AssertExpectations(c)
}

// Test Case: Missing service state
func (t *ZZKTest) TestHostStateListener_Spawn_ErrServiceState(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

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

	listener := NewHostStateListener(handler, hostId)
	listener.SetConnection(conn)

	shutdown := make(chan interface{})

	done := make(chan struct{})
	ok, ev, err := conn.ExistsW("/hosts/hostid/instances/"+req.StateID(), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	go func() {
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Logf("Listener cleaned up orphaned node")
		timer.Reset(time.Second)

		select {
		case <-done:
			c.Logf("Listener shut down")
		case <-timer.C:
			c.Fatalf("Listener did not shut down")
		}
	case <-done:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/hosts/hostid/instances/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)

	case <-timer.C:
		close(shutdown)
		c.Fatalf("Listener did not shut down")
	}
	handler.AssertExpectations(c)
}

// Test Case: Missing service state once the hoststate listener is running
func (t *ZZKTest) TestHostStateListener_Spawn_ErrServiceState2(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

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
	listener := NewHostStateListener(handler, hostId)
	listener.SetConnection(conn)

	// service's stateId must exist
	done := make(chan struct{})
	sspth := "/services/serviceid/" + req.StateID()
	ok, err := conn.Exists(sspth)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// host's stateId must exist
	hspth := "/hosts/hostid/instances/"+req.StateID()
	ok, hsEvt, err := conn.ExistsW(hspth, done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// Run listener
	shutdown := make(chan interface{})
	go func() {
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	// Delete service stateId
	go func() {
		time.Sleep(2*time.Second) // wait a little to ensure listener is running
		err = conn.Delete(sspth)
		c.Assert(err, IsNil)
	}()

	timer := time.NewTimer(5*time.Second)
	// Wait for the event indicating service stateId was deleted
	select {
	case e := <-hsEvt:
		c.Assert(e.Type, Equals, client.EventNodeDeleted)
		ok, err := conn.Exists(sspth)
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Host StateId not cleaned up")
	}
	timer.Reset(2*time.Second)
	select {
		case <-done:
			c.Logf("Listener shut down")
		case <-timer.C:
			c.Fatalf("Listener did not shutdown")
			close(shutdown)
	}
	handler.AssertExpectations(c)
}

// Test Case: Error on attach
func (t *ZZKTest) TestHostStateListener_Spawn_ErrAttach(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})

	handler.On("StopContainer", serviceId, 1).Return(nil)
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), serviceId, 1).Return(nil, ErrTestNoAttach)

	listener := NewHostStateListener(handler, hostId)
	listener.SetConnection(conn)

	done := make(chan struct{})
	ok, ev, err := conn.ExistsW("/hosts/hostid/instances/"+req.StateID(), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	go func() {
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Logf("Listener cleaned up orphaned node")
		timer.Reset(time.Second)

		select {
		case <-done:
			c.Logf("Listener shut down")
		case <-timer.C:
			c.Fatalf("Listener did not shut down")
		}
	case <-done:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/hosts/hostid/instances/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Listener did not shut down")
	}
	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a running container
func (t *ZZKTest) TestHostStateListener_Spawn_AttachRun(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

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
	listener := NewHostStateListener(handler, hostId)
	listener.SetConnection(conn)

	done := make(chan struct{})
	sspth := "/services/serviceid/" + req.StateID()
	ev, err := conn.GetW(sspth, &ServiceState{}, done)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	go func() {
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	select {
	case <-ev:
		c.Errorf("service state changed unexpectedly")
	case <-time.After(time.Second):
	}
	close(shutdown)
	select {
	case e := <-ev:
		c.Assert(e.Type, Equals, client.EventNodeDeleted)
	case <-done:
		c.Logf("Listener shut down, checking orphaned node deletion")
		ok, err := conn.Exists("/services/serviceid/" + req.StateID())
		c.Assert(err, IsNil)
		c.Check(ok, Equals, false)
	case <-time.After(5 * time.Second):
		c.Fatalf("state not deleted")
	}
	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a paused running container
func (t *ZZKTest) TestHostStateListener_Spawn_AttachResume(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}

	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, hostId)
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
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener shutdown")
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
				c.Fatalf("Listener shutdown")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		}

		c.Check(ssdat.Paused, Equals, false)
	case <-done:
		c.Fatalf("Listener shutdown")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	})
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case e := <-ev:
		c.Check(e.Type, Equals, client.EventNodeDeleted)
	case <-done:
		c.Logf("Listener shutdown")
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

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, hostId)
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
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener shutdown")
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
				c.Fatalf("Listener shutdown")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		}
		ssdat2.SetVersion(nil)
		c.Check(ssdat2.Terminated.IsZero(), Equals, true)
		ssdat2.Terminated = ssdat.Terminated
		c.Check(ssdat2.Started.Equal(ssdat.Started), Equals, true)
		ssdat2.Started = ssdat.Started
		c.Check(ssdat2, DeepEquals, ssdat)
	case <-done:
		c.Fatalf("Listener shutdown")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(args mock.Arguments) { containerExit <- time.Now() }).Once()

	close(shutdown)
	timer = time.NewTimer(time.Second)
	select {
	case e := <-ev:
		c.Check(e.Type, Equals, client.EventNodeDeleted)
	case <-done:
		c.Logf("Listener shutdown")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}
	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a running container and pauses the state
func (t *ZZKTest) TestHostStateListener_Spawn_AttachPause(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, hostId)
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
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener shutdown")
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
				c.Fatalf("Listener shutdown")
			case <-timer.C:
				c.Fatalf("Listener took too long")
			}
		}

		c.Check(ssdat.Paused, Equals, true)
	case <-done:
		c.Fatalf("Listener shutdown")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}

	// shutdown
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) { containerExit <- time.Now() }).Once()
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case e := <-ev:
		c.Check(e.Type, Equals, client.EventNodeDeleted)
		//c.Check(e.Type, Equals, client.EventNodeDataChanged)
	case <-done:
		c.Logf("Listener shutdown")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}
	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a paused running container (no change)
func (t *ZZKTest) TestHostStateListener_Spawn_AttachPausePaused(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, hostId)
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
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener shutdown")
	case <-timer.C:
	}

	// shutdown
	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	}).Once()
	close(shutdown)
	timer = time.NewTimer(time.Second)
	select {
	case e := <-ev:
		c.Check(e.Type, Equals, client.EventNodeDeleted)
	case <-done:
		c.Logf("Listener shutdown")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}
	handler.AssertExpectations(c)
}

// Test Case: Listener to pause a stopped container (no change)
func (t *ZZKTest) TestHostStateListener_Spawn_DetachPause(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, hostId)
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
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		c.Fatalf("Unexpected event from service state")
	case <-done:
		c.Fatalf("Listener shutdown")
	case <-timer.C:
	}

	// shutdown
	handler.On("StopContainer", serviceId, 1).Return(nil).Once()
	close(shutdown)
	timer = time.NewTimer(time.Second)
	select {
	case e := <-ev:
		c.Check(e.Type, Equals, client.EventNodeDeleted)
	case <-done:
		c.Logf("Listener shutdown")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}
	handler.AssertExpectations(c)
}

// Test Case: Listener attaches to a running container and stops
func (t *ZZKTest) TestHostStateListener_Spawn_AttachStop(c *C) {

	conn := setUpServiceAndHostPaths(c)
	handler := &mocks.HostStateHandler{}

	req := StateRequest{
		HostID:     hostId,
		ServiceID:  serviceId,
		InstanceID: 1,
	}
	err := CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, hostId)
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

	done := make(chan struct{})

	ev, err := conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	handler.On("StopContainer", serviceId, 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	})
	timer := time.NewTimer(time.Second)
	select {
	case e := <-ev:
		c.Check(e.Type, Equals, client.EventNodeDeleted)
	case <-done:
		c.Logf("Listener shutdown")
	case <-timer.C:
		c.Fatalf("Listener took too long")
	}
	handler.AssertExpectations(c)
}
