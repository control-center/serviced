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

package service

import (
	"path"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"

	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestHostRegistryListener_Listen(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)

	// Initialize the host registry
	err = InitHostRegistry(conn)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	defer close(shutdown)

	listener := NewHostRegistryListener()
	go zzk.Listen(shutdown, make(chan error, 1), conn, listener)

	// Add a service
	svc := service.Service{ID: "test-service-1"}
	err = UpdateService(conn, &svc)
	c.Assert(err, IsNil)

	// Add hosts
	hosts := make(map[string]string)
	register := func(hostID string) string {
		c.Logf("Registering host %s", hostID)
		host := host.Host{ID: hostID}

		err := AddHost(conn, &host)
		c.Assert(err, IsNil)

		p, err := conn.CreateEphemeral(hostregpath(hostID), &HostNode{Host: &host})
		c.Assert(err, IsNil)

		return path.Base(p)
	}
	hosts["test-host-1"] = register("test-host-1")
	hosts["test-host-2"] = register("test-host-2")

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
	addstates("test-host-1", &svc, 2)
	addstates("test-host-2", &svc, 2)

	// unregister a host and verify the states have been removed
	unregister := func(hostID, ehostID string) {
		var wg sync.WaitGroup

		hsids, err := conn.Children(hostpath(hostID))
		c.Assert(err, IsNil)

		// Monitor the states per service
		for _, hsid := range hsids {
			var hs HostState
			err = conn.Get(hostpath(hostID, hsid), &hs)
			c.Assert(err, IsNil)

			wg.Add(1)
			go func(hsid, serviceID string) {
				defer wg.Done()
				for {
					event, err := conn.GetW(servicepath(serviceID, hsid), &HostNode{})
					c.Assert(err, IsNil)
					if e := <-event; e.Type == client.EventNodeDeleted {
						return
					}
				}
			}(hs.ServiceStateID, hs.ServiceID)
		}

		// Monitor the host state
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				hsids, event, err := conn.ChildrenW(hostpath(hostID))
				c.Assert(err, IsNil)
				if len(hsids) == 0 {
					return
				}
				<-event
			}
		}()

		c.Logf("Unregistering host %s", hostID)
		err = conn.Delete(hostregpath(ehostID))
		c.Assert(err, IsNil)
		wg.Wait()
	}

	done := make(chan struct{})
	go func() {
		done <- struct{}{}
		unregister("test-host-1", hosts["test-host-1"])
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Minute):
		c.Errorf("timeout")
	}
}

func (t *ZZKTest) TestHostRegistryListener_Spawn(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)

	// Initialize the host registry
	err = InitHostRegistry(conn)
	c.Assert(err, IsNil)

	listener := NewHostRegistryListener()
	listener.SetConnection(conn)

	// Add a service
	svc := service.Service{ID: "test-service-1"}
	err = UpdateService(conn, &svc)
	c.Assert(err, IsNil)

	// Add hosts
	register := func(hostID string) string {
		c.Logf("Registering host %s", hostID)
		host := host.Host{ID: hostID}

		err := AddHost(conn, &host)
		c.Assert(err, IsNil)

		p, err := conn.CreateEphemeral(hostregpath(hostID), &HostNode{Host: &host})
		c.Assert(err, IsNil)

		return path.Base(p)
	}
	ehostID := register("test-host-1")

	var wg sync.WaitGroup
	wg.Add(1)
	shutdown := make(chan interface{})
	go func() {
		defer wg.Done()
		listener.Spawn(shutdown, ehostID)
	}()

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

	// shutdown listener
	c.Logf("Testing shutdown")
	time.Sleep(time.Second)
	close(shutdown)
	wg.Wait()

	// verify none of the states were removed
	for _, stateID := range stateIDs {
		exists, err := conn.Exists(hostpath("test-host-1", stateID))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, true)

		exists, err = conn.Exists(servicepath(svc.ID, stateID))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, true)
	}

	// start the listener again
	shutdown = make(chan interface{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(shutdown, ehostID)
	}()

	// unregister a host and verify the states have been removed
	unregister := func(hostID, ehostID string) {
		var wg sync.WaitGroup

		hsids, err := conn.Children(hostpath(hostID))
		c.Assert(err, IsNil)

		// Monitor the states per service
		for _, hsid := range hsids {
			var hs HostState
			err = conn.Get(hostpath(hostID, hsid), &hs)
			c.Assert(err, IsNil)

			wg.Add(1)
			go func(hsid, serviceID string) {
				defer wg.Done()
				for {
					event, err := conn.GetW(servicepath(serviceID, hsid), &HostNode{})
					c.Assert(err, IsNil)
					if e := <-event; e.Type == client.EventNodeDeleted {
						return
					}
				}
			}(hs.ServiceStateID, hs.ServiceID)
		}

		// Monitor the host state
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				hsids, event, err := conn.ChildrenW(hostpath(hostID))
				c.Assert(err, IsNil)
				if len(hsids) == 0 {
					return
				}
				<-event
			}
		}()

		c.Logf("Unregistering host %s", hostID)
		err = conn.Delete(hostregpath(ehostID))
		c.Assert(err, IsNil)
		wg.Wait()
	}

	defer close(shutdown)
	c.Logf("Removing ephemeral node %s", ehostID)
	time.Sleep(time.Second)

	done := make(chan struct{})
	go func() {
		done <- struct{}{}
		unregister("test-host-1", ehostID)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Minute):
		c.Errorf("timeout")
	}
}

func (t *ZZKTest) TestHostRegistryListener_unregister(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)

	// Initialize the host registry
	err = InitHostRegistry(conn)
	c.Assert(err, IsNil)

	listener := NewHostRegistryListener()
	listener.SetConnection(conn)

	// Add a service
	svc := service.Service{ID: "test-service-1"}
	err = UpdateService(conn, &svc)
	c.Assert(err, IsNil)

	// Add hosts
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
	register("test-host-2")

	// Add states
	states := make(map[string][]string)
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
	states["test-host-1"] = addstates("test-host-1", &svc, 2)
	states["test-host-2"] = addstates("test-host-2", &svc, 2)

	// unregister the host instances
	c.Logf("Unregistering service instances for 'test-host-1'")
	listener.unregister("test-host-1")

	// verify states removed
	for _, state := range states["test-host-1"] {
		exists, err := conn.Exists(hostpath("test-host-1", state))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, false)
		exists, err = conn.Exists(servicepath(svc.ID, state))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, false)
	}

	// verify states preserved
	for _, state := range states["test-host-2"] {
		exists, err := conn.Exists(hostpath("test-host-2", state))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, true)
		exists, err = conn.Exists(servicepath(svc.ID, state))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, true)
	}
}
