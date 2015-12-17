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
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"

	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestHostRegistryListener_Spawn(c *C) {
	conn, err := zzk.GetLocalConnection("/TestHostRegistry_Spawn")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)

	// Add a host
	addHost := func(hostID string) *host.Host {
		host := host.Host{ID: hostID}
		err := AddHost(conn, &host)
		c.Assert(err, IsNil)
		return &host
	}

	// Register a host
	regHost := func(host *host.Host) string {
		hpath, err := registerHost(conn, host)
		c.Assert(err, IsNil)
		return path.Base(hpath)
	}

	// Add a service
	addService := func(serviceID string) *service.Service {
		svc := service.Service{ID: "test-service-1"}
		err = UpdateService(conn, svc, false, false)
		c.Assert(err, IsNil)
		return &svc
	}

	// Add service states
	addServiceStates := func(svc *service.Service, host *host.Host, count int) []servicestate.ServiceState {
		states := make([]servicestate.ServiceState, count)
		for i := range states {
			state, err := servicestate.BuildFromService(svc, host.ID)
			c.Assert(err, IsNil)
			err = addInstance(conn, *state)
			c.Assert(err, IsNil)
			states[i] = *state
		}
		return states
	}

	listener := NewHostRegistryListener()
	listener.SetConnection(conn)

	svc1 := addService("test-service-1")
	svc2 := addService("test-service-2")

	host1 := addHost("test-host-1")
	eHostID1 := regHost(host1)
	host1states := make(map[string]servicestate.ServiceState)
	for _, state := range addServiceStates(svc1, host1, 2) {
		host1states[state.ID] = state
	}
	for _, state := range addServiceStates(svc2, host1, 1) {
		host1states[state.ID] = state
	}

	host2 := addHost("test-host-2")
	eHostID2 := regHost(host2)
	host2states := make(map[string]servicestate.ServiceState)
	for _, state := range addServiceStates(svc1, host2, 1) {
		host2states[state.ID] = state
	}
	for _, state := range addServiceStates(svc2, host2, 2) {
		host2states[state.ID] = state
	}

	// test shutdown
	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		listener.Spawn(shutdown, eHostID1)
	}()
	close(shutdown)
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		c.Fatalf("timeout waiting for shutdown")
	}
	stateIDs, err := conn.Children(hostpath(host1.ID))
	c.Assert(err, IsNil)
	c.Assert(stateIDs, HasLen, len(host1states))
	for _, stateID := range stateIDs {
		state, ok := host1states[stateID]
		c.Assert(ok, Equals, true)
		exists, err := conn.Exists(servicepath(state.ServiceID, state.ID))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, true)
	}
	stateIDs, err = conn.Children(hostpath(host2.ID))
	c.Assert(err, IsNil)
	c.Assert(stateIDs, HasLen, len(host2states))
	for _, stateID := range stateIDs {
		state, ok := host2states[stateID]
		c.Assert(ok, Equals, true)
		exists, err := conn.Exists(servicepath(state.ServiceID, state.ID))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, true)
	}

	// test delete
	shutdown = make(chan interface{})
	defer close(shutdown)
	done = make(chan struct{})
	go func() {
		defer close(done)
		listener.Spawn(shutdown, eHostID2)
	}()
	err = conn.Delete(hostregpath(eHostID2))
	c.Assert(err, IsNil)
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		c.Fatalf("timeout waiting for shutdown")
	}
	// make sure host1 states are still there
	stateIDs, err = conn.Children(hostpath(host1.ID))
	c.Assert(err, IsNil)
	c.Assert(stateIDs, HasLen, len(host1states))
	for _, stateID := range stateIDs {
		state, ok := host1states[stateID]
		c.Assert(ok, Equals, true)
		exists, err := conn.Exists(servicepath(state.ServiceID, state.ID))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, true)
	}
	// host2 states should be deleted
	stateIDs, err = conn.Children(hostpath(host2.ID))
	c.Assert(err, IsNil)
	c.Assert(stateIDs, HasLen, 0)
	for _, state := range host2states {
		exists, err := conn.Exists(servicepath(state.ServiceID, state.ID))
		if err != nil {
			c.Assert(err, Equals, client.ErrNoNode)
		}
		c.Assert(exists, Equals, false)
	}
}
