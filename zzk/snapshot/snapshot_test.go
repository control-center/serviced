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

package snapshot

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
)

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

func TestSnapshotListener_Listen(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()

	handler := &TestSnapshotHandler{
		ResultMap: map[string]SnapshotResult{
			"service-id-success": SnapshotResult{time.Second, "success-label", nil},
			"service-id-failure": SnapshotResult{time.Second, "", fmt.Errorf("failure-label")},
		},
	}

	t.Log("Create snapshots and shutdown")
	shutdown := make(chan interface{})
	listener := NewSnapshotListener(handler)
	go zzk.Listen(shutdown, make(chan error, 1), conn, listener)

	// send success snapshot
	var snapshot Snapshot
	if nodeID, err := Send(conn, "service-id-success"); err != nil {
		t.Fatalf("Could not send success snapshot")
	} else if err := Recv(conn, nodeID, &snapshot); err != nil {
		t.Fatalf("Could not receieve success snapshot")
	}

	// verify fields
	result := handler.ResultMap["service-id-success"]
	if snapshot.ServiceID != "service-id-success" {
		t.Errorf("MISMATCH: Service IDs do not match 'service-id-success' != %s", snapshot.ServiceID)
	} else if snapshot.Label != result.Label {
		t.Errorf("MISMATCH: Labels do not match '%s' != '%s'", result.Label, snapshot.Label)
	} else if result.Err != nil {
		t.Errorf("MISMATCH: Err msgs do not match '%s' != '%s'", result.Err, snapshot.Err)
	}

	// send fail snapshot and shutdown
	if nodeID, err := Send(conn, "service-id-failure"); err != nil {
		t.Fatal("Could not send failure snapshot: ", err)
	} else if err := Recv(conn, nodeID, &snapshot); err != nil {
		t.Fatal("Could not receive failure snapshot: ", err)
	}

	// verify the fields
	result = handler.ResultMap["service-id-failure"]
	if snapshot.ServiceID != "service-id-failure" {
		t.Errorf("MISMATCH: Service IDs do not match 'service-id-success' != %s", snapshot.ServiceID)
	} else if snapshot.Label != result.Label {
		t.Errorf("MISMATCH: Labels do not match '%s' != '%s'", result.Label, snapshot.Label)
	} else if result.Err == nil || result.Err.Error() != snapshot.Err {
		t.Errorf("MISMATCH: Err msgs do not match '%s' != '%s'", result.Err, snapshot.Err)
	}

	// make sure listener shuts down
	close(shutdown)
}

func TestSnapshotListener_Spawn(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestSnapshotHandler{
		ResultMap: map[string]SnapshotResult{
			"service-id-success": SnapshotResult{time.Second, "success-label", nil},
			"service-id-failure": SnapshotResult{time.Second, "", fmt.Errorf("failure-label")},
		},
	}
	listener := NewSnapshotListener(handler)
	listener.SetConnection(conn)
	var wg sync.WaitGroup

	// send snapshots
	t.Log("Sending successful snapshot")
	nodeID, err := Send(conn, "service-id-success")
	if err != nil {
		t.Fatalf("Could not send success snapshot")
	}
	var snapshot Snapshot
	event, err := conn.GetW(listener.GetPath(nodeID), &snapshot)
	if err != nil {
		t.Fatalf("Could not look up %s: %s", listener.GetPath("service-id-success"), err)
	}
	shutdown := make(chan interface{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(shutdown, "service-id-success")
	}()
	<-event
	t.Logf("Shutting down listener")
	close(shutdown)
	wg.Wait()
	if err := Recv(conn, "service-id-success", &snapshot); err != nil {
		t.Fatalf("Could not receive success snapshot")
	}

	// verify fields
	result := handler.ResultMap["service-id-success"]
	if snapshot.ServiceID != "service-id-success" {
		t.Errorf("MISMATCH: Service IDs do not match 'service-id-success' != %s", snapshot.ServiceID)
	} else if snapshot.Label != result.Label {
		t.Errorf("MISMATCH: Labels do not match '%s' != '%s'", result.Label, snapshot.Label)
	} else if result.Err != nil {
		t.Errorf("MISMATCH: Err msgs do not match '%s' != '%s'", result.Err, snapshot.Err)
	}

	t.Log("Sending failure snapshot")
	nodeID, err = Send(conn, "service-id-failure")
	if err != nil {
		t.Fatalf("Could not send success snapshot")
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener.Spawn(make(<-chan interface{}), "service-id-failure")
	}()
	if err := Recv(conn, nodeID, &snapshot); err != nil {
		t.Fatalf("Could not receive success snapshot")
	}
	wg.Wait()

	// verify the fields
	result = handler.ResultMap["service-id-failure"]
	if snapshot.ServiceID != "service-id-failure" {
		t.Errorf("MISMATCH: Service IDs do not match 'service-id-success' != %s", snapshot.ServiceID)
	} else if snapshot.Label != result.Label {
		t.Errorf("MISMATCH: Labels do not match '%s' != '%s'", result.Label, snapshot.Label)
	} else if result.Err == nil || result.Err.Error() != snapshot.Err {
		t.Errorf("MISMATCH: Err msgs do not match '%s' != '%s'", result.Err, snapshot.Err)
	}
}
