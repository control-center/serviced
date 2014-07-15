// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package zookeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
)

func TestLock(t *testing.T) {

	/* start the cluster */
	tc, err := zklib.StartTestCluster(1)
	if err != nil {
		t.Fatalf("could not start test zk cluster: %s", err)
	}
	defer os.RemoveAll(tc.Path)
	defer tc.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", tc.Servers[0].Port)}

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
	lock := conn.NewLock("/foo/bar")
	if err = lock.Lock(); err != nil {
		t.Fatalf("unexpected error aquiring lock: %s", err)
	}

	// create a second lock and test that a locking attempt blocks
	lock2 := conn.NewLock("/foo/bar")
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
