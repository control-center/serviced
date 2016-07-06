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
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"

	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestHostRegistryListener_Spawn(c *C) {
	conn, err := zzk.GetLocalConnection("/TestHostRegistry_Spawn")
	c.Assert(err, IsNil)

	// set up the resource pool
	p := &pool.ResourcePool{
		ID:                "test-pool",
		ConnectionTimeout: 2 * time.Second,
	}
	ppth := path.Join("/pools", p.ID)
	err = conn.Create(ppth, &PoolNode{ResourcePool: p})
	c.Assert(err, IsNil)

	// set up a service
	s := &service.Service{
		ID: "test-service",
	}
	spth := path.Join(ppth, "/services", s.ID)
	err = conn.Create(spth, &ServiceNode{Service: s})
	c.Assert(err, IsNil)

	// initialize the listener
	listener := NewHostRegistryListener(p.ID)
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
	case <-time.After(p.ConnectionTimeout):
		close(stop)
		c.Fatalf("Timed out waiting for listener to exit")
	}

	// case 2: remove a host, offline, no instances
	h1 := &host.Host{
		ID:        "h1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
	time.Sleep(p.ConnectionTimeout)
	err = conn.Delete(h1pth)
	c.Assert(err, IsNil)
	select {
	case <-done:
	case <-time.After(p.ConnectionTimeout):
		close(stop)
		c.Fatalf("Timed out waiting for listener to exit")
	}

	// case 3: offline, instance
	err = conn.Create(h1pth, &HostNode{Host: h1})
	c.Assert(err, IsNil)
	c.Logf("Created node at path %s", h1pth)

	st8 := &servicestate.ServiceState{
		ID:        "test-state",
		HostID:    h1.ID,
		ServiceID: s.ID,
	}
	err = addInstance(conn, p.ID, *st8)
	c.Assert(err, IsNil)

	stop = make(chan interface{})
	done = make(chan struct{})
	ok, st8ev, err := conn.ExistsW(path.Join(spth, st8.ID), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	ok, hst8ev, err := conn.ExistsW(path.Join(h1pth, "instances", st8.ID), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	go func() {
		defer close(done)
		listener.Spawn(stop, "h1")
	}()

	select {
	case <-st8ev:
		ok, err := conn.Exists(path.Join(spth, st8.ID))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
		ok, err = conn.Exists(path.Join(h1pth, "instances", st8.ID))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-hst8ev:
		ok, err := conn.Exists(path.Join(spth, st8.ID))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
		ok, err = conn.Exists(path.Join(h1pth, "instances", st8.ID))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-done:
		c.Fatalf("Unexpected shutdown of listener")
	case <-time.After(p.ConnectionTimeout):
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
	case <-time.After(p.ConnectionTimeout):
	}
	eh1pth, err := conn.CreateEphemeral(path.Join(h1pth, "online", h1.ID), &client.Dir{})
	c.Assert(err, IsNil)
	select {
	case hosts := <-hostsCh:
		c.Assert(hosts, DeepEquals, []host.Host{*h1})
	case <-time.After(p.ConnectionTimeout):
		c.Fatalf("Timed out waiting for host availability")
	}

	// case 5: offline, instance
	err = addInstance(conn, p.ID, *st8)
	c.Assert(err, IsNil)
	ok, st8ev, err = conn.ExistsW(path.Join(spth, st8.ID), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	ok, hst8ev, err = conn.ExistsW(path.Join(h1pth, "instances", st8.ID), done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	err = conn.Delete(path.Join(h1pth, "online", path.Base(eh1pth)))
	c.Assert(err, IsNil)
	select {
	case <-st8ev:
		c.Fatalf("Unexpected removal of service instance")
	case <-hst8ev:
		c.Fatalf("Unexpected removal of state instance")
	case <-time.After(p.ConnectionTimeout + time.Second):
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
	_, err = conn.CreateEphemeral(path.Join(h2pth, "online", h2.ID), &client.Dir{})
	c.Assert(err, IsNil)
	done2 := make(chan struct{})
	go func() {
		defer close(done2)
		listener.Spawn(stop, h2.ID)
	}()
	time.Sleep(p.ConnectionTimeout - time.Second)
	eh1pth, err = conn.CreateEphemeral(path.Join(h1pth, "online", h1.ID), &client.Dir{})
	c.Assert(err, IsNil)
	select {
	case <-st8ev:
		c.Fatalf("Unexpected removal of service instance")
	case <-hst8ev:
		c.Fatalf("Unexpected removal of state instance")
	case <-time.After(p.ConnectionTimeout + time.Second):
	}

	// case 7: reschedule
	err = conn.Delete(path.Join(h1pth, "online", path.Base(eh1pth)))
	c.Assert(err, IsNil)
	select {
	case <-st8ev:
		ok, err := conn.Exists(path.Join(spth, st8.ID))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
		ok, err = conn.Exists(path.Join(h1pth, "instances", st8.ID))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-hst8ev:
		ok, err := conn.Exists(path.Join(spth, st8.ID))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
		ok, err = conn.Exists(path.Join(h1pth, "instances", st8.ID))
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, false)
	case <-time.After(2*p.ConnectionTimeout + time.Second):
		c.Fatalf("Timed out waiting waiting for instance delete")
	}

	// case 8: turn off listeners
	close(stop)
	timer := time.After(p.ConnectionTimeout)
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
