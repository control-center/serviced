package service

import (
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
)

func TestHostRegistryListener_Listen(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()

	listener := NewHostRegistryListener(conn)
	alert := make(chan bool)
	shutdown := make(chan interface{})
	listener.alertC = alert
	go listener.Listen(shutdown)

	// Create the service
	svc := &service.Service{
		Id:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.Id, err)
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
			t.Fatalf("Could not generate instance from service %s", svc.Id)
		} else if err := addInstance(conn, state); err != nil {
			t.Fatalf("Could not add instance %s from service %s", state.Id, state.ServiceID)
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
	if hosts := listener.GetHosts(); len(hosts) != 1 {
		t.Errorf("Found %d hosts; expected 1 host", len(hosts))
	} else if hosts[0].ID != host.ID {
		t.Errorf("MISMATCH: expected %s host id; actual", host.ID, hosts[0].ID)
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
	if hosts := listener.GetHosts(); len(hosts) != 0 {
		t.Errorf("Hosts were not removed: %v", hosts)
	}

	for _, state := range states {
		if exists, err := conn.Exists(hostpath(state.HostID, state.Id)); err != nil {
			t.Fatalf("Could not check existance of host state %s: %s", state.Id, err)
		} else if exists {
			t.Fatal("State still exists for host state ", state.Id)
		} else if exists, err := conn.Exists(servicepath(state.ServiceID, state.Id)); err != nil {
			t.Fatalf("Could not check existance of service state %s: %s", state.Id, err)
		} else if exists {
			t.Fatal("State still exists for service state ", state.Id)
		}
	}

	// Shutdown!
	close(shutdown)
}

func TestHostRegistryListener_sync(t *testing.T) {
}

func TestHostRegistryListener_register(t *testing.T) {
}

func TestHostRegistryListener_unregister(t *testing.T) {
}