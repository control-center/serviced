// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package docker

import (
	"fmt"
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client"
)

type ActionResult struct {
	Duration time.Duration
	Result   []byte
	Err      error
}

func (result *ActionResult) do() ([]byte, error) {
	<-time.After(result.Duration)
	return result.Result, result.Err
}

type TestActionHandler struct {
	ResultMap map[string]ActionResult
}

func (handler *TestActionHandler) AttachAndRun(dockerID string, command []string) ([]byte, error) {
	if result, ok := handler.ResultMap[dockerID]; ok {
		return result.do()
	}

	return nil, fmt.Errorf("action not found")
}

func TestActionListener_Spawn(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestActionHandler{
		ResultMap: map[string]ActionResult{
			"success": ActionResult{time.Second, []byte("success"), nil},
			"failure": ActionResult{time.Second, []byte("message failure"), fmt.Errorf("failure")},
		},
	}
	listener := NewActionListener(conn, handler, "test-host-1")

	// send actions
	t.Logf("Sending successful command")
	success, err := SendAction(conn, &Action{
		HostID:   listener.hostID,
		DockerID: "success",
		Command:  []string{"do", "some", "command"},
	})
	if err != nil {
		t.Fatalf("Could not send success action")
	}
	listener.Spawn(make(<-chan interface{}), success)

	t.Logf("Sending failure command")
	failure, err := SendAction(conn, &Action{
		HostID:   listener.hostID,
		DockerID: "failure",
		Command:  []string{"do", "some", "bad", "command"},
	})
	if err != nil {
		t.Fatalf("Could not send failure action")
	}
	listener.Spawn(make(<-chan interface{}), failure)
}