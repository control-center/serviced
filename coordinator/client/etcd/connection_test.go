// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
package etcd

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDriver(t *testing.T) {

	tc, err := NewTestCluster()
	if err != nil {
		t.Fatalf("Could not create a test etcd cluster: %s", err)
	}
	defer tc.Stop()

	dsnbytes, err := json.Marshal(DSN{Servers: tc.Machines(), Timeout: time.Second})
	if err != nil {
		t.Fatal("Error creating connection string")
	}
	dsnStr := string(dsnbytes)

	drv := Driver{}

	conn, err := drv.GetConnection(dsnStr)
	if err != nil {
		t.Fatalf("Could not create a connection: %s", err)
	}

	exists, err := conn.Exists("/foo")
	if err != nil {
		t.Fatalf("err calling exists: %s", err)
	}
	if exists {
		t.Fatal("foo should not exist")
	}

	err = conn.Delete("/foo")
	if err == nil {
		t.Fatalf("delete on non-existent object should fail")
	}

	err = conn.CreateDir("/foo")
	if err != nil {
		t.Fatalf("creating /foo should work: %s", err)
	}

	err = conn.Create("/foo/bar", []byte("test"))
	if err != nil {
		t.Fatalf("creating /foo/bar should work: %s", err)
	}

	exists, err = conn.Exists("/foo/bar")
	if err != nil {
		t.Fatalf("could not call exists: %s", err)
	}

	if !exists {
		t.Fatal("/foo/bar should not exist")
	}

	err = conn.Delete("/foo")
	if err != nil {
		t.Fatalf("delete of /foo should work: %s", err)
	}

	lockPath := "/foo/lock"
	lockId, err := conn.Lock(lockPath)
	if err != nil {
		t.Fatalf("")
	}
	t.Logf("Locked %s with lock id %s", lockPath, lockId)

	t.Logf("This should block")
	lockId2, err := conn.Lock(lockPath)
	if err != nil {
		t.Fatalf("")
	}
	t.Logf("Locked %s with lock id2 %s", lockPath, lockId2)

}
