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
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/service/mocks"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestHostRegistryListener_Spawn(c *C) {
	conn, err := zzk.GetLocalConnection("/TestHostRegistry_Spawn")
	c.Assert(err, IsNil)

	// set up the resource pool
	p := &pool.ResourcePool{
		ID:                "testpool",
		ConnectionTimeout: 2000,
	}
	ppth := path.Join("/pools", p.ID)
	err = conn.Create(ppth, &PoolNode{ResourcePool: p})
	c.Assert(err, IsNil)

	// set up a service
	s := &ServiceNode{
		ID: "testservice",
	}
	spth := path.Join(ppth, "/services", s.ID)

	err = conn.Create(spth, s)
	c.Assert(err, IsNil)

	handler := &mocks.VirtualIPUnassignmentHandler{}
	handler.On("UnassignAll", mock.Anything, mock.Anything).Return(nil)

	// initialize the listener
	listener := NewHostRegistryListener(p.ID, handler)
	listener.SetConnection(conn)

	// case 1: try to listen on a node that doesn't exist
	stop := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		listener.Spawn(stop, "h0")
	}()
	select {
	case <-done:
	case <-time.After(p.GetConnectionTimeout()):
		close(stop)
		c.Fatalf("Timed out waiting for listener to exit")
	}

	// case 2: remove a host, offline, no instances
	h1 := &host.Host{
		ID:        "h1",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	h1pth := path.Join(ppth, "/hosts", h1.ID)
	err = conn.Create(h1pth, &HostNode{Host: h1})
	c.Assert(err, IsNil)
	stop = make(chan interface{})
	done = make(chan struct{})
	go func() {
		defer close(done)
		listener.Spawn(stop, h1.ID)
	}()
	time.Sleep(p.GetConnectionTimeout())
	err = conn.Delete(h1pth)
	c.Assert(err, IsNil)
	select {
	case <-done:
	case <-time.After(p.GetConnectionTimeout()):
		close(stop)
		c.Fatalf("Timed out waiting for listener to exit")
	}

	// case 3: offline, instance
	err = conn.Create(h1pth, &HostNode{Host: h1})
	c.Assert(err, IsNil)
	c.Logf("Created node at path %s", h1pth)

	req := StateRequest{
		PoolID:     p.ID,
		HostID:     h1.ID,
		ServiceID:  s.ID,
		InstanceID: 0,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	stop = make(chan interface{})
	done = make(chan struct{})
	ok, st8ev, err := conn.ExistsW(path.Join(spth, req.StateID()), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	ok, hst8ev, err := conn.ExistsW(path.Join(h1pth, "instances", req.StateID()), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	go func() {
		defer close(done)
		listener.Spawn(stop, "h1")
	}()

	select {
	case <-st8ev:
		ok, err := conn.Exists(path.Join(spth, req.StateID()))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
		ok, err = conn.Exists(path.Join(h1pth, "instances", req.StateID()))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-hst8ev:
		ok, err := conn.Exists(path.Join(spth, req.StateID()))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
		ok, err = conn.Exists(path.Join(h1pth, "instances", req.StateID()))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-done:
		c.Fatalf("Unexpected shutdown of listener")
	case <-time.After(p.GetConnectionTimeout()):
		close(stop)
		c.Fatalf("Timed out waiting for instance cleanup")
	}

	// case 4: online
	hostsCh := make(chan []host.Host)
	go func() {
		hosts, err := listener.GetRegisteredHosts(stop)
		c.Assert(err, IsNil)
		hostsCh <- hosts
	}()
	select {
	case <-hostsCh:
		close(stop)
		c.Fatalf("Received unexpected response of registered hosts")
	case <-time.After(p.GetConnectionTimeout()):
	}
	// Need a path to the node, even though the node name does not correlate
	// when creating ephemerals
	eh1pth, err := conn.CreateEphemeral(path.Join(h1pth, "online", h1.ID), &client.Dir{})
	c.Assert(err, IsNil)
	select {
	case hosts := <-hostsCh:
		c.Assert(hosts, DeepEquals, []host.Host{*h1})
	case <-time.After(p.GetConnectionTimeout()):
		c.Fatalf("Timed out waiting for host availability")
	}

	// case 5: offline, instance
	req = StateRequest{
		PoolID:     p.ID,
		HostID:     h1.ID,
		ServiceID:  s.ID,
		InstanceID: 1,
	}
	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	ok, st8ev, err = conn.ExistsW(path.Join(spth, req.StateID()), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	ok, hst8ev, err = conn.ExistsW(path.Join(h1pth, "instances", req.StateID()), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	err = conn.Delete(path.Join(h1pth, "online", path.Base(eh1pth)))
	c.Assert(err, IsNil)
	select {
	case <-st8ev:
		c.Fatalf("Unexpected removal of service instance")
	case <-hst8ev:
		c.Fatalf("Unexpected removal of state instance")
	case <-time.After(p.GetConnectionTimeout() + time.Second):
	}

	// case 6: another host goes online
	h2 := &host.Host{
		ID:        "h2",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	h2pth := path.Join(ppth, "/hosts", h2.ID)
	err = conn.Create(h2pth, &HostNode{Host: h2})
	c.Assert(err, IsNil)
	// Need a path to the node, even though the node name does not correlate
	// when creating ephemerals
	_, err = conn.CreateEphemeral(path.Join(h2pth, "online", h2.ID), &client.Dir{})
	c.Assert(err, IsNil)
	done2 := make(chan struct{})
	go func() {
		defer close(done2)
		listener.Spawn(stop, h2.ID)
	}()
	time.Sleep(p.GetConnectionTimeout() - time.Second)
	// Need a path to the node, even though the node name does not correlate
	// when creating ephemerals
	eh1pth, err = conn.CreateEphemeral(path.Join(h1pth, "online", h1.ID), &client.Dir{})
	c.Assert(err, IsNil)
	select {
	case <-st8ev:
		c.Fatalf("Unexpected removal of service instance")
	case <-hst8ev:
		c.Fatalf("Unexpected removal of state instance")
	case <-time.After(p.GetConnectionTimeout() + time.Second):
	}

	// case 7: reschedule
	err = conn.Delete(path.Join(h1pth, "online", path.Base(eh1pth)))
	c.Assert(err, IsNil)
	select {
	case <-st8ev:
		ok, err := conn.Exists(path.Join(spth, req.StateID()))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
		ok, err = conn.Exists(path.Join(h1pth, "instances", req.StateID()))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-hst8ev:
		ok, err := conn.Exists(path.Join(spth, req.StateID()))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
		ok, err = conn.Exists(path.Join(h1pth, "instances", req.StateID()))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-time.After(2*p.GetConnectionTimeout() + time.Second):
		c.Fatalf("Timed out waiting waiting for instance delete")
	}

	// case 8: turn off listeners
	close(stop)
	timer := time.After(p.GetConnectionTimeout())
	select {
	case <-done:
	case <-timer:
		c.Fatalf("Timed out waiting for listener to shutdown")
	}

	select {
	case <-done2:
	case <-timer:
		c.Fatalf("Timed out waiting for listener to shutdown")
	}
}
