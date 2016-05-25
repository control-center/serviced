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

func TestLock(t *testing.T) {

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
	conn, err := drv.GetConnection(dsn, "/test/basePath")
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	// create  a lock & lock it
	lock, err := conn.NewLock("/foo/bar")
	if err != nil {
		t.Fatalf("unexpected error initializing lock: %s", err)
	}
	if err = lock.Lock(); err != nil {
		t.Fatalf("unexpected error aquiring lock: %s", err)
	}

	// create a second lock and test that a locking attempt blocks
	lock2, err := conn.NewLock("/foo/bar")
	if err != nil {
		t.Fatalf("unexpected error initializing lock: %s", err)
	}
	lock2Response := make(chan error)
	go func() {
		lock2Response <- lock2.Lock()
	}()
	select {
	case response := <-lock2Response:
		t.Fatalf("Expected second lock to block, got %s", response)
	case <-time.After(time.Second):
		t.Log("good, lock2 failed to lock.")
	}

	// free the first lock, and test if the second lock unblocks
	if err = lock.Unlock(); err != nil {
		t.Fatalf("unexpected error releasing lock: %s", err)
	}
	select {
	case response := <-lock2Response:
		if response != nil {
			t.Fatalf("Did not expect error when attempting second lock!")
		}
	case <-time.After(time.Second * 3):
		t.Fatal("timeout on second lock")
	}

	// check if the second lock cleans up
	if err = lock2.Unlock(); err != nil {
		t.Fatalf("unexpected error releasing lock: %s", err)
	}

}
