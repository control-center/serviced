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
	"sort"
	"time"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/service2"
	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestParseStateID(c *C) {
	// invalid id
	hostID, serviceID, inst, err := ParseStateID("badaadadadafg")
	c.Assert(err, Equals, ErrInvalidStateID)
	c.Assert(hostID, Equals, "")
	c.Assert(serviceID, Equals, "")
	c.Assert(inst, Equals, 0)

	// another invalid id
	hostID, serviceID, inst, err = ParseStateID("dfgsgsg-1")
	c.Assert(err, Equals, ErrInvalidStateID)
	c.Assert(hostID, Equals, "")
	c.Assert(serviceID, Equals, "")
	c.Assert(inst, Equals, 0)

	// yet another invalid id
	hostID, serviceID, inst, err = ParseStateID("rg35g34-dfrhedfbsd-de4")
	c.Assert(err, Equals, ErrInvalidStateID)
	c.Assert(hostID, Equals, "")
	c.Assert(serviceID, Equals, "")
	c.Assert(inst, Equals, 0)

	// an acceptable id
	hostID, serviceID, inst, err = ParseStateID("45grwg34-fgrg43g5heefv-5")
	c.Assert(err, IsNil)
	c.Assert(hostID, Equals, "45grwg34")
	c.Assert(serviceID, Equals, "fgrg43g5heefv")
	c.Assert(inst, Equals, 5)
}

func (t *ZZKTest) TestGetServiceStates(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add 2 services
	err = conn.CreateDir("/pools/poolid/services/serviceid1")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/pools/poolid/services/serviceid2")
	c.Assert(err, IsNil)

	// add 2 hosts
	err = conn.CreateDir("/pools/poolid/hosts/hostid1")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/pools/poolid/hosts/hostid2")
	c.Assert(err, IsNil)

	// create states
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid1",
		InstanceID: 1,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid2",
		InstanceID: 2,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid2",
		InstanceID: 3,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	// 0 states
	states, err := GetServiceStates(conn, "poolid", "serviceid0")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 0)

	// =1 state
	states, err = GetServiceStates(conn, "poolid", "serviceid1")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 1)
	c.Assert(states[0].InstanceID, Equals, 1)

	// >1 state
	states, err = GetServiceStates(conn, "poolid", "serviceid2")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 2)
	actual := []int{states[0].InstanceID, states[1].InstanceID}
	sort.Ints(actual)
	c.Assert(actual, DeepEquals, []int{2, 3})
}

func (t *ZZKTest) TestDeleteServiceStates(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add 2 services
	err = conn.CreateDir("/pools/poolid/services/serviceid1")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/pools/poolid/services/serviceid2")
	c.Assert(err, IsNil)

	// add 2 hosts
	err = conn.CreateDir("/pools/poolid/hosts/hostid1")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/pools/poolid/hosts/hostid2")
	c.Assert(err, IsNil)

	// create states
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid1",
		InstanceID: 1,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid2",
		InstanceID: 2,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid2",
		InstanceID: 3,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	// 0 states
	count := DeleteServiceStates(conn, "poolid", "serviceid0")
	c.Check(count, Equals, 0)

	// =1 state
	count = DeleteServiceStates(conn, "poolid", "serviceid1")
	c.Check(count, Equals, 1)

	// >1 state
	count = DeleteServiceStates(conn, "poolid", "serviceid2")
	c.Check(count, Equals, 2)
}

func (t *ZZKTest) TestGetHostStates(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add 2 services
	err = conn.CreateDir("/pools/poolid/services/serviceid1")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/pools/poolid/services/serviceid2")
	c.Assert(err, IsNil)

	// add 2 hosts
	err = conn.CreateDir("/pools/poolid/hosts/hostid1")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/pools/poolid/hosts/hostid2")
	c.Assert(err, IsNil)

	// create states
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid1",
		InstanceID: 1,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid1",
		InstanceID: 2,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid2",
		InstanceID: 3,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	// 0 states
	states, err := GetHostStates(conn, "poolid", "hostid0")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 0)

	// =1 state
	states, err = GetHostStates(conn, "poolid", "hostid1")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 1)
	c.Assert(states[0].InstanceID, Equals, 1)

	// >1 state
	states, err = GetHostStates(conn, "poolid", "hostid2")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 2)
	actual := []int{states[0].InstanceID, states[1].InstanceID}
	sort.Ints(actual)
	c.Assert(actual, DeepEquals, []int{2, 3})
}

func (t *ZZKTest) TestDeleteHostStates(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add 2 services
	err = conn.CreateDir("/pools/poolid/services/serviceid1")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/pools/poolid/services/serviceid2")
	c.Assert(err, IsNil)

	// add 2 hosts
	err = conn.CreateDir("/pools/poolid/hosts/hostid1")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/pools/poolid/hosts/hostid2")
	c.Assert(err, IsNil)

	// create states
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid1",
		InstanceID: 1,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid1",
		InstanceID: 2,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid2",
		InstanceID: 3,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	// 0 states
	count := DeleteHostStates(conn, "poolid", "hostid0")
	c.Check(count, Equals, 0)

	// =1 state
	count = DeleteHostStates(conn, "poolid", "hostid1")
	c.Check(count, Equals, 1)

	// >1 state
	count = DeleteHostStates(conn, "poolid", "hostid2")
	c.Check(count, Equals, 2)
}

