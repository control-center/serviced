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

package zookeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
)

func TestQueue(t *testing.T) {
	/* start the cluster */
	tc, err := zklib.StartTestCluster(1, nil, os.Stderr)
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
	conn, err := drv.GetConnection(dsn, "/basePath")
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	// create a queue
	t.Log("create a queue")
	if err := conn.CreateDir("/like/a/queue"); err != nil {
		t.Fatal("unexpected error creating queue")
	}
	q := conn.NewQueue("/like/a/queue")

	// verify the queue has no data
	t.Log("checking queue data")
	if err := q.Current(&testNodeT{}); err != ErrEmptyQueue {
		t.Errorf("expected err %s; got err %s", ErrEmptyQueue, err)
	}

	if err := q.Next(&testNodeT{}); err != ErrEmptyQueue {
		t.Errorf("expected err %s; got err %s", ErrEmptyQueue, err)
	}

	if q.HasLock() {
		t.Errorf("unexpected lock found on queue!")
	}

	if err := q.Consume(); err != ErrNotLocked {
		t.Errorf("expected err %s; got %s", ErrNotLocked, err)
	}

	// Add the first node
	qNode0 := &testNodeT{
		Name: "Item0",
	}

	t.Logf("Adding node %+v", qNode0)
	if _, err := q.Put(qNode0); err != nil {
		t.Errorf("unexpected error enqueueing node %+v: %s", qNode0, err)
	}

	// verify the node is not in flight
	var actual testNodeT
	if err := q.Current(&actual); err != ErrNotLocked {
		t.Errorf("expected err %s; got err %s", ErrNotLocked, err)
	}

	// verify the node is next to be enqueued
	actual = testNodeT{}
	if err := q.Next(&actual); err != nil {
		t.Errorf("unexpected error getting the next in-flight node in the queue: %s", err)
	} else if actual.Name != qNode0.Name {
		t.Errorf("expected node %+v; got node %+v", qNode0, actual)
	}

	// get the next node in the queue
	t.Logf("Getting node off queue")
	actual = testNodeT{}
	if err := q.Get(&actual); err != nil {
		t.Errorf("unexpected error getting the node from the queue: %s", err)
	} else if actual.Name != qNode0.Name {
		t.Errorf("expected node %+v; got node %+v", qNode0, actual)
	}

	// verify you cannot hold more than one node in-flight on the same instance
	actual = testNodeT{}
	if err := q.Get(&actual); err != ErrDeadlock {
		t.Fatalf("expected err %+v; got err %+v", ErrDeadlock, err)
	}
	if !q.HasLock() {
		t.Fatalf("missing lock!")
	}

	// verify the node is in flight
	actual = testNodeT{}
	if err := q.Current(&actual); err != nil {
		t.Errorf("unexpected error getting the in-flight node in the queue: %s", err)
	} else if actual.Name != qNode0.Name {
		t.Errorf("expected node %+v; got node %+v", qNode0, actual)
	}

	// verify the node is not enqueued
	actual = testNodeT{}
	if err := q.Next(&actual); err != ErrEmptyQueue {
		t.Errorf("expected err %s; got err %s", ErrEmptyQueue, err)
	}

	// Add a second node
	qNode1 := &testNodeT{
		Name: "Item1",
	}

	t.Logf("Adding node %+v", qNode1)
	p1, err := q.Put(qNode1)
	if err != nil {
		t.Errorf("unexpected error enqueueing node %+v: %s", qNode1, err)
	}

	// verify the node is not in flight
	actual = testNodeT{}
	if err := q.Current(&actual); err != nil {
		t.Errorf("unexpected error getting the current in-flight node in the queue: %s", err)
	} else if actual.Name != qNode0.Name {
		t.Errorf("expected node %+v; got node %+v", qNode0, actual)
	}

	// verify the node is enqueued
	actual = testNodeT{}
	if err := q.Next(&actual); err != nil {
		t.Errorf("unexpected error getting the next in-flight node in the queue: %s", err)
	} else if actual.Name != qNode1.Name {
		t.Errorf("expected node %+v; got node %+v", qNode1, actual)
	}

	// Add a third node
	qNode2 := &testNodeT{
		Name: "Item2",
	}

	t.Logf("Adding node %+v", qNode2)
	if _, err := q.Put(qNode2); err != nil {
		t.Errorf("unexpected error enqueueing node %+v: %s", qNode2, err)
	}

	// verify the node is not in flight
	actual = testNodeT{}
	if err := q.Current(&actual); err != nil {
		t.Errorf("unexpected error getting the current in-flight node in the queue: %s", err)
	} else if actual.Name != qNode0.Name {
		t.Errorf("expected node %+v; got node %+v", qNode0, actual)
	}

	// verify the node is not enqueued
	actual = testNodeT{}
	if err := q.Next(&actual); err != nil {
		t.Errorf("unexpected error getting the next in-flight node in the queue: %s", err)
	} else if actual.Name != qNode1.Name {
		t.Errorf("expected node %+v; got node %+v", qNode1, actual)
	}

	// Remove the 2nd node
	if err := conn.Delete(path.Join("/like/a/queue", path.Base(p1))); err != nil {
		t.Errorf("unexpected error dequeueing node %+v: %s", qNode1, err)
	}

	// verify the node is now enqueued
	actual = testNodeT{}
	if err := q.Next(&actual); err != nil {
		t.Errorf("unexpected error getting the next in-flight node in the queue: %s", err)
	} else if actual.Name != qNode2.Name {
		t.Errorf("expected node %+v; got node %+v", qNode2, actual)
	}

	// initialize another queue on the same path
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		q := conn.NewQueue("/like/a/queue")

		// verify that the lock is not owned
		if q.HasLock() {
			t.Fatalf("queue unexpectedly has lock!")
		}

		// try to consume the current in-flight node
		if err := q.Consume(); err != ErrNotLocked {
			t.Fatalf("expected err %s; got %s", ErrNotLocked, err)
		}

		var actual testNodeT
		if err := q.Get(&actual); err != nil {
			t.Errorf("unexpected error getting the node from the queue: %s", err)
		} else if actual.Name != qNode2.Name {
			t.Errorf("expected node %+v; got node %+v")
		}

		defer q.Consume()

		// verify the correct node is in-flight
		actual = testNodeT{}
		if err := q.Current(&actual); err != nil {
			t.Errorf("unexpected error getting the current in-flight node in the queue: %s", err)
		} else if actual.Name != qNode2.Name {
			t.Errorf("expected node %+v; got node %+v", qNode2, actual)
		}

		// verify that no node is next in queue
		actual = testNodeT{}
		if err := q.Next(&actual); err != ErrEmptyQueue {
			t.Errorf("expected error %s; got %s", ErrEmptyQueue, err)
		}
	}()

	// consume the current in-flight node
	if err := q.Consume(); err != nil {
		t.Fatalf("unexpected error consuming node %+v: %s", qNode0, err)
	}

	if q.HasLock() {
		t.Fatalf("queue unexpectedly has lock!")
	}

	if err := q.Consume(); err != ErrNotLocked {
		t.Errorf("expected err %s; got err %s", ErrEmptyQueue, err)
	}

	wg.Wait()
	t.Logf("Thread done")

	// verify no nodes are in-flight
	actual = testNodeT{}
	if err := q.Current(&actual); err != ErrEmptyQueue {
		t.Errorf("expected err %s; got err %s", ErrEmptyQueue, err)
	}

	// verify no nodes are enqueued
	actual = testNodeT{}
	if err := q.Current(&actual); err != ErrEmptyQueue {
		t.Errorf("expected err %s; got err %s", ErrEmptyQueue, err)
	}
}
