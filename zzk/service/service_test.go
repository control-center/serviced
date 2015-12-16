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
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"

	. "gopkg.in/check.v1"
)

type TestServiceHandler struct {
	Host *host.Host
	Err  error
}

func (handler *TestServiceHandler) SelectHost(svc *service.Service) (*host.Host, error) {
	return handler.Host, handler.Err
}

func (t *ZZKTest) TestServiceListener_NoHostState(c *C) {
	conn, err := zzk.GetLocalConnection("/TestServiceListener_NoHostState")
	c.Assert(err, IsNil)
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	_, err = registerHost(conn, handler.Host)
	c.Assert(err, IsNil)

	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/test-host-1")
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	done := make(chan interface{})
	listener := NewServiceListener(handler)
	go func() {
		zzk.Listen(shutdown, make(chan error, 1), conn, listener)
		close(done)
	}()

	svc := service.Service{
		ID:           "test-service-1",
		DesiredState: int(service.SVCRun),
		Instances:    1,
	}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	// get the instance id
	getInstances := func(svc *service.Service) []string {
		timeout := time.After(time.Minute)
		for {
			var instanceIDs []string
			childWDone := make(chan struct{})
			defer close(childWDone)
			stateIDs, ev, err := conn.ChildrenW(servicepath(svc.ID), childWDone)
			c.Assert(err, IsNil)
			for _, stateID := range stateIDs {
				var state servicestate.ServiceState
				err := conn.Get(servicepath(svc.ID, stateID), &ServiceStateNode{ServiceState: &state})
				c.Assert(err, IsNil)
				err = updateInstance(conn, state.HostID, state.ID, func(_ *HostState, _ *servicestate.ServiceState) {})
				if err != nil {
					c.Assert(err, Equals, client.ErrNoNode)
				} else {
					instanceIDs = append(instanceIDs, stateID)
				}
			}
			if len(instanceIDs) == svc.Instances {
				return instanceIDs
			}
			select {
			case <-ev:
			case <-timeout:
				c.Fatalf("wait time exceeded timeout!")
			}
		}
	}

	instanceIDs := getInstances(&svc)
	// delete the host path
	err = conn.Delete(hostpath("test-host-1", instanceIDs[0]))
	c.Assert(err, IsNil)
	c.Assert(getInstances(&svc), Not(DeepEquals), instanceIDs)
	close(shutdown)
	<-done
}

