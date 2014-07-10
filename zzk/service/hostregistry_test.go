package service

import (
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	zkutils "github.com/zenoss/serviced/zzk/utils"
)

func TestHostRegistryListener_Listen(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()

	listener, err := NewHostRegistryListener(conn)
	if err != nil {
		t.Fatal("Could not create listener: %s", err)
	}
	alert := make(chan bool)
	shutdown := make(chan interface{})
	listener.alertC = alert
	go listener.Listen(shutdown)

	// Create the service
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.ID, err)
	}

	// Create a host
	host := &host.Host{ID: "test-host-1"}

	// Add some instances
	t.Log("Setting up data")
	var states []*servicestate.ServiceState
	for i := 0; i < 3; i++ {
		// Create a service instance
		state, err := servicestate.BuildFromService(svc, host.ID)
		if err != nil {
			t.Fatalf("Could not generate instance from service %s", svc.ID)
		} else if err := addInstance(conn, state); err != nil {
			t.Fatalf("Could not add instance %s from service %s", state.ID, state.ServiceID)
		}
		states = append(states, state)
	}

	// Create the host "ephemeral" node (for the sake of unit tests, it doesn't really have to be ephemeral)
	<-time.After(1 * time.Second)
	t.Log("Adding ephemeral host")
	if err := conn.Create(hostregpath(host.ID), &HostNode{Host: host}); err != nil {
		t.Fatalf("Could not add host %s: %s", host.ID, err)
	}

	// Verify that the host exists
	t.Log("Verifying host was added")
	<-alert
	if count := len(listener.hostmap); count != 1 {
		t.Errorf("Found %d hosts; expected 1 host", count)
	} else {
		for _, hostID := range listener.hostmap {
			if hostID != host.ID {
				t.Errorf("MISMATCH: expected %s host id; actual", host.ID, hostID)
			}
		}
	}

	// Remove the host "ephemeral" node (host network goes down :( )
	<-time.After(1 * time.Second)
	t.Log("Removing ephemeral host")
	if err := conn.Delete(hostregpath(host.ID)); err != nil {
		t.Fatalf("Could not remove host %s: %s", host.ID, err)
	}

	// Verify the service states were removed
	t.Log("Verifying host removed")
	<-alert
	if count := len(listener.hostmap); count != 0 {
		t.Errorf("Hosts were not removed: %v", listener.hostmap)
	}

	for _, state := range states {
		if exists, err := zkutils.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
			t.Fatalf("Could not check existance of host state %s: %s", state.ID, err)
		} else if exists {
			t.Fatal("State still exists for host state ", state.ID)
		}
	}

	// Shutdown!
	close(shutdown)
}

func TestHostRegistryListener_sync(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	listener, err := NewHostRegistryListener(conn)
	if err != nil {
		t.Fatal("Could not create listener: %s", err)
	}

	// Add some hosts
	hosts := map[string]*host.Host{
		"ehost-1": &host.Host{ID: "test-host-1"},
		"ehost-2": &host.Host{ID: "test-host-2"},
		"ehost-3": &host.Host{ID: "test-host-3"},
		"ehost-4": &host.Host{ID: "test-host-4"},
	}

	for ehost, host := range hosts {
		if err := conn.CreateDir(hostpath(host.ID)); err != nil {
			t.Fatalf("Could not create host node %s: %s", host.ID, err)
		}
		if err := conn.Create(hostregpath(ehost), &HostNode{Host: host}); err != nil {
			t.Fatalf("Could not add host %s to registry %s: %s", host.ID, ehost, err)
		}
	}

	nodes := [][]string{
		{"ehost-1", "ehost-2", "ehost-3"},
		{"ehost-1", "ehost-3"},
		{"ehost-3", "ehost-4"},
	}

	for _, sync := range nodes {
		listener.sync(sync)
		if len(sync) != len(listener.hostmap) {
			t.Errorf("MISMATCH: Expected %d mapped nodes; Actual: %d", len(nodes), len(listener.hostmap))
		}
		for _, n := range sync {
			if hostID, ok := listener.hostmap[n]; !ok {
				t.Errorf("HOST %s (%v) not found", n, hosts[n])
			} else if hostID != hosts[n].ID {
				t.Errorf("MISMATCH: Expected host %s from %s; Actual: %s", hosts[n].ID, n, hostID)
			}
		}
	}
}

func TestHostRegistryListener_register(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	listener, err := NewHostRegistryListener(conn)
	if err != nil {
		t.Fatal("Could not create listener: %s", err)
	}
	host := &host.Host{ID: "test-host-1"}

	// no running listener
	if err := listener.register("test-ehost-1", host.ID); err != ErrHostNotInitialized {
		t.Errorf("Expected error: '%s'; Actual error: '%s'", ErrHostNotInitialized, err)
	}

	// success
	if err := conn.CreateDir(hostpath(host.ID)); err != nil {
		t.Fatalf("Could not create host node %s: %s", host.ID, err)
	}
	if err := listener.register("test-ehost-1", host.ID); err != nil {
		t.Errorf("Could not register host node %s: %s", "test-ehost-1", err)
	}
	if hostID, ok := listener.hostmap["test-ehost-1"]; !ok || hostID == "" {
		t.Errorf("Host %s not found", "test-ehost-1")
	} else if hostID != host.ID {
		t.Errorf("MISMATCH: expected %s; actual %s", host.ID, hostID)
	}
}

func TestHostRegistryListener_unregister(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	listener, err := NewHostRegistryListener(conn)
	if err != nil {
		t.Fatal("Could not create listener: %s", err)
	}

	// Create the host
	host := &host.Host{ID: "test-host-1"}

	// Create the service
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.ID, err)
	}

	// Add some instances
	var states []*servicestate.ServiceState
	for i := 0; i < 3; i++ {
		// Create a service instance
		state, err := servicestate.BuildFromService(svc, host.ID)
		if err != nil {
			t.Fatalf("Could not generate instance from service %s: %s", svc.ID, err)
		} else if err := addInstance(conn, state); err != nil {
			t.Fatalf("Could not add instance %s from service %s: %s", state.ID, state.ServiceID, err)
		}
		states = append(states, state)
	}

	// register the ephemeral node
	if err := listener.register("test-ehost-1", host.ID); err != nil {
		t.Fatalf("Could not register node: %s", err)
	}

	// unregister the ephemeral node
	if err := listener.unregister("test-ehost-1"); err != nil {
		t.Errorf("Could not unregister node: %s", err)
	}

	if _, ok := listener.hostmap["test-ehost-1"]; ok {
		t.Errorf("Did not remove node from host map")
	}

	// verify the children were removed
	if ssids, err := conn.Children(hostpath(host.ID)); err != nil {
		t.Fatalf("Errror looking up children on host path: %s", err)
	} else if count := len(ssids); count > 0 {
		t.Errorf("Some instances still left behind: %d", count)
	}
}