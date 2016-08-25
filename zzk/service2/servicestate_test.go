// Copyright 2016 The Serviced Authors.
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

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/service2"
	"github.com/control-center/serviced/zzk/service2/mocks"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

var (
	ErrTestHostNotFound = errors.New("host not found")
)

func (t *ZZKTest) TestServiceListener_Spawn(c *C) {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.ServiceHandler{}

	// Basic set up
	svc := &service.Service{
		ID:           "serviceid",
		Name:         "serviceA",
		DesiredState: int(service.SVCStop),
		Instances:    1,
	}
	spth := "/pools/poolid/services/serviceid"
	sdat := &ServiceNode{Service: svc}
	err = conn.Create(spth, sdat)
	c.Assert(err, IsNil)

	// an online host
	err = conn.CreateDir("/pools/poolid/hosts/hostid/online/online")
	c.Assert(err, IsNil)
	handler.On("SelectHost", mock.AnythingOfType("*service.Service")).Return("hostid", "", nil)

	listener := NewServiceListener("poolid", handler)
	listener.SetConnection(conn)

	shutdown := make(chan interface{})
	defer func() { close(shutdown) }()

	done := make(chan struct{})
	go func() {
		listener.Spawn(shutdown, "serviceid")
		close(done)
	}()

	// run
	ch, ev, err := conn.ChildrenW(spth, done)
	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 0)

	sdat = &ServiceNode{Service: &service.Service{}}
	err = conn.Get(spth, sdat)
	c.Assert(err, IsNil)
	sdat.DesiredState = int(service.SVCRun)
	err = conn.Set(spth, sdat)
	c.Assert(err, IsNil)
	c.Logf("Set state to run")

	timer := time.NewTimer(5 * time.Second)

	select {
	case <-ev:
		time.Sleep(500 * time.Millisecond) // Lag on events with create
		ch, err = conn.Children(spth)
		c.Assert(err, IsNil)
		c.Check(ch, HasLen, 1)
	case <-done:
		c.Fatalf("listener exited")
	case <-timer.C:
		c.Errorf("listener timed out")
	}

	// pause
	hspth := "/pools/poolid/hosts/hostid/instances/hostid-serviceid-0"
	hsdat := &HostState{}
	ev, err = conn.GetW(hspth, hsdat, done)
	c.Assert(err, IsNil)
	c.Assert(hsdat.DesiredState, Equals, service.SVCRun)

	sdat = &ServiceNode{Service: &service.Service{}}
	err = conn.Get(spth, sdat)
	c.Assert(err, IsNil)
	sdat.DesiredState = int(service.SVCPause)
	err = conn.Set(spth, sdat)
	c.Assert(err, IsNil)
	c.Logf("Set state to pause")

	timer.Reset(5 * time.Second)

	select {
	case <-ev:
		ev, err = conn.GetW(hspth, hsdat, done)
		c.Assert(err, IsNil)
		c.Check(hsdat.DesiredState, Equals, service.SVCPause)
	case <-done:
		c.Fatalf("listener exited")
	case <-timer.C:
		c.Errorf("listener timed out")
	}

	// resume
	sdat = &ServiceNode{Service: &service.Service{}}
	err = conn.Get(spth, sdat)
	c.Assert(err, IsNil)
	sdat.DesiredState = int(service.SVCRun)
	err = conn.Set(spth, sdat)
	c.Assert(err, IsNil)
	c.Logf("Set state to resume")

	timer.Reset(5 * time.Second)

	select {
	case <-ev:
		ev, err = conn.GetW(hspth, hsdat, done)
		c.Assert(err, IsNil)
		c.Check(hsdat.DesiredState, Equals, service.SVCRun)
	case <-done:
		c.Fatalf("listener exited")
	case <-timer.C:
		c.Errorf("listener timed out")
	}

	// stop
	sdat = &ServiceNode{Service: &service.Service{}}
	err = conn.Get(spth, sdat)
	c.Assert(err, IsNil)
	sdat.DesiredState = int(service.SVCStop)
	err = conn.Set(spth, sdat)
	c.Assert(err, IsNil)
	c.Logf("Set state to stop")

	timer.Reset(5 * time.Second)

	select {
	case <-ev:
		err = conn.Get(hspth, hsdat)
		c.Assert(err, IsNil)
		c.Check(hsdat.DesiredState, Equals, service.SVCStop)
	case <-done:
		c.Fatalf("listener exited")
	case <-timer.C:
		c.Errorf("listener timed out")
	}

	// shutdown
	close(shutdown)
	c.Logf("Shutting down listener")

	timer.Reset(5 * time.Second)

	select {
	case <-done:
	case <-timer.C:
		c.Errorf("listener timed out")
	}

	// delete
	shutdown = make(chan interface{})

	done = make(chan struct{})
	go func() {
		listener.Spawn(shutdown, "serviceid")
		close(done)
	}()

	err = conn.Delete(spth)
	c.Assert(err, IsNil)
	c.Logf("Deleted service")

	timer.Reset(5 * time.Second)

	select {
	case <-done:
	case <-timer.C:
		c.Errorf("listener timed out")
	}
}