func (t *ZZKTest) TestServiceListener_Listen(c *C) {
	conn, err := zzk.GetLocalConnection("/TestServiceListener_Listen")
	c.Assert(err, IsNil)
	err = conn.CreateDir(zkService)
	c.Assert(err, IsNil)
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	_, err = registerHost(conn, handler.Host)
	c.Assert(err, IsNil)

	err = conn.CreateDir("/hosts/test-host-1")
	c.Assert(err, IsNil)

	c.Log("Start and stop listener with no services")
	shutdown := make(chan interface{})
	done := make(chan interface{})
	listener := NewServiceListener(handler)
	go func() {
		zzk.Listen(shutdown, make(chan error, 1), conn, listener)
		close(done)
	}()

	<-time.After(2 * time.Second)
	c.Log("shutting down listener with no services")
	close(shutdown)
	<-done

	c.Log("Start and stop listener with multiple services")
	shutdown = make(chan interface{})
	done = make(chan interface{})
	go func() {
		zzk.Listen(shutdown, make(chan error, 1), conn, listener)
		close(done)
	}()

	svcs := []service.Service{
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

	for i := range svcs {
		err := UpdateService(conn, svcs[i], false, false)
		c.Assert(err, IsNil)
	}

	// wait for instances to start
	for {
		rss, err := LoadRunningServices(conn)
		c.Assert(err, IsNil)
		if count := len(rss); count < 5 {
			<-time.After(time.Second)
		} else {
			break
		}
	}

	// shutdown
	c.Log("services started, now shutting down")
	close(shutdown)
	<-done

}

func (t *ZZKTest) TestServiceListener_Spawn(c *C) {
	conn, err := zzk.GetLocalConnection("/TestServiceListener_Spawn")
	c.Assert(err, IsNil)
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	_, err = registerHost(conn, handler.Host)
	c.Assert(err, IsNil)

	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/test-host-1")
	c.Assert(err, IsNil)

	// Add 1 service
	svc := service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

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
	c.Log("Signaling shutdown for service listener")
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

		childWDone := make(chan struct{})
		defer close(childWDone)
		for {
			stateIDs, event, err = conn.ChildrenW(servicepath(svc.ID), childWDone)
			c.Assert(err, IsNil)
			if count := len(stateIDs); count == svc.Instances {
				break
			}
			<-event
		}

		for _, ssID := range stateIDs {
			var hs HostState
			hpath := hostpath(handler.Host.ID, ssID)
			err = conn.Get(hpath, &hs)
			c.Assert(err, IsNil)
			if hs.DesiredState == int(service.SVCRun) {
				count++
			}
		}
		return count
	}

	c.Log("Starting service with 2 instances")
	svc.Instances = 2
	svc.DesiredState = int(service.SVCRun)
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)
	c.Assert(getInstances(), Equals, svc.Instances)

	c.Log("Pause service")
	svc.DesiredState = int(service.SVCPause)
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	for {
		if count := getInstances(); count > 0 {
			c.Logf("Waiting for %d instances to pause", count)
			<-time.After(5 * time.Second)
		} else {
			break
		}
	}

	svc.DesiredState = int(service.SVCRun)
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)
	for {
		if count := getInstances(); count < svc.Instances {
			c.Logf("Waiting for %d instances to resume", svc.Instances)
			<-time.After(5 * time.Second)
		} else {
			break
		}
	}

	// Stop service
	c.Log("Stopping service")
	svc.DesiredState = int(service.SVCStop)
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	for {
		if count := getInstances(); count > 0 {
			c.Logf("Waiting for %d instances to stop", count)
			<-time.After(5 * time.Second)
		} else {
			break
		}
	}

	// Remove the service
	c.Log("Removing service")
	err = conn.Delete(servicepath(svc.ID))
	c.Assert(err, IsNil)
	wg.Wait()
}

func (t *ZZKTest) TestServiceListener_getServiceStates(c *C) {
	conn, err := zzk.GetLocalConnection("/base_getServiceStates")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/test-host-1")
	c.Assert(err, IsNil)
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	eHostID, err := registerHost(conn, handler.Host)
	c.Assert(err, IsNil)
	svc := service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)
	listener := NewServiceListener(handler)
	listener.SetConnection(conn)

	c.Log("Starting 1 instance")
	rss, err := LoadRunningServicesByService(conn, svc.ID)
	svc.Instances = 1
	listener.sync(false, &svc, rss)
	stateIDs, err := conn.Children(hostpath(handler.Host.ID))
	c.Assert(err, IsNil)
	c.Assert(stateIDs, HasLen, svc.Instances)
	// unregister the host
	err = conn.Delete(eHostID)
	c.Assert(err, IsNil)
	rss, err = listener.getServiceStates(&svc, stateIDs)
	c.Assert(err, IsNil)
	c.Assert(rss, HasLen, 0)
}

