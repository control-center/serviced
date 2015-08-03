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

// +build integration

package snapshot

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/control-center/serviced/zzk"

	. "gopkg.in/check.v1"
)

var _ = Suite(&ZZKTest{})

type ZZKTest struct {
	zzk.ZZKTestSuite
}

func Test(t *testing.T) {
	TestingT(t)
}

type SnapshotResult struct {
	Duration time.Duration
	Label    string
	Err      error
}

func (result *SnapshotResult) do() (string, error) {
	<-time.After(result.Duration)
	return result.Label, result.Err
}

type TestSnapshotHandler struct {
	ResultMap map[string]SnapshotResult
}

func (handler *TestSnapshotHandler) TakeSnapshot(serviceID string) (string, error) {
	if result, ok := handler.ResultMap[serviceID]; ok {
		return result.do()
	}

	return "", fmt.Errorf("service ID not found")
}

func (handler *TestSnapshotHandler) expected(serviceID string) Snapshot {
	result := handler.ResultMap[serviceID]

	snapshot := Snapshot{ServiceID: serviceID, Label: result.Label}
	if result.Err != nil {
		snapshot.Err = result.Err.Error()
	}
	return snapshot
}

func (t *ZZKTest) TestSnapshotListener_Listen(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	handler := &TestSnapshotHandler{
		ResultMap: map[string]SnapshotResult{
			"service-id-success": SnapshotResult{time.Second, "success-label", nil},
			"service-id-failure": SnapshotResult{time.Second, "", fmt.Errorf("failure-label")},
		},
	}

	c.Log("Create snapshots and shutdown")
	shutdown := make(chan interface{})
	listener := NewSnapshotListener(handler)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		zzk.Listen(shutdown, make(chan error, 1), conn, listener)
	}()

	var actual Snapshot

	c.Log("Sending success snapshot")
	serviceID := "service-id-success"
	if nodeID, err := Send(conn, serviceID); err != nil {
		c.Errorf("Could not send success snasphot")
	} else if err := Recv(conn, nodeID, &actual); err != nil {
		c.Errorf("Could not receieve success snapshot")
	}
	actual.SetVersion(nil)
	c.Assert(actual, Equals, handler.expected(serviceID))

	c.Log("Sending failure snapshot")
	serviceID = "service-id-failure"
	if nodeID, err := Send(conn, serviceID); err != nil {
		c.Errorf("Could not send failure snapshot: %s", err)
	} else if err := Recv(conn, nodeID, &actual); err != nil {
		c.Errorf("Could not receive failure snapshot: %s", err)
	}
	actual.SetVersion(nil)
	c.Assert(actual, Equals, handler.expected(serviceID))

	c.Log("Shutting down the listener")
	close(shutdown)
	wg.Wait()
}

func (t *ZZKTest) TestSnapshotListener_Spawn(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	handler := &TestSnapshotHandler{
		ResultMap: map[string]SnapshotResult{
			"service-id-success": SnapshotResult{time.Second, "success-label", nil},
			"service-id-failure": SnapshotResult{time.Second, "", fmt.Errorf("failure-label")},
		},
	}
	listener := NewSnapshotListener(handler)
	listener.SetConnection(conn)

	send := func(serviceID string) {
		c.Logf("Sending snapshot %s", serviceID)
		nodeID, err := Send(conn, serviceID)
		c.Assert(err, IsNil)

		var node Snapshot
		event, err := conn.GetW(listener.GetPath(nodeID), &node)
		c.Assert(err, IsNil)
		node.SetVersion(nil)
		c.Assert(node, Equals, Snapshot{ServiceID: serviceID})

		var wg sync.WaitGroup
		wg.Add(1)
		shutdown := make(chan interface{})
		go func() {
			defer wg.Done()
			listener.Spawn(shutdown, nodeID)
		}()

		// wait for the node to change
		c.Logf("Waiting for %s to change", serviceID)
		select {
		case <-event:
		case <-time.After(15 * time.Second):
			// NOTE: you may have a race condition here if the timeout
			// on the snapshot exceeds the time to wait
			c.Errorf("timeout")
		}

		c.Logf("Shutting down listener for %s", serviceID)
		close(shutdown)
		wg.Wait()

		c.Logf("Verifying snapshot for %s", serviceID)
		var actual Snapshot
		err = Recv(conn, nodeID, &actual)
		actual.SetVersion(nil)
		c.Assert(err, IsNil)
		c.Assert(actual, Equals, handler.expected(serviceID))

		c.Logf("Verifying cleanup for %s", serviceID)
		exists, err := conn.Exists(listener.GetPath(nodeID))
		c.Assert(err, IsNil)
		c.Assert(exists, Equals, false)
	}

	send("service-id-success")
	send("service-id-failure")
}