func (t *ZZKTest) TestServiceListener_Sync_Unlocked(c *C) {

	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.ServiceHandler{}

	// Basic set up
	svc := &service.Service{
		ID:        "serviceid",
		Name:      "serviceA",
		Instances: 1,
	}
	err = conn.Create("/pools/poolid/services/serviceid", &ServiceNode{Service: svc})
	c.Assert(err, IsNil)

	// an online host
	err = conn.CreateDir("/pools/poolid/hosts/hostid/online/online")
	c.Assert(err, IsNil)
	handler.On("SelectHost", svc).Return("hostid", "", nil)

	listener := NewServiceListener("poolid", handler)
	listener.SetConnection(conn)

	// start
	svc.Instances = 10
	reqs := []StateRequest{}

	delta, ok := listener.Sync(false, svc, reqs)
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 10)

	for i := 0; i < 10; i++ {
		req := StateRequest{
			PoolID:     "poolid",
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: i,
		}
		ok, err := IsValidState(conn, req)
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}

	// start again but delete some states
	deleted := []bool{true, true, false, false, false, false, true, false, true, true}
	reqs = []StateRequest{}
	for i, ok := range deleted {
		req := StateRequest{
			PoolID:     "poolid",
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: i,
		}
		if ok {
			err = DeleteState(conn, req)
			c.Assert(err, IsNil)
		} else {
			reqs = append(reqs, req)
		}
	}

	delta, ok = listener.Sync(false, svc, reqs)
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 5)

	for i := 0; i < 10; i++ {
		req := StateRequest{
			PoolID:     "poolid",
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: i,
		}
		ok, err := IsValidState(conn, req)
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}

	// stop
	svc.Instances = 5
	reqs = make([]StateRequest, 10)
	for i := range reqs {
		req := StateRequest{
			PoolID:     "poolid",
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: i,
		}
		reqs[i] = req
	}
	delta, ok = listener.Sync(false, svc, reqs)
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, -5)

	for i := 0; i < 10; i++ {
		req := StateRequest{
			PoolID:     "poolid",
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: i,
		}
		ok, err := IsValidState(conn, req)
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}

	// stop again
	delta, ok = listener.Sync(false, svc, reqs)
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, -5)

	for i := 0; i < 10; i++ {
		req := StateRequest{
			PoolID:     "poolid",
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: i,
		}
		ok, err := IsValidState(conn, req)
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}

	// no change
	deleted = []bool{false, false, false, false, false, true, true, true, true, true}
	reqs = []StateRequest{}
	for i, ok := range deleted {
		req := StateRequest{
			PoolID:     "poolid",
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: i,
		}
		if ok {
			err = DeleteState(conn, req)
			c.Assert(err, IsNil)
		} else {
			reqs = append(reqs, req)
		}
	}

	delta, ok = listener.Sync(false, svc, reqs)
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 0)
}

func (t *ZZKTest) TestServiceListener_Sync_Locked(c *C) {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.ServiceHandler{}

	// Basic set up
	svc := &service.Service{
		ID:        "serviceid",
		Name:      "serviceA",
		Instances: 1,
	}
	err = conn.Create("/pools/poolid/services/serviceid", &ServiceNode{Service: svc})
	c.Assert(err, IsNil)

	// an online host
	err = conn.CreateDir("/pools/poolid/hosts/hostid/online/online")
	c.Assert(err, IsNil)
	handler.On("SelectHost", svc).Return("hostid", "", nil)

	listener := NewServiceListener("poolid", handler)
	listener.SetConnection(conn)

	// start 5 instances
	svc.Instances = 5
	reqs := []StateRequest{}

	delta, ok := listener.Sync(false, svc, reqs)
	c.Assert(ok, Equals, true)
	c.Assert(delta, Equals, 5)

	// start 10 instances
	svc.Instances = 10

	reqs = make([]StateRequest, 5)
	for i := 0; i < 5; i++ {
		req := StateRequest{
			PoolID:     "poolid",
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: i,
		}
		ok, err := IsValidState(conn, req)
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
		reqs[i] = req
	}

	delta, ok = listener.Sync(true, svc, reqs)
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 0)

	// stop 2 instances
	svc.Instances = 3
	delta, ok = listener.Sync(true, svc, reqs)
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, -2)
}

