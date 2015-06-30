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
	"sync"
	"testing"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"
)

type TestServiceHandler struct {
	Host *host.Host
	Err  error
}

func (handler *TestServiceHandler) SelectHost(svc *service.Service) (*host.Host, error) {
	return handler.Host, handler.Err
}

func TestServiceListener_Listen(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}

	t.Log("Start and stop listener with no services")
	shutdown := make(chan interface{})
	done := make(chan interface{})
	listener := NewServiceListener(handler)
	go func() {
		zzk.Listen(shutdown, make(chan error, 1), conn, listener)
		close(done)
	}()

	<-time.After(2 * time.Second)
	t.Log("shutting down listener with no services")
	close(shutdown)
	<-done

	t.Log("Start and stop listener with multiple services")
	shutdown = make(chan interface{})
	done = make(chan interface{})
	go func() {
		zzk.Listen(shutdown, make(chan error, 1), conn, listener)
		close(done)
	}()

	svcs := []*service.Service{
		{
			ID:           "test-service-1",
			Endpoints:    make([]service.ServiceEndpoint, 1),
			DesiredState: int(service.SVCRun),
			Instances:    3,
		}, {
			ID:           "test-service-2",
			Endpoints:    make([]service.ServiceEndpoint, 1),
			DesiredState: int(service.SVCRun),
			Instances:    2,
		},
	}

	for _, s := range svcs {
		if err := conn.Create(servicepath(s.ID), &ServiceNode{Service: s}); err != nil {
			t.Fatalf("Could not create service %s: %s", s.ID, err)
		}
	}

	// wait for instances to start
	for {
		if rss, err := LoadRunningServices(conn); err != nil {
			t.Fatalf("Could not load running services: %s", err)
		} else if count := len(rss); count < 5 {
			<-time.After(time.Second)
		} else {
			break
		}
	}

	// shutdown
	t.Log("services started, now shutting down")
	close(shutdown)
	<-done

}

func TestServiceListener_Spawn(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}

	// Add 1 service
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Error trying to create %s: %s", svc.ID, err)
	}

	var wg sync.WaitGroup
	shutdown := make(chan interface{})
	listener := NewServiceListener(handler)
	listener.SetConnection(conn)
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(shutdown, svc.ID)
	}()

	// wait 3 seconds and shutdown
	<-time.After(3 * time.Second)
	t.Log("Signaling shutdown for service listener")
	close(shutdown)
	wg.Wait()

	// start listener with 2 instances and stop service
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(make(<-chan interface{}), svc.ID)
	}()

	getInstances := func() (count int) {
		var (
			stateIDs []string
			event    <-chan client.Event
			err      error
		)

		for {
			stateIDs, event, err = conn.ChildrenW(servicepath(svc.ID))
			if err != nil {
				t.Fatalf("Error looking up service states for %s: %s", svc.ID, err)
			}

			if count := len(stateIDs); count == svc.Instances {
				break
			}
			<-event
		}

		for _, ssID := range stateIDs {
			hpath := hostpath(handler.Host.ID, ssID)
			var hs HostState
			if err := conn.Get(hpath, &hs); err == client.ErrNoNode {
				// pass
			} else if err != nil {
				t.Fatalf("Error looking up instance %s: %s", ssID, err)
			}
			if hs.DesiredState == int(service.SVCRun) {
				count++
			}
		}
		return count
	}

	t.Log("Starting service with 2 instances")
	svc.Instances = 2
	svc.DesiredState = int(service.SVCRun)
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not update service %s: %s", svc.ID, err)
	}

	if count := getInstances(); count != svc.Instances {
		t.Errorf("Expected %d started instances; actual: %d", svc.Instances, count)
	}

	t.Log("Pause service")
	svc.DesiredState = int(service.SVCPause)
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not pause service %s: %s", svc.ID, err)
	}

	for {
		if count := getInstances(); count > 0 {
			t.Logf("Waiting for %d instances to pause", count)
			<-time.After(5 * time.Second)
		} else {
			break
		}
	}

	svc.DesiredState = int(service.SVCRun)
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not update service %s: %s", svc.ID, err)
	}
	for {
		if count := getInstances(); count < svc.Instances {
			t.Logf("Waiting for %d instances to resume", svc.Instances)
			<-time.After(5 * time.Second)
		} else {
			break
		}
	}

	// Stop service
	t.Log("Stopping service")
	svc.DesiredState = int(service.SVCStop)
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not update service %s: %s", svc.ID, err)
	}

	for {
		if count := getInstances(); count > 0 {
			t.Logf("Waiting for %d instances to stop", count)
			<-time.After(5 * time.Second)
		} else {
			break
		}
	}

	// Remove the service
	t.Log("Removing service")
	if err := conn.Delete(servicepath(svc.ID)); err != nil {
		t.Fatalf("Could not remove service %s at %s", svc.ID, err)
	}
	wg.Wait()
}

