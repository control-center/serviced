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

package zookeeper

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	zzktest "github.com/control-center/serviced/zzk/test"
)

func TestLeader(t *testing.T) {

	/* start the cluster */
	zzkServer := &zzktest.ZZKServer{}
	if err := zzkServer.Start(); err != nil {
		t.Fatalf("Could not start zookeeper: %s", err)
	}
	defer zzkServer.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", zzkServer.Port)}

	// setup the driver
	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	// create a connection
	conn, err := drv.GetConnection(dsn, "/bossPath")
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	// create  a leader and TakeLead
	leader1Node := &testNodeT{
		Name: "leader1",
	}
	leader1, err := conn.NewLeader("/like/a/boss")
	if err != nil {
		t.Fatal("unexpected error initializing leader")
	}
	leaderDone1 := make(chan struct{})
	defer close(leaderDone1)
	_, err = leader1.TakeLead(leader1Node, leaderDone1)
	if err != nil {
		t.Fatalf("could not take lead! %s", err)
	}

	leader2Node := &testNodeT{
		Name: "leader2",
	}
	leader2, err := conn.NewLeader("/like/a/boss")
	if err != nil {
		t.Fatal("unexpected error initializing leader")
	}
	leader2Response := make(chan error)
	leaderDone2 := make(chan struct{})
	defer close(leaderDone2)
	go func() {
		_, err := leader2.TakeLead(leader2Node, leaderDone2)
		leader2Response <- err
	}()

	select {
	case err = <-leader2Response:
		t.Fatalf("expected leader2 to block!: %s", err)
	case <-time.After(time.Second):
	}

	currentLeaderNode := &testNodeT{
		Name: "",
	}
	// get current Leader
	currentLeader, err := conn.NewLeader("/like/a/boss")
	if err != nil {
		t.Fatalf("unexpected error initializing leader: %s", err)
	}
	err = currentLeader.Current(currentLeaderNode)
	if err != nil {
		t.Fatalf("unexpected error getting current leader:%s", err)
	}

	if currentLeaderNode.Name != leader1Node.Name {
		t.Fatalf("expected leader %s , got %s", currentLeaderNode.Name, leader1Node.Name)
	}

	// let the first leader go
	err = leader1.ReleaseLead()
	if err != nil {
		t.Fatal("unexpected error releasing leader1 ")
	}

	select {
	case err = <-leader2Response:
		if err != nil {
			t.Fatalf("unexpected error when leader 1 was release and waiting on leader2: %s", err)

		}
	case <-time.After(time.Second * 3):
		t.Fatalf("expected leader2 to take over but we blocked")
	}

}
