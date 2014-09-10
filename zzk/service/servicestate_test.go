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
	"testing"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
)

func TestGetServiceStatus(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()

	// Create the service
	svc := &service.Service{
		ID:        "test-service-1",
		Endpoints: make([]service.ServiceEndpoint, 1),
	}

	if err := UpdateService(conn, svc); err != nil {
		t.Fatalf("Could not add service %s: %s", svc.ID, err)
	}
	if err := AddHost(conn, &host.Host{ID: "test-host-1"}); err != nil {
		t.Fatalf("Could not register host: %s", err)
	}

	// 0 states
	statusmap, err := GetServiceStatus(conn, svc.ID)
	if err != nil {
		t.Fatalf("Could not get status for service %s: %s", svc.ID, err)
	}

	if statusmap != nil && len(statusmap) > 0 {
		t.Errorf("Expected 0 statuses returned; got %d", len(statusmap))
	}

	// Add service states
	var states []*servicestate.ServiceState
	for i := 0; i < 3; i++ {
		state, err := servicestate.BuildFromService(svc, "test-host-1")
		if err != nil {
			t.Fatalf("Could not generate instance from service %s", svc.ID)
		} else if err := addInstance(conn, state); err != nil {
			t.Fatalf("Could not add instance %s from service %s", state.ID, state.ServiceID)
		}
		states = append(states, state)
	}

	expected := make(map[string]dao.Status)
	// State 0 started
	states[0].Started = time.Now()
	if err := UpdateServiceState(conn, states[0]); err != nil {
		t.Fatalf("Could not \"start\" service state %s: %s", states[0].ID, err)
	}
	expected[states[0].ID] = dao.Running

	// State 1 paused
	states[1].Started = time.Now()
	states[1].Paused = true
	if err := UpdateServiceState(conn, states[1]); err != nil {
		t.Fatalf("Could not \"pause\" service state %s: %s", states[1].ID, err)
	}
	expected[states[1].ID] = dao.Resuming

	// State 2 stopped (no update)
	expected[states[2].ID] = dao.Starting

	t.Log("Desired state is RUN")
	statusmap, err = GetServiceStatus(conn, svc.ID)
	if err != nil {
		t.Fatalf("Could not get the status for service %s: %s", svc.ID, err)
	} else if len(statusmap) != len(states) {
		t.Errorf("MISMATCH: expected %d states; actual %d", len(states), len(statusmap))
	}

	// Verify
	for _, svcstatus := range statusmap {
		expect, ok := expected[svcstatus.State.ID]
		if !ok {
			t.Fatalf("Missing service state %s", svcstatus.State.ID)
		} else if expect != svcstatus.Status {
			t.Errorf("MISMATCH: expected %s; actual %s", expect, svcstatus.Status)
		}
	}

	t.Log("Desired state is PAUSE")
	for _, state := range states {
		if err := pauseInstance(conn, state.HostID, state.ID); err != nil {
			t.Fatalf("Could not pause instance %s: %s", state.ID, err)
		}
	}
	expected[states[0].ID] = dao.Pausing
	expected[states[1].ID] = dao.Paused
	expected[states[2].ID] = dao.Stopped

	statusmap, err = GetServiceStatus(conn, svc.ID)
	if err != nil {
		t.Fatalf("Could not get the status for service %s: %s", svc.ID, err)
	} else if len(statusmap) != len(states) {
		t.Errorf("MISMATCH: expected %d states; actual %d", len(states), len(statusmap))
	}

	// Verify
	for _, svcstatus := range statusmap {
		expect, ok := expected[svcstatus.State.ID]
		if !ok {
			t.Fatalf("Missing service state %s", svcstatus.State.ID)
		} else if expect != svcstatus.Status {
			t.Errorf("MISMATCH: expected %s; actual %s", expect, svcstatus.Status)
		}
	}

	t.Log("Desired state is STOP")
	for _, state := range states {
		if err := StopServiceInstance(conn, state.HostID, state.ID); err != nil {
			t.Fatalf("Could not stop instance %s: %s", state.ID, err)
		}
	}
	expected[states[0].ID] = dao.Stopping
	expected[states[1].ID] = dao.Stopping
	expected[states[2].ID] = dao.Stopped

	statusmap, err = GetServiceStatus(conn, svc.ID)
	if err != nil {
		t.Fatalf("Could not get the status for service %s: %s", svc.ID, err)
	} else if len(statusmap) != len(states) {
		t.Errorf("MISMATCH: expected %d states; actual %d", len(states), len(statusmap))
	}

	// Verify
	for _, svcstatus := range statusmap {
		expect, ok := expected[svcstatus.State.ID]
		if !ok {
			t.Fatalf("Missing service state %s", svcstatus.State.ID)
		} else if expect != svcstatus.Status {
			t.Errorf("MISMATCH: expected %s; actual %s", expect, svcstatus.Status)
		}
	}

}
