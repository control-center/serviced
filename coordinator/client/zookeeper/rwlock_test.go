// Copyright 2016 The Serviced Authors.
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
	"sync"
	"testing"
	"time"

	"github.com/control-center/serviced/zzk/test"
)

//	"github.com/control-center/serviced/zzk/test"

const (
	rwlocktestIsolationRoot = "/rwlocktest"
)

var (
	zzkServer *zzktest.ZZKServer
)

// Test setup
func startZookeeper(basePath string, t *testing.T) *Connection {
	// start the cluster
	zzkServer = &zzktest.ZZKServer{}
	if err := zzkServer.Start(); err != nil {
		t.Fatalf("Could not start zookeeper: %s", err)
	}
	cleanUp := true
	defer func() {
		if cleanUp {
			zzkServer.Stop()
		}
	}()

	time.Sleep(time.Second)

	// set up the driver
	servers := []string{fmt.Sprintf("127.0.0.1:%d", zzkServer.Port)}
	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	// create a connection
	clientConn, err := drv.GetConnection(dsn, basePath)
	if err != nil {
		t.Fatalf("unexpected error getting connection: %s", err)
	}

	// Special for the RWLock tests:
	// Convert internface client.Connection to implementation zookeeper.Connection
	conn, ok := clientConn.(*Connection)
	if !ok {
		t.Fatalf("wrong connection type returned by driver")
	}

	cleanUp = false
	return conn
}

func obtainLock(l *RWLock, isWrite bool) error {
	if isWrite {
		return l.Lock()
	} else {
		return l.RLock()
	}
}

func verifyBlock(conn *Connection, t *testing.T, path1 string, isWrite1 bool, path2 string, isWrite2 bool) {
	var err error

	// create a lock and write-lock it
	lock1 := conn.NewRWLock(path1)
	if err = obtainLock(lock1, isWrite1); err != nil {
		t.Fatalf("unexpected error acquiring lock 1: %s", err)
	}

	// create a second lock and test that a write-locking attempt blocks
	lock2 := conn.NewRWLock(path2)
	lock2Response := make(chan error)
	go func() {
		lock2Response <- obtainLock(lock2, isWrite2)
	}()
	select {
	case response := <-lock2Response:
		if response != nil {
			t.Errorf("unexpected error locking lock 2: %s", response)
		} else {
			t.Errorf("lock 2 did not block as expected")
		}
	case <-time.After(time.Second):
		t.Log("good, lock 2 is blocking")
	}

	// free the first lock, and test if the second lock unblocks
	if err = lock1.Unlock(); err != nil {
		t.Fatalf("unexpected error releasing lock 1: %s", err)
	}
	select {
	case response := <-lock2Response:
		if response != nil {
			t.Errorf("unexpected error locking lock 2: %s", response)
		}
	case <-time.After(time.Second * 3):
		t.Errorf("timeout waiting for lock 2 to unblock")
	}

	// check if the second lock cleans up
	if err = lock2.Unlock(); err != nil {
		t.Errorf("unexpected error releasing lock 2: %s", err)
	}
}

func verifyNoBlock(conn *Connection, t *testing.T, path1 string, lockType1 bool, path2 string, lockType2 bool) {
	var err error

	// create a lock and read-lock it
	lock1 := conn.NewRWLock(path1)
	if err = obtainLock(lock1, false); err != nil {
		t.Errorf("unexpected error acquiring lock 1: %s", err)
	}

	// create a second lock and test that a write-locking attempt blocks
	lock2 := conn.NewRWLock(path2)
	lock2Response := make(chan error)
	go func() {
		lock2Response <- obtainLock(lock2, false)
	}()
	select {
	case response := <-lock2Response:
		if response != nil {
			t.Errorf("unexpected error locking lock 2: %s", response)
		} else {
			t.Log("good, lock 2 did not block")
		}
	case <-time.After(time.Second * 3):
		t.Errorf("lock 2 unexpectedly blocked")
	}

	// check that both locks clean up
	if err = lock1.Unlock(); err != nil {
		t.Errorf("unexpected error releasing lock 1: %s", err)
	}
	if err = lock2.Unlock(); err != nil {
		t.Errorf("unexpected error releasing lock 2: %s", err)
	}
}

// On the same object, two write locks, second SHOULD block
func TestRWLock_SameWriteWrite(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyBlock(conn, t, "/foo/bar", true, "/foo/bar", true)
}

// On the same object, two write locks, second should NOT block
func TestRWLock_SameReadRead(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyNoBlock(conn, t, "/foo/bar", false, "/foo/bar", false)
}

// On the same object, write lock then read lock, second SHOULD block
func TestRWLock_SameWriteRead(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyBlock(conn, t, "/foo/bar", true, "/foo/bar", false)
}

// On the same object, read lock then write lock, second SHOULD block
func TestRWLock_SameReadWrite(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyBlock(conn, t, "/foo/bar", false, "/foo/bar", true)
}

// Write lock a parent, write lock a child, second should NOT block
func TestRWLock_ParentWriteChildWrite(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyNoBlock(conn, t, "/foo", true, "/foo/bar", true)
}

// Write lock a parent, read lock a child, second should NOT block
func TestRWLock_ParentWriteChildRead(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyNoBlock(conn, t, "/foo", true, "/foo/bar", false)
}

// Read lock a parent, write lock a child, second should NOT block
func TestRWLock_ParentReadChildWrite(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyNoBlock(conn, t, "/foo", false, "/foo/bar", true)
}

// Read lock a parent, read lock a child, second should NOT block
func TestRWLock_ParentReadChildRead(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyNoBlock(conn, t, "/foo", false, "/foo/bar", false)
}

// Write lock a child, write lock a parent, second should NOT block
func TestRWLock_ChildWriteParentWrite(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyNoBlock(conn, t, "/foo/bar", true, "/foo", true)
}