func (t *ZZKTest) TestIsValidState(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add a service
	err = conn.CreateDir("/pools/poolid/services/serviceid")
	c.Assert(err, IsNil)

	// add a host
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	// neither service nor host
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 0,
	}

	ok, err := IsValidState(conn, req)
	c.Check(err, IsNil)
	c.Check(ok, Equals, false)

	// service but no host
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = conn.CreateDir("/pools/poolid/services/serviceid/" + req.StateID())
	c.Assert(err, IsNil)
	ok, err = IsValidState(conn, req)
	c.Check(err, IsNil)
	c.Check(ok, Equals, false)

	// host but no service
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 2,
	}
	err = conn.CreateDir("/pools/poolid/hosts/hostid/instances/" + req.StateID())
	c.Assert(err, IsNil)
	ok, err = IsValidState(conn, req)
	c.Check(err, IsNil)
	c.Check(ok, Equals, false)

	// service and host
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 3,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)
	ok, err = IsValidState(conn, req)
	c.Check(err, IsNil)
	c.Check(ok, Equals, true)
}

func (t *ZZKTest) TestCleanHostStates(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add a service
	err = conn.CreateDir("/pools/poolid/services/serviceid")
	c.Assert(err, IsNil)

	// add a host
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	// bad state
	err = conn.CreateDir("/pools/poolid/hosts/hostid/instances/badstateid")
	c.Assert(err, IsNil)

	// incongruent state
	err = conn.CreateDir("/pools/poolid/hosts/hostid/instances/hostid-serviceid-0")
	c.Assert(err, IsNil)

	// good state
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	err = CleanHostStates(conn, "poolid", "hostid")
	c.Check(err, IsNil)

	states, err := GetHostStates(conn, "poolid", "hostid")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 1)
	c.Check(states[0].HostID, Equals, "hostid")
	c.Check(states[0].ServiceID, Equals, "serviceid")
	c.Check(states[0].InstanceID, Equals, 1)
}

func (t *ZZKTest) TestCleanServiceStates(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add a service
	err = conn.CreateDir("/pools/poolid/services/serviceid")
	c.Assert(err, IsNil)

	// add a host
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	// bad state
	err = conn.CreateDir("/pools/poolid/hosts/serviceid/badstateid")
	c.Assert(err, IsNil)

	// incongruent state
	err = conn.CreateDir("/pools/poolid/hosts/serviceid/hostid-serviceid-0")
	c.Assert(err, IsNil)

	// good state
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	err = CleanServiceStates(conn, "poolid", "serviceid")
	c.Check(err, IsNil)

	states, err := GetServiceStates(conn, "poolid", "serviceid")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 1)
	c.Check(states[0].HostID, Equals, "hostid")
	c.Check(states[0].ServiceID, Equals, "serviceid")
	c.Check(states[0].InstanceID, Equals, 1)
}

func (t *ZZKTest) TestMonitorState(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add a service
	err = conn.CreateDir("/pools/poolid/services/serviceid")
	c.Assert(err, IsNil)

	// add a host
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	// state does not exist
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 0,
	}

	shutdown := make(chan struct{})
	done := make(chan struct{})
	go func() {
		state, err := MonitorState(shutdown, conn, req, func(s *State, exists bool) bool {
			return false
		})

		_, ok := err.(*StateError)
		c.Check(ok, Equals, true)
		c.Check(state, IsNil)
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-done:
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for monitor")
	}

	// state does not exist, but that is okay
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 0,
	}

	shutdown = make(chan struct{})
	done = make(chan struct{})
	go func() {
		state, err := MonitorState(shutdown, conn, req, func(s *State, exists bool) bool {
			return true
		})

		c.Check(err, IsNil)
		expected := &State{
			HostID:     "hostid",
			ServiceID:  "serviceid",
			InstanceID: 0,
		}
		c.Check(state, DeepEquals, expected)
		close(done)
	}()

	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for monitor")
	}

	// host state does not exist
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 1,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)
	err = conn.Delete("/pools/poolid/hosts/hostid/instances/" + req.StateID())
	c.Assert(err, IsNil)

	shutdown = make(chan struct{})
	done = make(chan struct{})
	go func() {
		state, err := MonitorState(shutdown, conn, req, func(s *State, exists bool) bool {
			return false
		})

		_, ok := err.(*StateError)
		c.Check(ok, Equals, true)
		c.Check(state, IsNil)
		close(done)
	}()

	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for monitor")
	}

	// service state does not exist
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 2,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)
	err = conn.Delete("/pools/poolid/services/serviceid/" + req.StateID())
	c.Assert(err, IsNil)

	shutdown = make(chan struct{})
	done = make(chan struct{})
	go func() {
		state, err := MonitorState(shutdown, conn, req, func(s *State, exists bool) bool {
			return false
		})

		_, ok := err.(*StateError)
		c.Check(ok, Equals, true)
		c.Check(state, IsNil)
		close(done)
	}()

	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for monitor")
	}

	// shutdown
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 3,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	shutdown = make(chan struct{})
	done = make(chan struct{})
	go func() {
		state, err := MonitorState(shutdown, conn, req, func(s *State, exists bool) bool {
			return false
		})

		c.Check(err, IsNil)
		c.Check(state, IsNil)
		close(done)
	}()

	timer.Reset(time.Second)
	select {
	case <-done:
		c.Fatalf("Monitor exited prematurely")
	case <-timer.C:
		close(shutdown)
	}

	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("Timed out waiting for monitor")
	}

	// check passes
	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 4,
	}
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)

	shutdown = make(chan struct{})
	done = make(chan struct{})
	go func() {
		state, err := MonitorState(shutdown, conn, req, func(s *State, exists bool) bool {
			return s.ContainerID == "dockerid"
		})

		c.Check(err, IsNil)
		c.Check(state.ContainerID, Equals, "dockerid")
		close(done)
	}()

	timer.Reset(time.Second)
	select {
	case <-done:
		c.Fatalf("Monitor exited prematurely")
	case <-timer.C:
	}

	err = UpdateState(conn, req, func(s *State) bool {
		s.ContainerID = "dockerid"
		return true
	})
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("Timed out waiting for monitor")
	}
}