func TestServiceListener_clean(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	svc := &service.Service{
		ID: "test-service-1",
	}
	spath := servicepath(svc.ID)
	if err := conn.Create(spath, &ServiceNode{Service: svc}); err != nil {
		t.Fatalf("Error while creating node %s: %s", spath, err)
	}
	listener := NewServiceListener(handler)
	listener.SetConnection(conn)

	t.Log("Starting instances")
	svc.Instances = 2
	listener.sync(svc, []dao.RunningService{})
	rss, err := LoadRunningServicesByService(conn, svc.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", svc.ID, err)
	} else if count := len(rss); count != svc.Instances {
		t.Fatalf("Expected %d instances; Got: %d instances", count, svc.Instances)
	}

	err = listener.clean(&rss)
	if err != nil {
		t.Fatalf("Error while cleaning up instances: %s", err)
	} else if count := len(rss); count != svc.Instances {
		t.Fatalf("Expected %d instances; Got: %d instances", count, svc.Instances)
	}

	// Delete the host record for the first node
	id := rss[0].ID
	err = conn.Delete(hostpath(rss[0].HostID, id))
	if err != nil {
		t.Fatalf("Could not delete %s: %s", id, err)
	}

	err = listener.clean(&rss)
	if err != nil {
		t.Fatalf("Error while cleaning up instances: %s", err)
	} else if count := len(rss); count == 0 || count == svc.Instances {
		t.Fatalf("Mismatch on number of instances")
	}

	for _, rs := range rss {
		if rs.ID == id {
			t.Errorf("Did not clean node %s: %s", rs.ID, err)
		}
	}

	if ok, err := conn.Exists(servicepath(svc.ID, id)); err != nil && err != client.ErrNoNode {
		t.Fatalf("Could not check node %s: %s", servicepath(svc.ID, id), err)
	} else if ok {
		t.Errorf("Node exists %s: %s", servicepath(svc.ID, id), err)
	}
}

func TestServiceListener_sync_restartAllOnInstanceChanged(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	svc := &service.Service{
		ID:            "test-service-1",
		Endpoints:     make([]service.ServiceEndpoint, 1),
		ChangeOptions: []string{"restartAllOnInstanceChanged"},
	}
	spath := servicepath(svc.ID)
	if err := conn.Create(spath, &ServiceNode{Service: svc}); err != nil {
		t.Fatalf("Error while creating node %s: %s", spath, err)
	}
	listener := NewServiceListener(handler)
	listener.SetConnection(conn)
	rss, err := LoadRunningServicesByService(conn, svc.ID)

	// Start 5 instances and verify
	t.Log("Starting 5 instances")
	svc.Instances = 5
	listener.sync(svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", handler.Host.ID, err)
	} else if count := len(rss); count != svc.Instances {
		t.Errorf("MISMATCH: expected %d instances; actual %d", svc.Instances, count)
	}

	// Add three more instances; SHOULD NOT CHANGE UNLESS ALL INSTANCES HAVE
	// BEEN REMOVED
	t.Log("Starting 3 more instances")
	svc.Instances = 8
	listener.sync(svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", handler.Host.ID, err)
	} else if count := len(rss); count != 5 {
		t.Errorf("MISMATCH: expected %d instances; actual %d", 5, count)
	}
}

