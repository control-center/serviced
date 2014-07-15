// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package service

import (
	"path"
	"sync"
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/zzk"
)

func TestHostRegistryListener_Spawn(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	listener := NewHostRegistryListener(conn)

	// Register the host
	host := &host.Host{ID: "test-host-1"}
	if err := RegisterHost(conn, host.ID); err != nil {
		t.Fatalf("Could not register host %s: %s", host.ID, err)
	}

	// Create the ephemeral host
	ehostpath, err := conn.CreateEphemeral(hostregpath(host.ID), &HostNode{Host: host})
	if err != nil {
		t.Fatalf("Could not create ephemeral host")
	}
	ehostID := path.Base(ehostpath)

	var (
		wg       sync.WaitGroup
		shutdown = make(chan interface{})
	)
	listener.shutdown = shutdown
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(shutdown, ehostID)
	}()

	// add some service instances
	t.Log("Creating some service instances")
	svc := &service.Service{ID: "test-service-1"}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.ID, err)
	}
	var states []*servicestate.ServiceState
	for i := 0; i < 3; i++ {
		state, err := servicestate.BuildFromService(svc, host.ID)
		if err != nil {
			t.Fatalf("Could not create service state from service %s: %s", svc.ID, err)
		}
		if err := addInstance(conn, state); err != nil {
			t.Fatalf("Could not add service state %s: %s", state.ID, err)
		}
		states = append(states, state)
	}

	// shutdown listener
	<-time.After(time.Second)
	t.Log("Shutting down listener")
	close(shutdown)
	wg.Wait()

	// verify none of the service states were removed
	for _, state := range states {
		if exists, err := zzk.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
			t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
		} else if !exists {
			t.Errorf("Deleted host node %s", state.ID)
		} else if exists, err := zzk.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
			t.Fatalf("Error checking path %s: %s", servicepath(state.ServiceID, state.ID), err)
		} else if !exists {
			t.Errorf("Deleted service node %s", state.ID)
		}
	}

	shutdown = make(chan interface{})
	listener.shutdown = shutdown
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(shutdown, ehostID)
	}()

	// remove the ephemeral node
	<-time.After(time.Second)
	t.Log("Removing the ephemeral node: ", ehostpath)
	if err := conn.Delete(ehostpath); err != nil {
		t.Fatalf("Error trying to remove node %s: %s", ehostpath, err)
	}
	wg.Wait()

	// verify that all of the service states were removed
	for _, state := range states {
		// verify the state has been removed
		if exists, err := zzk.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
			t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
		} else if exists {
			t.Errorf("Failed to delete host node %s", state.ID)
		} else if exists, err := zzk.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
			t.Fatalf("Error checking path %s: %s", servicepath(state.ServiceID, state.ID), err)
		} else if exists {
			t.Errorf("Failed to delete service node %s", state.ID)
		}
	}
}

func TestHostRegistryListener_unregister(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	listener := NewHostRegistryListener(conn)

	host1 := &host.Host{ID: "test-host-1"}
	host2 := &host.Host{ID: "test-host-2"}

	// add some service instances
	t.Log("Creating some service instances")
	svc := &service.Service{ID: "test-service-1"}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.ID, err)
	}
	var states []*servicestate.ServiceState
	for i := 0; i < 3; i++ {
		state, err := servicestate.BuildFromService(svc, host1.ID)
		if err != nil {
			t.Fatalf("Could not create service state from service %s: %s", svc.ID, err)
		}
		if err := addInstance(conn, state); err != nil {
			t.Fatalf("Could not add service instance %s: %s", state.ID, err)
		}
		states = append(states, state)
	}
	for i := 0; i < 2; i++ {
		state, err := servicestate.BuildFromService(svc, host2.ID)
		if err != nil {
			t.Fatalf("Could not create service state from service %s: %s", svc.ID, err)
		}
		if err := addInstance(conn, state); err != nil {
			t.Fatalf("Could not add service instance %s: %s", state.ID, err)
		}
		states = append(states, state)
	}

	// unregister the host instances
	t.Log("Unregistering service instances for ", host1.ID)
	listener.unregister(host1.ID)
	var saved, removed int
	for _, state := range states {
		if state.HostID == host1.ID {
			// verify the state has been removed
			if exists, err := zzk.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
			} else if exists {
				t.Errorf("Failed to delete host node %s", state.ID)
			} else if exists, err := zzk.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", servicepath(state.ServiceID, state.ID), err)
			} else if exists {
				t.Errorf("Failed to delete service node %s", state.ID)
			} else {
				removed++
			}
		} else {
			// verify the state has been preserved
			if exists, err := zzk.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
			} else if !exists {
				t.Errorf("Deleted host node %s", state.ID)
			} else if exists, err := zzk.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", servicepath(state.ServiceID, state.ID), err)
			} else if !exists {
				t.Errorf("Deleted service node %s", state.ID)
			} else {
				saved++
			}
		}
	}

	if saved != 2 {
		t.Errorf("Some service states were not saved")
	}
	if removed != 3 {
		t.Errorf("Some service states were not removed")
	}
}