func (t *ZZKTest) TestServiceListener_sync_restartAllOnInstanceChanged(c *C) {
	conn, err := zzk.GetLocalConnection("/base_sync_restartAllOnInstanceChanged")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/test-host-1")
	c.Assert(err, IsNil)
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	_, err = registerHost(conn, handler.Host)
	c.Assert(err, IsNil)
	svc := service.Service{
		ID:            "test-service-1",
		Endpoints:     make([]service.ServiceEndpoint, 1),
		ChangeOptions: []string{"restartAllOnInstanceChanged"},
	}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)
	listener := NewServiceListener(handler)
	listener.SetConnection(conn)
	rss, err := LoadRunningServicesByService(conn, svc.ID)

	// Start 5 instances and verify
	c.Log("Starting 5 instances")
	svc.Instances = 5
	listener.sync(false, &svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	c.Assert(err, IsNil)
	c.Assert(rss, HasLen, svc.Instances)

	// Add three more instances; SHOULD NOT CHANGE UNLESS ALL INSTANCES HAVE
	// BEEN REMOVED
	c.Log("Starting 3 more instances")
	svc.Instances = 8
	listener.sync(false, &svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	c.Assert(err, IsNil)
	c.Assert(rss, HasLen, 5)
}

func (t *ZZKTest) TestServiceListener_sync(c *C) {
	conn, err := zzk.GetLocalConnection("/base_sync")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/test-host-1")
	c.Assert(err, IsNil)
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	_, err = registerHost(conn, handler.Host)
	c.Assert(err, IsNil)
	svc := service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)
	listener := NewServiceListener(handler)
	listener.SetConnection(conn)

	rss, err := LoadRunningServicesByService(conn, svc.ID)
	c.Assert(err, IsNil)
	c.Assert(rss, HasLen, 0)

	// Start 5 instances and verify
	c.Log("Starting 5 instances")
	svc.Instances = 5
	listener.sync(false, &svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	c.Assert(err, IsNil)
	c.Assert(rss, HasLen, svc.Instances)

	usedInstanceID := make(map[int]*servicestate.ServiceState)
	for _, rs := range rss {
		var state servicestate.ServiceState
		spath := servicepath(svc.ID, rs.ID)
		err = conn.Get(spath, &ServiceStateNode{ServiceState: &state})
		c.Assert(err, IsNil)
		_, ok := usedInstanceID[state.InstanceID]
		c.Assert(ok, Equals, false)
		usedInstanceID[state.InstanceID] = &state

		var hs HostState
		hpath := hostpath(handler.Host.ID, rs.ID)
		err = conn.Get(hpath, &hs)
		c.Assert(err, IsNil)
		c.Assert(hs.DesiredState, Not(Equals), int(service.SVCStop))
	}

	// Start 3 instances and verify
	c.Log("Adding 3 more instances")
	svc.Instances = 8
	listener.sync(false, &svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	c.Assert(err, IsNil)
	c.Assert(rss, HasLen, svc.Instances)

	usedInstanceID = make(map[int]*servicestate.ServiceState)
	for _, rs := range rss {
		var state servicestate.ServiceState
		spath := servicepath(svc.ID, rs.ID)
		err := conn.Get(spath, &ServiceStateNode{ServiceState: &state})
		c.Assert(err, IsNil)
		_, ok := usedInstanceID[state.InstanceID]
		c.Assert(ok, Equals, false)
		usedInstanceID[state.InstanceID] = &state

		var hs HostState
		hpath := hostpath(handler.Host.ID, rs.ID)
		err = conn.Get(hpath, &hs)
		c.Assert(err, IsNil)
		c.Assert(hs.DesiredState, Not(Equals), int(service.SVCStop))
	}

	// Stop 4 instances
	c.Log("Stopping 4 instances")
	svc.Instances = 4
	listener.sync(false, &svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	c.Assert(err, IsNil)
	c.Assert(rss, HasLen, 8)

	var stopped []*HostState
	for _, rs := range rss {
		var hs HostState
		hpath := hostpath(handler.Host.ID, rs.ID)
		err := conn.Get(hpath, &hs)
		c.Assert(err, IsNil)
		if hs.DesiredState == int(service.SVCStop) {
			stopped = append(stopped, &hs)
		}
	}
	c.Assert(len(rss)-len(stopped), Equals, svc.Instances)

	// Remove 2 stopped instances
	c.Log("Removing 2 stopped instances")
	for i := 0; i < 2; i++ {
		hs := stopped[i]
		var state servicestate.ServiceState
		err := conn.Get(servicepath(hs.ServiceID, hs.ServiceStateID), &ServiceStateNode{ServiceState: &state})
		c.Assert(err, IsNil)
		err = removeInstance(conn, state.ServiceID, state.HostID, state.ID)
		c.Assert(err, IsNil)
	}

	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	c.Assert(err, IsNil)
	c.Assert(len(rss) < svc.Instances, Equals, false)

	// Start 1 instance
	c.Log("Adding 1 more instance")
	svc.Instances = 5
	listener.sync(false, &svc, rss)
	rss, err = LoadRunningServicesByHost(conn, handler.Host.ID)
	c.Assert(err, IsNil)
	c.Assert(len(rss) < svc.Instances, Equals, false)
}

func (t *ZZKTest) TestServiceListener_start(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/test-host-1")
	c.Assert(err, IsNil)
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	_, err = registerHost(conn, handler.Host)
	c.Assert(err, IsNil)

	// Add 1 instance for 1 host
	svc := service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	listener := NewServiceListener(handler)
	listener.SetConnection(conn)
	listener.start(&svc, []int{1})

	// Look up service instance
	var state servicestate.ServiceState
	children, err := conn.Children(listener.GetPath(svc.ID))
	c.Assert(err, IsNil)
	c.Assert(children, HasLen, 1)

	spath := listener.GetPath(svc.ID, children[0])
	err = conn.Get(spath, &ServiceStateNode{ServiceState: &state})
	c.Assert(err, IsNil)

	// Look up host state
	var hs HostState
	hpath := hostpath(handler.Host.ID, state.ID)
	err = conn.Get(hpath, &hs)
	c.Assert(err, IsNil)

	// Check values
	c.Check(state.ID, Equals, children[0])
	c.Check(state.ServiceID, Equals, svc.ID)
	c.Check(state.HostID, Equals, handler.Host.ID)
	c.Check(state.HostIP, Equals, handler.Host.IPAddr)
	c.Check(state.Endpoints, HasLen, len(svc.Endpoints))
	c.Check(hs.ServiceStateID, Equals, state.ID)
	c.Check(hs.HostID, Equals, handler.Host.ID)
	c.Check(hs.ServiceID, Equals, svc.ID)
	c.Check(hs.DesiredState, Equals, int(service.SVCRun))
}

func (t *ZZKTest) TestServiceListener_pause(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/test-host-1")
	c.Assert(err, IsNil)
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	_, err = registerHost(conn, handler.Host)
	c.Assert(err, IsNil)

	svc := service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	listener := NewServiceListener(handler)
	listener.SetConnection(conn)
	listener.start(&svc, []int{1})

	rss, err := LoadRunningServicesByHost(conn, handler.Host.ID)
	c.Assert(err, IsNil)
	c.Check(rss, HasLen, 1)

	// Pause instance
	listener.pause(rss)

	// Verify the state of the instance
	var hs HostState
	hpath := hostpath(handler.Host.ID, rss[0].ID)
	err = conn.Get(hpath, &hs)
	c.Assert(err, IsNil)
	c.Check(hs.DesiredState, Equals, int(service.SVCPause))
}

func (t *ZZKTest) TestServiceListener_stop(c *C) {
	conn, err := zzk.GetLocalConnection("/base")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)
	err = conn.CreateDir("/hosts/test-host-1")
	c.Assert(err, IsNil)
	handler := &TestServiceHandler{Host: &host.Host{ID: "test-host-1", IPAddr: "test-host-1-ip"}}
	_, err = registerHost(conn, handler.Host)
	c.Assert(err, IsNil)

	svc := service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	listener := NewServiceListener(handler)
	listener.SetConnection(conn)
	listener.start(&svc, []int{1, 2})

	rss, err := LoadRunningServicesByHost(conn, handler.Host.ID)
	c.Assert(err, IsNil)
	c.Assert(rss, HasLen, 2)

	// Stop 1 instance
	listener.stop(rss[:1])

	// Verify the state of the instances
	var hs HostState
	hpath := hostpath(handler.Host.ID, rss[0].ID)
	err = conn.Get(hpath, &hs)
	c.Assert(err, IsNil)
	c.Check(hs.DesiredState, Equals, int(service.SVCStop))

	hpath = hostpath(handler.Host.ID, rss[1].ID)
	err = conn.Get(hpath, &hs)
	c.Assert(err, IsNil)
	c.Check(hs.DesiredState, Equals, int(service.SVCRun))
}