func (t *ZZKTest) TestCRUDState(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add a service
	err = conn.CreateDir("/pools/poolid/services/serviceid")
	c.Assert(err, IsNil)

	// add a host
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 3,
	}

	// create state
	startTime := time.Now()
	err = CreateState(conn, req, "")
	c.Assert(err, IsNil)
	ok, err := conn.Exists("/pools/poolid/services/serviceid/hostid-serviceid-3")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	ok, err = conn.Exists("/pools/poolid/hosts/hostid/instances/hostid-serviceid-3")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// create duplicate state
	err = CreateState(conn, req, "")
	stateErr, ok := err.(*StateError)
	c.Check(ok, Equals, true)
	c.Check(stateErr.Request, DeepEquals, req)
	c.Check(stateErr.Operation, Equals, "create")

	// state exists
	state, err := GetState(conn, req)
	c.Assert(err, IsNil)
	c.Check(state.ContainerID, Equals, "")
	c.Check(state.ImageID, Equals, "")
	c.Check(state.Paused, Equals, false)
	c.Check(startTime.Before(state.Started), Equals, false)
	c.Check(startTime.Before(state.Terminated), Equals, false)
	c.Check(state.DesiredState, Equals, service.SVCRun)
	c.Check(startTime.Before(state.Scheduled), Equals, true)
	c.Check(state.HostID, Equals, "hostid")
	c.Check(state.ServiceID, Equals, "serviceid")
	c.Check(state.InstanceID, Equals, 3)

	// update state (no commit)
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCPause
		s.ServiceState = ServiceState{
			ContainerID: "dockerid",
			ImageID:     "imageid",
			Paused:      true,
			Started:     time.Now(),
		}
		return false
	})
	c.Assert(err, IsNil)

	state, err = GetState(conn, req)
	c.Assert(err, IsNil)
	c.Check(state.ContainerID, Equals, "")
	c.Check(state.ImageID, Equals, "")
	c.Check(state.Paused, Equals, false)
	c.Check(startTime.Before(state.Started), Equals, false)
	c.Check(startTime.Before(state.Terminated), Equals, false)
	c.Check(state.DesiredState, Equals, service.SVCRun)
	c.Check(startTime.Before(state.Scheduled), Equals, true)
	c.Check(state.HostID, Equals, "hostid")
	c.Check(state.ServiceID, Equals, "serviceid")
	c.Check(state.InstanceID, Equals, 3)

	// update state (commit)
	err = UpdateState(conn, req, func(s *State) bool {
		s.DesiredState = service.SVCPause
		s.ServiceState = ServiceState{
			ContainerID: "dockerid",
			ImageID:     "imageid",
			Paused:      true,
			Started:     time.Now(),
		}
		return true
	})

	c.Assert(err, IsNil)
	state, err = GetState(conn, req)
	c.Assert(err, IsNil)
	c.Check(state.ContainerID, Equals, "dockerid")
	c.Check(state.ImageID, Equals, "imageid")
	c.Check(state.Paused, Equals, true)
	c.Check(startTime.Before(state.Started), Equals, true)
	c.Check(startTime.Before(state.Terminated), Equals, false)
	c.Check(state.DesiredState, Equals, service.SVCPause)
	c.Check(startTime.Before(state.Scheduled), Equals, true)
	c.Check(state.HostID, Equals, "hostid")
	c.Check(state.ServiceID, Equals, "serviceid")
	c.Check(state.InstanceID, Equals, 3)

	// delete state
	err = DeleteState(conn, req)
	c.Assert(err, IsNil)

	state, err = GetState(conn, req)
	stateErr, ok = err.(*StateError)
	c.Check(ok, Equals, true)
	c.Check(stateErr.Request, DeepEquals, req)
	c.Check(stateErr.Operation, Equals, "get")
	c.Assert(state, IsNil)
}
