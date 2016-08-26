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
	. "github.com/control-center/serviced/zzk/service2"
	"github.com/control-center/serviced/zzk/service2/mocks"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

var (
	ErrTestNoAttach = errors.New("could not attach to container")
)

// Test Case: Bad state id
func (t *ZZKTest) TestHostStateListener_Spawn_BadStateID(c *C) {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	listener := NewHostStateListener(handler, "hostid")
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
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	// delete the host state
	handler.On("StopContainer", "serviceid", 1).Return(nil)
	err = conn.Delete("/hosts/hostid/instances/" + req.StateID())
	c.Assert(err, IsNil)

	listener := NewHostStateListener(handler, "hostid")
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
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	// delete the service state
	handler.On("StopContainer", "serviceid", 1).Return(nil)
	err = conn.Delete("/services/serviceid/" + req.StateID())
	c.Assert(err, IsNil)

	listener := NewHostStateListener(handler, "hostid")
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

// Test Case: Error on attach
func (t *ZZKTest) TestHostStateListener_Spawn_ErrAttach(c *C) {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})

	handler.On("StopContainer", "serviceid", 1).Return(nil)
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), "serviceid", 1).Return(nil, ErrTestNoAttach)

	listener := NewHostStateListener(handler, "hostid")
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
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	// attached, run, no change
	ssdat := ServiceState{
		ContainerID: "containerid",
		ImageID:     "imageid",
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
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), "serviceid", 1).Return(retExit, nil).Once()
	handler.On("StopContainer", "serviceid", 1).Return(nil).Run(func(_ mock.Arguments) {
		containerExit <- time.Now()
	})
	listener := NewHostStateListener(handler, "hostid")
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
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, "hostid")
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: "containerid",
		ImageID:     "imageid",
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
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), "serviceid", 1).Return(retExit, nil).Once()

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
	handler.On("ResumeContainer", mock.AnythingOfType("*service.Service"), 1).Return(nil)
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
	handler.On("StopContainer", "serviceid", 1).Return(nil).Run(func(_ mock.Arguments) {
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
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, "hostid")
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: "containerid",
		ImageID:     "imageid",
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
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), "serviceid", 1).Return(retExit, nil).Once()

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
		ContainerID: "containerid2",
		ImageID:     "imageid",
		Paused:      false,
		Started:     time.Now(),
	}
	var retShutdown <-chan interface{} = shutdown
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), "serviceid", 1).Return(nil, nil).Once()
	handler.On("StartContainer", retShutdown, mock.AnythingOfType("*service.Service"), 1).Return(ssdat, retExit, nil)

	containerExit <- time.Now()
	timer.Reset(time.Second)
	select {
	case <-ev:
		// state is either terminated or overwritten
		ssdat2 := &ServiceState{}
		ev, err = conn.GetW("/services/serviceid/"+req.StateID(), ssdat2, done)
		c.Assert(err, IsNil)

		if ssdat2.ContainerID == "containerid" {
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

	handler.On("StopContainer", "serviceid", 1).Return(nil).Run(func(_ mock.Arguments) {
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

// Test Case: Listener attaches to a running container and pauses the state
func (t *ZZKTest) TestHostStateListener_Spawn_AttachPause(c *C) {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, "hostid")
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: "containerid",
		ImageID:     "imageid",
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
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), "serviceid", 1).Return(retExit, nil).Once()

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

	// pause container
	handler.On("PauseContainer", mock.AnythingOfType("*service.Service"), 1).Return(nil)
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
	handler.On("StopContainer", "serviceid", 1).Return(nil).Run(func(_ mock.Arguments) {
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

// Test Case: Listener attaches to a paused running container (no change)
func (t *ZZKTest) TestHostStateListener_Spawn_AttachPausePaused(c *C) {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, "hostid")
	listener.SetConnection(conn)

	// set up a paused running container
	ssdat := &ServiceState{
		ContainerID: "containerid",
		ImageID:     "imageid",
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
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), "serviceid", 1).Return(retExit, nil).Once()

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
	handler.On("StopContainer", "serviceid", 1).Return(nil).Run(func(_ mock.Arguments) {
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

// Test Case: Listener to pause a stopped container (no change)
func (t *ZZKTest) TestHostStateListener_Spawn_DetachPause(c *C) {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, "hostid")
	listener.SetConnection(conn)

	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCPause
		return true
	})
	c.Assert(err, IsNil)
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), "serviceid", 1).Return(nil, nil).Once()

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
	handler.On("StopContainer", "serviceid", 1).Return(nil)
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

// Test Case: Listener attaches to a running container and stops
func (t *ZZKTest) TestHostStateListener_Spawn_AttachStop(c *C) {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.HostStateHandler{}

	// Basic set up
	svc := &service.Service{
		ID:     "serviceid",
		Name:   "serviceA",
		PoolID: "poolid",
	}
	spth := "/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	listener := NewHostStateListener(handler, "hostid")
	listener.SetConnection(conn)

	// set up a running container
	ssdat := &ServiceState{
		ContainerID: "containerid",
		ImageID:     "imageid",
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
	handler.On("AttachContainer", mock.AnythingOfType("*service.ServiceState"), "serviceid", 1).Return(retExit, nil).Once()

	done := make(chan struct{})

	ev, err := conn.GetW("/services/serviceid/"+req.StateID(), ssdat, done)
	c.Assert(err, IsNil)
	go func() {
		listener.Spawn(shutdown, req.StateID())
		close(done)
	}()

	handler.On("StopContainer", "serviceid", 1).Return(nil).Run(func(_ mock.Arguments) {
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