// Write lock a child, read lock a parent, second should NOT block
func TestRWLock_ChildWriteParentRead(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyNoBlock(conn, t, "/foo/bar", true, "/foo", false)
}

// Read lock a child, write lock a parent, second should NOT block
func TestRWLock_ChildReadParentWrite(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyNoBlock(conn, t, "/foo/bar", false, "/foo", true)
}

// Read lock a child, read lock a parent, second should NOT block
func TestRWLock_ChildReadParentRead(t *testing.T) {
	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	verifyNoBlock(conn, t, "/foo/bar", false, "/foo", false)
}

// Can lock the root node
func TestRWLock_LockRoot(t *testing.T) {
	// setup
	conn := startZookeeper("/", t)
	defer zzkServer.Stop()

	verifyBlock(conn, t, "/", true, "/", false)
}

// Make sure multiple read locks are all released at once
func TestRWLock_MultipleReadLocks(t *testing.T) {
	numGoroutines := 4
	objectToLock := "/foo/multiread"

	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	// get a lock so goroutines block
	mainlock := conn.NewRWLock(objectToLock)
	if err := mainlock.Lock(); err != nil {
		t.Fatalf("unexpected error acquiring mainlock: %s", err)
	}

	// spin off several goroutines
	ready := make(chan int)
	unlock := make(chan struct{})
	var wg sync.WaitGroup
	for i := 1; i <= numGoroutines; i++ {
		wg.Add(1)
		go func(myId int) {
			defer wg.Done() // Signal that I am done
			// Get read lock
			myLock := conn.NewRWLock(objectToLock)
			if err := myLock.RLock(); err != nil {
				t.Fatalf("unexpected error acquiring lock %d: %s", myId, err)
			}
			// Signal that I am unblocked
			ready <- myId
			// Wait for unlock signal
			select {
			case <-unlock:
			}
			// Clean up
			if err := myLock.Unlock(); err != nil {
				t.Errorf("unexpected error from unlock %d: %s", myId, err)
			}
		}(i)
	}

	// make sure all are blocked
	numEvents := 0
	timeout := time.NewTimer(2 * time.Second)
	select {
	case <-ready:
		t.Fatalf("oee or more goroutines did not block")
	case <-timeout.C:
		t.Logf("good, all subroutines blocked")
	}

	// unblock the goroutines
	if err := mainlock.Unlock(); err != nil {
		t.Fatalf("unexpected error from unlock main: %s", err)
	}

	// wait for the subroutines to all signal they are unblocked
	numEvents = 0
	timeout = time.NewTimer(time.Second)
	for numEvents < numGoroutines {
		select {
		case ev := <-ready:
			t.Logf("goroutine %d is ready", ev)
			numEvents++
		case <-timeout.C:
			t.Fatalf("timed out waiting for goroutines to complete")
		}
	}

	// tell subroutines to unlock
	close(unlock)
	wg.Wait()
}

// Make sure multiple write locks are released one at a time
func TestRWLock_MultipleWriteLocks(t *testing.T) {
	numGoroutines := 4
	objectToLock := "/foo/multiwrite"

	// setup
	conn := startZookeeper(rwlocktestIsolationRoot, t)
	defer zzkServer.Stop()

	// data to be managed by the lock
	var listA []int
	var listB []int
	var listC []int

	// get a lock so goroutines block
	mainlock := conn.NewRWLock(objectToLock)
	if err := mainlock.Lock(); err != nil {
		t.Fatalf("unexpected error acquiring main lock: %s", err)
	}

	// spin off several goroutines
	ready := make(chan int)
	event := make(chan int)
	for i := 1; i <= numGoroutines; i++ {
		go func(myId int) {
			defer func() { event <- myId }() // Signal that I am done
			// Get write lock
			ready <- myId
			myLock := conn.NewRWLock(objectToLock)
			if err := myLock.Lock(); err != nil {
				t.Fatalf("unexpected error acquiring lock %d: %s", myId, err)
			}
			// Update the data
			time.Sleep(250 * time.Millisecond)
			listA = append(listA, myId)
			time.Sleep(250 * time.Millisecond)
			listB = append(listB, myId)
			time.Sleep(250 * time.Millisecond)
			listC = append(listC, myId)
			// Unlock
			if err := myLock.Unlock(); err != nil {
				t.Errorf("unexpected error from unlock %d: %s", myId, err)
			}
		}(i)
	}

	// wait until all goroutines are spun off and obtaining their locks
	numEvents := 0
	timeout := time.NewTimer(2 * time.Second)
	for numEvents < numGoroutines {
		select {
		case <-ready:
			numEvents++
		case <-timeout.C:
			t.Fatalf("goroutines never got spun off")
		}
	}
	time.Sleep(250 * time.Microsecond) // event is sent before Lock() is called

	// unblock the goroutines
	if err := mainlock.Unlock(); err != nil {
		t.Fatalf("unexpected error from unlock main: %s", err)
	}

	// wait for the goroutines to finish (they should to one at at time as each unlocks)
	numEvents = 0
	timeout = time.NewTimer((time.Duration(numGoroutines) * time.Second) + (2 * time.Second))
	for numEvents < numGoroutines {
		select {
		case ev := <-event:
			t.Logf("goroutine %d is done", ev)
			numEvents++
		case <-timeout.C:
			t.Fatalf("timed out waiting for goroutines to update the data")
		}
	}

	// verify all the data lists are the same
	t.Logf("data lists: A=%v, B=%v, C=%v", listA, listB, listC)
	for i := 0; i < len(listA); i++ {
		if listA[i] != listB[i] || listB[i] != listC[i] {
			t.Fatalf("lists do not match at element %d", i)
		}
	}
}
