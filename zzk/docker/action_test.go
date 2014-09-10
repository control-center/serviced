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

package docker

import (
	"fmt"
	"testing"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
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

func TestActionListener_Listen(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()
	handler := &TestActionHandler{
		ResultMap: map[string]ActionResult{
			"success": ActionResult{2 * time.Second, []byte("success"), nil},
			"failure": ActionResult{time.Second, []byte("message failure"), fmt.Errorf("failure")},
		},
	}

	t.Log("Start actions and shutdown")
	shutdown := make(chan interface{})
	done := make(chan interface{})

	listener := NewActionListener(handler, "test-host-1")
	go func() {
		zzk.Listen(shutdown, make(chan error, 1), conn, listener)
		close(done)
	}()

	// send actions
	success, err := SendAction(conn, &Action{
		HostID:   listener.hostID,
		DockerID: "success",
		Command:  []string{"do", "some", "command"},
	})
	if err != nil {
		t.Fatal("Could not send success action")
	}
	successW, err := conn.GetW(actionPath(listener.hostID, success), &Action{})
	if err != nil {
		t.Fatal("Failed creating watch for success action: ", err)
	}

	failure, err := SendAction(conn, &Action{
		HostID:   listener.hostID,
		DockerID: "failure",
		Command:  []string{"do", "some", "bad", "command"},
	})
	if err != nil {
		t.Fatalf("Could not send fail action")
	}
	failureW, err := conn.GetW(actionPath(listener.hostID, failure), &Action{})
	if err != nil {
		t.Fatal("Failed creating watch for failure action: ", err)
	}

	// wait for actions to complete and shutdown
	<-successW
	<-failureW
	close(shutdown)
	<-done

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
	listener := NewActionListener(handler, "test-host-1")
	listener.SetConnection(conn)

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
