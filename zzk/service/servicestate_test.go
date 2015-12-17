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

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"

	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestGetServiceStatus(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	err = conn.CreateDir(servicepath())
	c.Assert(err, IsNil)

	// Add a service
	svc := service.Service{ID: "test-service-1", Instances: 3}
	err = UpdateService(conn, svc, false, false)
	c.Assert(err, IsNil)

	// Add a host
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

	// Case 1: Zero service states
	statusmap, err := GetServiceStatus(conn, svc.ID)
	c.Assert(err, IsNil)
	c.Assert(statusmap, NotNil)
	c.Assert(statusmap, DeepEquals, map[string]dao.ServiceStatus{})

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
	stateIDs := addstates("test-host-1", &svc, 3)

	states := make(map[string]servicestate.ServiceState)
	updatestate := func(stateID, serviceID string, update func(state *servicestate.ServiceState)) servicestate.ServiceState {
		var node ServiceStateNode
		err = conn.Get(servicepath(serviceID, stateID), &node)
		c.Assert(err, IsNil)
		update(node.ServiceState)
		err = conn.Set(servicepath(serviceID, stateID), &node)
		c.Assert(err, IsNil)

		return *node.ServiceState
	}

	// State 0 started
	states[stateIDs[0]] = updatestate(stateIDs[0], svc.ID, func(s *servicestate.ServiceState) {
		s.Started = time.Now().UTC()
	})

	// State 1 paused
	states[stateIDs[1]] = updatestate(stateIDs[1], svc.ID, func(s *servicestate.ServiceState) {
		s.Started = time.Now().UTC()
		s.Paused = true
	})

	// State 2 stopped (no update)
	states[stateIDs[2]] = updatestate(stateIDs[2], svc.ID, func(s *servicestate.ServiceState) {})

	c.Log("Desired state is RUN")
	statusmap, err = GetServiceStatus(conn, svc.ID)
	c.Assert(err, IsNil)
	c.Assert(statusmap, DeepEquals, map[string]dao.ServiceStatus{
		stateIDs[0]: dao.ServiceStatus{states[stateIDs[0]], dao.Running, nil},
		stateIDs[1]: dao.ServiceStatus{states[stateIDs[1]], dao.Resuming, nil},
		stateIDs[2]: dao.ServiceStatus{states[stateIDs[2]], dao.Starting, nil},
	})

	c.Log("Desired state is PAUSE")
	for _, state := range stateIDs {
		err := pauseInstance(conn, "test-host-1", state)
		c.Assert(err, IsNil)
	}
	statusmap, err = GetServiceStatus(conn, svc.ID)
	c.Assert(err, IsNil)
	c.Assert(statusmap, DeepEquals, map[string]dao.ServiceStatus{
		stateIDs[0]: dao.ServiceStatus{states[stateIDs[0]], dao.Pausing, nil},
		stateIDs[1]: dao.ServiceStatus{states[stateIDs[1]], dao.Paused, nil},
		stateIDs[2]: dao.ServiceStatus{states[stateIDs[2]], dao.Stopped, nil},
	})

	c.Log("Desired state is STOP")
	for _, state := range stateIDs {
		err := StopServiceInstance(conn, "test-host-1", state)
		c.Assert(err, IsNil)
	}
	statusmap, err = GetServiceStatus(conn, svc.ID)
	c.Assert(err, IsNil)
	c.Assert(statusmap, DeepEquals, map[string]dao.ServiceStatus{
		stateIDs[0]: dao.ServiceStatus{states[stateIDs[0]], dao.Stopping, nil},
		stateIDs[1]: dao.ServiceStatus{states[stateIDs[1]], dao.Stopping, nil},
		stateIDs[2]: dao.ServiceStatus{states[stateIDs[2]], dao.Stopped, nil},
	})
}
