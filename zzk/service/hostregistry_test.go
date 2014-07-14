package service

import (
	"fmt"
	"path"
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
		t.Fatalf("Could not initialize host registry listener: %s", err)
	}

	var (
		shutdown = make(chan interface{})
		wait     = make(chan interface{})
	)
	go func() {
		listener.Listen(shutdown)
		close(wait)
	}()

	// Create services
	numServices := 5
	var svcs []*service.Service
	for i := 0; i < numServices; i++ {
		svc := &service.Service{ID: fmt.Sprintf("test-service-%d", i)}
		if err := UpdateService(conn, svc); err != nil {
			t.Fatalf("Could not add service %s: %s", svc.ID, err)
		}
		svcs = append(svcs, svc)
	}

	// Register hosts
	t.Log("Registering hosts")
	numHosts := 5
	hosts := make(map[string]*host.Host)
	for i := 0; i < numHosts; i++ {
		host := &host.Host{ID: fmt.Sprintf("test-host-%d", i)}
		if err := RegisterHost(conn, host.ID); err != nil {
			t.Fatalf("Could not register host %s: %s", host.ID, err)
		}
		ehostpath, err := conn.CreateEphemeral(hostregpath(host.ID), &HostNode{Host: host})
		t.Log("Ephemeral node: ", ehostpath)
		if err != nil {
			t.Fatalf("Could not register host %s: %s", host.ID, err)
		}
		hosts[ehostpath] = host
	}

	// Add service states
	t.Log("Adding service states")
	var states []*servicestate.ServiceState
	for _, host := range hosts {
		for _, svc := range svcs {
			state, err := servicestate.BuildFromService(svc, host.ID)
			if err != nil {
				t.Fatalf("Could not create service state: %s", err)
			}
			if err := addInstance(conn, state); err != nil {
				t.Fatalf("Could not add service state %s: %s", state.ID, err)
			}
			if _, err := LoadRunningService(conn, state.ServiceID, state.ID); err != nil {
				t.Fatalf("Could not get running service: %s", state.ID)
			}
			states = append(states, state)
		}
	}

	// Delete hosts
	deleteHosts := 2
	deletedHosts := make(map[string]interface{})
	for ehostpath, host := range hosts {
		<-time.After(time.Second)
		t.Log("Removing host: ", host.ID)
		if err := conn.Delete(ehostpath); err != nil {
			t.Fatalf("Could not delete ephemeral node %s: %s", ehostpath, err)
		}
		deletedHosts[host.ID] = nil
		deleteHosts--
		if deleteHosts == 0 {
			break
		}
	}

	// Shutdown
	<-time.After(time.Second)
	close(shutdown)
	<-wait
	for _, state := range states {
		if _, ok := deletedHosts[state.HostID]; ok {
			// verify the state has been removed
			if exists, err := zkutils.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
			} else if exists {
				t.Errorf("Failed to delete host node %s", state.ID)
			} else if exists, err := zkutils.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", servicepath(state.ServiceID, state.ID), err)
			} else if exists {
				t.Errorf("Failed to delete service node %s", state.ID)
			}
		} else {
			// verify the state has been preserved
			if exists, err := zkutils.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
			} else if !exists {
				t.Errorf("Deleted host node %s", state.ID)
			} else if exists, err := zkutils.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", servicepath(state.ServiceID, state.ID), err)
			} else if !exists {
				t.Errorf("Deleted service node %s", state.ID)
			}
		}
	}
}

func TestHostRegistryListener_listenHost(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	listener, err := NewHostRegistryListener(conn)
	if err != nil {
		t.Fatalf("Could not initialize host registry listener: %s", err)
	}

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
		shutdown = make(chan interface{})
		done     = make(chan string)
	)
	listener.shutdown = shutdown
	go listener.listenHost(done, ehostID)

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
	if id := <-done; id != ehostID {
		t.Errorf("Expected eHost %s; Actual %s", ehostID, id)
	}

	// verify none of the service states were removed
	for _, state := range states {
		if exists, err := zkutils.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
			t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
		} else if !exists {
			t.Errorf("Deleted host node %s", state.ID)
		} else if exists, err := zkutils.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
			t.Fatalf("Error checking path %s: %s", servicepath(state.ServiceID, state.ID), err)
		} else if !exists {
			t.Errorf("Deleted service node %s", state.ID)
		}
	}

	shutdown = make(chan interface{})
	listener.shutdown = shutdown
	go listener.listenHost(done, ehostID)

	// remove the ephemeral node
	<-time.After(time.Second)
	t.Log("Removing the ephemeral node: ", ehostpath)
	if err := conn.Delete(ehostpath); err != nil {
		t.Fatalf("Error trying to remove node %s: %s", ehostpath, err)
	}
	if id := <-done; id != ehostID {
		t.Errorf("Expected eHost %s; Actual %s", ehostID, id)
	}

	// verify that all of the service states were removed
	for _, state := range states {
		// verify the state has been removed
		if exists, err := zkutils.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
			t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
		} else if exists {
			t.Errorf("Failed to delete host node %s", state.ID)
		} else if exists, err := zkutils.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
			t.Fatalf("Error checking path %s: %s", servicepath(state.ServiceID, state.ID), err)
		} else if exists {
			t.Errorf("Failed to delete service node %s", state.ID)
		}
	}
}

func TestHostRegistryListener_unregister(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	listener, err := NewHostRegistryListener(conn)
	if err != nil {
		t.Fatalf("Could not initialize host registry listener: %s", err)
	}

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
			if exists, err := zkutils.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
			} else if exists {
				t.Errorf("Failed to delete host node %s", state.ID)
			} else if exists, err := zkutils.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", servicepath(state.ServiceID, state.ID), err)
			} else if exists {
				t.Errorf("Failed to delete service node %s", state.ID)
			} else {
				removed++
			}
		} else {
			// verify the state has been preserved
			if exists, err := zkutils.PathExists(conn, hostpath(state.HostID, state.ID)); err != nil {
				t.Fatalf("Error checking path %s: %s", hostpath(state.HostID, state.ID), err)
			} else if !exists {
				t.Errorf("Deleted host node %s", state.ID)
			} else if exists, err := zkutils.PathExists(conn, servicepath(state.ServiceID, state.ID)); err != nil {
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

func TestHostRegistryListener_GetHosts(t *testing.T) {
}
