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

package docker

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/control-center/serviced/zzk"
	. "gopkg.in/check.v1"
)

type ZZKTest struct {
	zzk.ZZKTestSuite
}

var _ = Suite(&ZZKTest{})

func Test(t *testing.T) {
	TestingT(t)
}

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

func (t *ZZKTest) TestActionListener_Listen(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	handler := &TestActionHandler{
		ResultMap: map[string]ActionResult{
			"success": ActionResult{2 * time.Second, []byte("success"), nil},
			"failure": ActionResult{time.Second, []byte("message failure"), fmt.Errorf("failure")},
		},
	}

	c.Log("Start actions and shutdown")
	shutdown := make(chan interface{})
	done := make(chan interface{})

	listener := NewActionListener(handler, "test-host-1")
	go func() {
		zzk.Listen(shutdown, make(chan error, 1), conn, listener)
		close(done)
	}()

	// send actions
	var wg sync.WaitGroup

	sendAction := func(dockerID string, command []string) {
		getWDone := make(chan struct{})

		id, err := SendAction(conn, &Action{
			HostID:   listener.hostID,
			DockerID: dockerID,
			Command:  command,
		})
		c.Assert(err, IsNil)

		// There *might* be a race condition here if the node is processed before
		// we acquire the event data (see duration timeouts above)
		event, err := conn.GetW(actionPath(listener.hostID, id), &Action{}, getWDone)
		c.Assert(err, IsNil)

		wg.Add(1)
		go func() {
			defer close(getWDone)
			defer wg.Done()
			ev := <-event
			c.Logf("Received event: %+v", ev)
		}()
		return
	}

	sendAction("success", []string{"do", "some", "command"})
	sendAction("failure", []string{"do", "some", "bad", "command"})

	c.Log("Waiting for actions to complete")
	wg.Wait()
	c.Log("Actions completed")
	close(shutdown)
	<-done
}

func (t *ZZKTest) TestActionListener_Spawn(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	handler := &TestActionHandler{
		ResultMap: map[string]ActionResult{
			"success": ActionResult{time.Second, []byte("success"), nil},
			"failure": ActionResult{time.Second, []byte("message failure"), fmt.Errorf("failure")},
		},
	}
	listener := NewActionListener(handler, "test-host-1")
	listener.SetConnection(conn)

	// send actions
	c.Logf("Sending successful command")
	success, err := SendAction(conn, &Action{
		HostID:   listener.hostID,
		DockerID: "success",
		Command:  []string{"do", "some", "command"},
	})
	if err != nil {
		c.Fatalf("Could not send success action")
	}
	listener.Spawn(nil, success)

	c.Logf("Sending failure command")
	failure, err := SendAction(conn, &Action{
		HostID:   listener.hostID,
		DockerID: "failure",
		Command:  []string{"do", "some", "bad", "command"},
	})
	if err != nil {
		c.Fatalf("Could not send failure action")
	}
	listener.Spawn(make(<-chan interface{}), failure)
}