func TestServiceListener_sync(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	spath := servicepath(svc.ID)
	if err := conn.Create(spath, &ServiceNode{Service: svc}); err != nil {
		t.Fatalf("Error while creating node %s: %s", spath, err)
	}
	listener := NewServiceListener(handler)
	listener.SetConnection(conn)

	rss, err := LoadRunningServicesByService(conn, svc.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", svc.ID, err)
	} else if count := len(rss); count > 0 {
		t.Fatalf("Expected 0 instances; Got: %d", count)
	}

	// Start 5 instances and verify
	t.Log("Starting 5 instances")
	svc.Instances = 5
	listener.sync(svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", handler.Host.ID, err)
	} else if count := len(rss); count != svc.Instances {
		t.Errorf("MISMATCH: expected %d instances; actual %d", svc.Instances, count)
	}

	usedInstanceID := make(map[int]*servicestate.ServiceState)
	for _, rs := range rss {
		var state servicestate.ServiceState
		spath := servicepath(svc.ID, rs.ID)
		if err := conn.Get(spath, &ServiceStateNode{ServiceState: &state}); err != nil {
			t.Fatalf("Error while looking up %s: %s", spath, err)
		} else if ss, ok := usedInstanceID[state.InstanceID]; ok {
			t.Errorf("DUPLICATE: found 2 instances with the same id: [%v] [%v]", ss, state)
		}
		usedInstanceID[state.InstanceID] = &state

		var hs HostState
		hpath := hostpath(handler.Host.ID, rs.ID)
		if err := conn.Get(hpath, &hs); err != nil {
			t.Fatalf("Error while looking up %s: %s", hpath, err)
		} else if hs.DesiredState == int(service.SVCStop) {
			t.Errorf("Found stopped service at %s", hpath)
		}
	}

	// Start 3 instances and verify
	t.Log("Adding 3 more instances")
	svc.Instances = 8
	listener.sync(svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", handler.Host.ID, err)
	} else if count := len(rss); count != svc.Instances {
		t.Errorf("MISMATCH: expected %d instances; actual %d", svc.Instances, count)
	}

	usedInstanceID = make(map[int]*servicestate.ServiceState)
	for _, rs := range rss {
		var state servicestate.ServiceState
		spath := servicepath(svc.ID, rs.ID)
		if err := conn.Get(spath, &ServiceStateNode{ServiceState: &state}); err != nil {
			t.Fatalf("Error while looking up %s: %s", spath, err)
		} else if ss, ok := usedInstanceID[state.InstanceID]; ok {
			t.Errorf("DUPLICATE: found 2 instances with the same id: [%v] [%v]", ss, state)
		}
		usedInstanceID[state.InstanceID] = &state

		var hs HostState
		hpath := hostpath(handler.Host.ID, rs.ID)
		if err := conn.Get(hpath, &hs); err != nil {
			t.Fatalf("Error while looking up %s: %s", hpath, err)
		} else if hs.DesiredState == int(service.SVCStop) {
			t.Errorf("Found stopped service at %s", hpath)
		}
	}

	// Stop 4 instances
	t.Log("Stopping 4 instances")
	svc.Instances = 4
	listener.sync(svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", handler.Host.ID, err)
	} else if count := len(rss); count != 8 { // Services are scheduled to be stopped, but haven't yet
		t.Errorf("MISMATCH: expected %d instances; actual: %d", svc.Instances, count)
	}

	var stopped []*HostState
	for _, rs := range rss {
		var hs HostState
		hpath := hostpath(handler.Host.ID, rs.ID)
		if err := conn.Get(hpath, &hs); err != nil {
			t.Fatalf("Error while looking up %s: %s", hpath, err)
		} else if hs.DesiredState == int(service.SVCStop) {
			stopped = append(stopped, &hs)
		}
	}
	if running := len(rss) - len(stopped); svc.Instances != running {
		t.Errorf("MISMATCH: expected %d running instances; actual %d", svc.Instances, running)
	}

	// Remove 2 stopped instances
	t.Log("Removing 2 stopped instances")
	for i := 0; i < 2; i++ {
		hs := stopped[i]
		var state servicestate.ServiceState
		if err := conn.Get(servicepath(hs.ServiceID, hs.ServiceStateID), &ServiceStateNode{ServiceState: &state}); err != nil {
			t.Fatalf("Error while getting %s: %s", hs.ServiceStateID, err)
		} else if err := removeInstance(conn, state.ServiceID, state.HostID, state.ID); err != nil {
			t.Fatalf("Error while deleting %s: %s", hs.ServiceStateID, err)
		}
	}

	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", handler.Host.ID, err)
	} else if count := len(rss); count < svc.Instances {
		t.Errorf("MISMATCH: expected AT LEAST %d running instances; actual %d", svc.Instances, count)
	}

	// Start 1 instance
	t.Log("Adding 1 more instance")
	svc.Instances = 5
	listener.sync(svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", handler.Host.ID, err)
	} else if count := len(rss); count < svc.Instances {
		t.Errorf("MISMATCH: expected AT LEAST %d running instances; actual %d", svc.Instances, count)
	}
}

