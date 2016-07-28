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

package service

import (
	"sort"
	"time"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
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

func (t *ZZKTest) TestGetServiceStates2(c *C) {
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
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid2",
		InstanceID: 2,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid2",
		InstanceID: 3,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	// 0 states
	states, err := GetServiceStates2(conn, "poolid", "serviceid0")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 0)

	// =1 state
	states, err = GetServiceStates2(conn, "poolid", "serviceid1")
	c.Assert(err, IsNil)
	c.Assert(states, HasLen, 1)
	c.Assert(states[0].InstanceID, Equals, 1)

	// >1 state
	states, err = GetServiceStates2(conn, "poolid", "serviceid2")
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
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid1",
		ServiceID:  "serviceid2",
		InstanceID: 2,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid2",
		InstanceID: 3,
	}
	err = CreateState(conn, req)
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
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid1",
		InstanceID: 2,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid2",
		InstanceID: 3,
	}
	err = CreateState(conn, req)
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
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid1",
		InstanceID: 2,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	req = StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid2",
		ServiceID:  "serviceid2",
		InstanceID: 3,
	}
	err = CreateState(conn, req)
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
	err = CreateState(conn, req)
	c.Assert(err, IsNil)
	ok, err := conn.Exists("/pools/poolid/services/serviceid/hostid-serviceid-3")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	ok, err = conn.Exists("/pools/poolid/hosts/hostid/instances/hostid-serviceid-3")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// create duplicate state
	err = CreateState(conn, req)
	stateErr, ok := err.(*StateError)
	c.Check(ok, Equals, true)
	c.Check(stateErr.Request, DeepEquals, req)
	c.Check(stateErr.Operation, Equals, "create")

	// state exists
	state, err := GetState(conn, req)
	c.Assert(err, IsNil)
	c.Check(state.DockerID, Equals, "")
	c.Check(state.ImageID, Equals, "")
	c.Check(state.Paused, Equals, false)
	c.Check(startTime.Before(state.Started), Equals, false)
	c.Check(startTime.Before(state.Terminated), Equals, false)
	c.Check(state.DesiredState, Equals, service.SVCRun)
	c.Check(startTime.Before(state.Scheduled), Equals, true)
	c.Check(state.HostID, Equals, "hostid")
	c.Check(state.ServiceID, Equals, "serviceid")
	c.Check(state.InstanceID, Equals, 3)

	// update state
	err = UpdateState(conn, req, func(h *HostState2, s *ServiceState) {
		h.DesiredState = service.SVCPause
		s.DockerID = "dockerid"
		s.ImageID = "imageid"
		s.Paused = true
		s.Started = time.Now()
	})
	c.Assert(err, IsNil)

	state, err = GetState(conn, req)
	c.Assert(err, IsNil)
	c.Check(state.DockerID, Equals, "dockerid")
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
