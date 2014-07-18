// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package snapshot

import (
	"fmt"
	"testing"
	"time"

	"github.com/control-center/serviced/coordinator/client"
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
	listener := NewSnapshotListener(conn, handler)
	go listener.Listen(shutdown)

	// send success snapshot
	if err := Send(conn, "service-id-success"); err != nil {
		t.Fatalf("Could not send success snapshot")
	}

	// wait for result
	var snapshot Snapshot
	if err := Recv(conn, "service-id-success", &snapshot); err != nil {
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
	if err := Send(conn, "service-id-failure"); err != nil {
		t.Fatal("Could not send failure snapshot: ", err)
	}

	// shutdown and wait for result
	if err := Recv(conn, "service-id-failure", &snapshot); err != nil {
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