func TestServiceListener_start(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}

	// Add 1 instance for 1 host
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Error while trying to add service %s: %s", svc.ID, err)
	}

	listener := NewServiceListener(handler)
	listener.SetConnection(conn)
	listener.start(svc, []int{1})

	// Look up service instance
	var state servicestate.ServiceState
	children, err := conn.Children(listener.GetPath(svc.ID))
	if err != nil {
		t.Fatalf("Error while looking up service instances: %s", err)
	}
	if len(children) != 1 {
		t.Fatalf("Wrong number of instances found in path: %s", listener.GetPath(svc.ID))
	}

	spath := listener.GetPath(svc.ID, children[0])
	if err := conn.Get(spath, &ServiceStateNode{ServiceState: &state}); err != nil {
		t.Fatalf("Error while looking up %s: %s", spath, err)
	}

	// Look up host state
	var hs HostState
	hpath := hostpath(handler.Host.ID, state.ID)
	if err := conn.Get(hpath, &hs); err != nil {
		t.Fatalf("Error while looking up %s: %s", hpath, err)
	}

	// Check values
	if state.ID != children[0] {
		t.Errorf("MISMATCH: service state id (%s) != nide id (%s): %s", state.ID, children[0], spath)
	}
	if state.ServiceID != svc.ID {
		t.Errorf("MISMATCH: service ids do ot match (%s != %s): %s", state.ServiceID, svc.ID, spath)
	}
	if state.HostID != handler.Host.ID {
		t.Errorf("MISMATCH: host ids do not match (%s != %s): %s", state.HostID, handler.Host.ID, spath)
	}
	if state.HostIP != handler.Host.IPAddr {
		t.Errorf("MISMATCH: host ips do not match (%s != %s): %s", state.HostIP, handler.Host.IPAddr, spath)
	}
	if len(state.Endpoints) != len(svc.Endpoints) {
		t.Errorf("MISMATCH: wrong number of endpoints (%d != %d): %s", len(state.Endpoints), len(svc.Endpoints), spath)
	}

	if hs.ServiceStateID != state.ID {
		t.Errorf("MISMATCH: host state id (%s) != node id (%s): %s", hs.ServiceStateID, state.ID, hpath)
	}
	if hs.HostID != handler.Host.ID {
		t.Errorf("MISMATCH: host ids do not match (%s != %s): %s", hs.HostID, handler.Host.ID, hpath)
	}
	if hs.ServiceID != svc.ID {
		t.Errorf("MISMATCH: service ids do not match (%s != %s): %s", hs.ServiceID, svc.ID, hpath)
	}
	if hs.DesiredState != int(service.SVCRun) {
		t.Errorf("MISMATCH: incorrect service state (%d != %d): %s", hs.DesiredState, service.SVCRun, hpath)
	}
}

func TestServiceListener_pause(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}

	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Error while trying to add service %s: %s", svc.ID, err)
	}

	listener := NewServiceListener(handler)
	listener.SetConnection(conn)
	listener.start(svc, []int{1})

	rss, err := LoadRunningServicesByHost(conn, handler.Host.ID)
	if err != nil {
		t.Fatalf("Error which looking up %s: %s", handler.Host.ID, err)
	} else if count := len(rss); count != 1 {
		t.Errorf("MISMATCH: expected 1 children; found %d", count)
	}

	// Pause instance
	listener.pause(rss)

	// Verify the state of the instance
	var hs HostState
	hpath := hostpath(handler.Host.ID, rss[0].ID)
	if err := conn.Get(hpath, &hs); err != nil {
		t.Fatalf("Error while looking up %s: %s", hpath, err)
	} else if hs.DesiredState != int(service.SVCPause) {
		t.Errorf("MISMATCH: expected service state paused (%d); actual (%d): %s", service.SVCPause, hs.DesiredState, hpath)
	}
}

func TestServiceListener_stop(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}

	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Error while trying to add service %s: %s", svc.ID, err)
	}

	listener := NewServiceListener(handler)
	listener.SetConnection(conn)
	listener.start(svc, []int{1, 2})

	rss, err := LoadRunningServicesByHost(conn, handler.Host.ID)
	if err != nil {
		t.Fatalf("Error while looking up %s: %s", handler.Host.ID, err)
	} else if count := len(rss); count != 2 {
		t.Errorf("MISMATCH: expected 2 children; found %d", count)
	}
	// Stop 1 instance
	listener.stop(rss[:1])

	// Verify the state of the instances
	var hs HostState
	hpath := hostpath(handler.Host.ID, rss[0].ID)
	if err := conn.Get(hpath, &hs); err != nil {
		t.Fatalf("Error while looking up %s: %s", hpath, err)
	} else if hs.DesiredState != int(service.SVCStop) {
		t.Errorf("MISMATCH: expected service stopped (%d); actual (%d): %s", service.SVCStop, hs.DesiredState, hpath)
	}

	hpath = hostpath(handler.Host.ID, rss[1].ID)
	if err := conn.Get(hpath, &hs); err != nil {
		t.Fatalf("Error while looking up %s: %s", hpath, err)
	} else if hs.DesiredState != int(service.SVCRun) {
		t.Errorf("MISMATCH: expected service started (%d); actual (%d): %s", service.SVCRun, hs.DesiredState, hpath)
	}
}