func (t *ZZKTest) TestServiceListener_Sync_RestartAllOnInstanceChanged(c *C) {
	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.ServiceHandler{}

	// Basic set up
	svc := &service.Service{
		ID:            "serviceid",
		Name:          "serviceA",
		Instances:     1,
		ChangeOptions: []string{"restartAllOnInstanceChanged"},
	}
	err = conn.Create("/pools/poolid/services/serviceid", &ServiceNode{Service: svc})
	c.Assert(err, IsNil)

	// an online host
	err = conn.CreateDir("/pools/poolid/hosts/hostid/online/online")
	c.Assert(err, IsNil)
	handler.On("SelectHost", svc).Return("hostid", "", nil)

	listener := NewServiceListener("poolid", handler)
	listener.SetConnection(conn)

	// start 5 instances
	svc.Instances = 5
	reqs := []StateRequest{}

	delta, ok := listener.Sync(false, svc, reqs)
	c.Assert(ok, Equals, true)
	c.Assert(delta, Equals, 5)

	// delete 1 instance and sync
	deleted := []bool{false, false, true, false, false}
	reqs = []StateRequest{}
	for i, ok := range deleted {
		req := StateRequest{
			PoolID:     "poolid",
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: i,
		}
		if ok {
			err = DeleteState(conn, req)
			c.Assert(err, IsNil)
		} else {
			reqs = append(reqs, req)
		}
	}

	delta, ok = listener.Sync(false, svc, reqs)
	c.Assert(ok, Equals, true)
	c.Assert(delta, Equals, -4)
}

func (t *ZZKTest) TestServiceListener_Start(c *C) {

	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.ServiceHandler{}

	// Basic set up
	svc := &service.Service{
		ID:        "serviceid",
		Name:      "serviceA",
		Instances: 1,
	}
	err = conn.Create("/pools/poolid/services/serviceid", &ServiceNode{Service: svc})
	c.Assert(err, IsNil)
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	listener := NewServiceListener("poolid", handler)
	listener.SetConnection(conn)

	// no host
	handler.On("SelectHost", svc).Return("", "", ErrTestHostNotFound).Once()
	c.Assert(listener.Start(svc, 0), Equals, false)

	handler.On("SelectHost", svc).Return("hostid", "", nil)

	// host state exists
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 0,
	}
	err = conn.CreateDir("/pools/poolid/hosts/hostid/instances/" + req.StateID())
	c.Assert(err, IsNil)
	c.Check(listener.Start(svc, 0), Equals, true)

	// host state does not exist
	c.Check(listener.Start(svc, 1), Equals, true)
}

func (t *ZZKTest) TestServiceListener_Stop_Offline(c *C) {

	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.ServiceHandler{}

	// Basic set up
	svc := &service.Service{
		ID:        "serviceid",
		Name:      "serviceA",
		Instances: 1,
	}
	err = conn.Create("/pools/poolid/services/serviceid", &ServiceNode{Service: svc})
	c.Assert(err, IsNil)

	// an offline host
	err = conn.CreateDir("/pools/poolid/hosts/hostid2")
	c.Assert(err, IsNil)

	listener := NewServiceListener("poolid", handler)
	listener.SetConnection(conn)

	// state does not exist
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	delta, ok := listener.Stop([]StateRequest{req})
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 1)

	// missing host state
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid",
		InstanceID: 3,
	}
	err = conn.CreateDir("/pools/poolid/services/serviceid/" + req.StateID())
	delta, ok = listener.Stop([]StateRequest{req})
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 1)

	// missing service state
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid",
		InstanceID: 5,
	}
	err = conn.CreateDir("/pools/poolid/hosts/hostid2/instances/" + req.StateID())
	delta, ok = listener.Stop([]StateRequest{req})
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 1)

	// state exists
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid",
		InstanceID: 6,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)
	delta, ok = listener.Stop([]StateRequest{req})
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 1)
}

func (t *ZZKTest) TestServiceListener_Stop_Online(c *C) {

	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.ServiceHandler{}

	// Basic set up
	svc := &service.Service{
		ID:        "serviceid",
		Name:      "serviceA",
		Instances: 1,
	}
	err = conn.Create("/pools/poolid/services/serviceid", &ServiceNode{Service: svc})
	c.Assert(err, IsNil)

	// an online host
	err = conn.CreateDir("/pools/poolid/hosts/hostid1/online/online")
	c.Assert(err, IsNil)

	listener := NewServiceListener("poolid", handler)
	listener.SetConnection(conn)

	// state does not exist
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid",
		InstanceID: 0,
	}
	delta, ok := listener.Stop([]StateRequest{req})
	c.Assert(ok, Equals, false)
	c.Assert(delta, Equals, 0)

	// missing host state
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid",
		InstanceID: 2,
	}
	err = conn.CreateDir("/pools/poolid/services/serviceid/" + req.StateID())
	delta, ok = listener.Stop([]StateRequest{req})
	c.Assert(ok, Equals, false)
	c.Assert(delta, Equals, 0)

	//  missing service state
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid",
		InstanceID: 4,
	}
	err = conn.CreateDir("/pools/poolid/hosts/hostid1/instances/" + req.StateID())
	delta, ok = listener.Stop([]StateRequest{req})
	c.Assert(ok, Equals, false)
	c.Assert(delta, Equals, 0)

	// state is not stopped
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid",
		InstanceID: 6,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)
	delta, ok = listener.Stop([]StateRequest{req})
	c.Assert(ok, Equals, true)
	c.Assert(delta, Equals, 1)

	// state is stopped
	delta, ok = listener.Stop([]StateRequest{req})
	c.Assert(ok, Equals, true)
	c.Assert(delta, Equals, 1)
}

func (t *ZZKTest) TestServiceListener_Pause(c *C) {

	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.ServiceHandler{}

	// Basic set up
	svc := &service.Service{
		ID:        "serviceid",
		Name:      "serviceA",
		Instances: 1,
	}
	err = conn.Create("/pools/poolid/services/serviceid", &ServiceNode{Service: svc})
	c.Assert(err, IsNil)
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	listener := NewServiceListener("poolid", handler)
	listener.SetConnection(conn)

	// state does not exist
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 0,
	}
	delta, ok := listener.Pause([]StateRequest{req})
	c.Check(ok, Equals, false)
	c.Check(delta, Equals, 0)

	// missing host state
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = conn.CreateDir("/pools/poolid/services/serviceid/" + req.StateID())
	c.Assert(err, IsNil)

	delta, ok = listener.Pause([]StateRequest{req})
	c.Check(ok, Equals, false)
	c.Check(delta, Equals, 0)

	// missing service state
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 2,
	}
	err = conn.CreateDir("/pools/poolid/hosts/hostid/instances/" + req.StateID())
	c.Assert(err, IsNil)

	delta, ok = listener.Pause([]StateRequest{req})
	c.Check(ok, Equals, false)
	c.Check(delta, Equals, 0)

	// state is running
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 3,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	delta, ok = listener.Pause([]StateRequest{req})
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 1)

	// state is not running
	delta, ok = listener.Pause([]StateRequest{req})
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 1)
}

func (t *ZZKTest) TestServiceListener_Resume(c *C) {

	// Pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	handler := &mocks.ServiceHandler{}

	// Basic set up
	svc := &service.Service{
		ID:        "serviceid",
		Name:      "serviceA",
		Instances: 1,
	}
	err = conn.Create("/pools/poolid/services/serviceid", &ServiceNode{Service: svc})
	c.Assert(err, IsNil)
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	listener := NewServiceListener("poolid", handler)
	listener.SetConnection(conn)

	// state does not exist
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 0,
	}
	delta, ok := listener.Resume([]StateRequest{req})
	c.Check(ok, Equals, false)
	c.Check(delta, Equals, 0)

	// missing host state
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = conn.CreateDir("/pools/poolid/services/serviceid/" + req.StateID())
	c.Assert(err, IsNil)

	delta, ok = listener.Resume([]StateRequest{req})
	c.Check(ok, Equals, false)
	c.Check(delta, Equals, 0)

	// missing service state
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 2,
	}
	err = conn.CreateDir("/pools/poolid/hosts/hostid/instances/" + req.StateID())
	c.Assert(err, IsNil)

	delta, ok = listener.Resume([]StateRequest{req})
	c.Check(ok, Equals, false)
	c.Check(delta, Equals, 0)

	// state is paused
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 3,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCPause
		return true
	})
	c.Assert(err, IsNil)

	delta, ok = listener.Resume([]StateRequest{req})
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 1)

	// state is not paused
	delta, ok = listener.Resume([]StateRequest{req})
	c.Check(ok, Equals, true)
	c.Check(delta, Equals, 1)
}
