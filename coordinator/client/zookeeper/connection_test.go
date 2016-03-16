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
	"path"
	"testing"
	"time"

	coordclient "github.com/control-center/serviced/coordinator/client"
	zzktest "github.com/control-center/serviced/zzk/test"
	"github.com/zenoss/glog"
)

type testNodeT struct {
	Name    string
	version interface{}
}

func (n *testNodeT) SetVersion(version interface{}) {
	glog.Infof("seting version to: %v", version)
	n.version = version
}
func (n *testNodeT) Version() interface{} { return n.version }

func TestZkDriver(t *testing.T) {
	zzkServer := &zzktest.ZZKServer{}
	if err := zzkServer.Start(); err != nil {
		t.Fatalf("Could not start zookeeper: %s", err)
	}
	defer zzkServer.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", zzkServer.Port)}

	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	basePath := "/basePath"
	conn, err := drv.GetConnection(dsn, basePath)
	if err != nil {
		t.Fatal("unexpected error getting connection")
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

	testNode := &testNodeT{
		Name: "test",
	}
	err = conn.Create("/foo/bar", testNode)
	if err != nil {
		t.Fatalf("creating /foo/bar should work: %s", err)
	}
	t.Logf("testNode version: %v", testNode.Version())

	exists, err = conn.Exists("/foo/bar")
	if err != nil {
		t.Fatalf("could not call exists: %s", err)
	}

	if !exists {
		t.Fatal("/foo/bar should  exist")
	}

	testNode2 := &testNodeT{
		Name: "baz",
	}
	err = conn.Get("/foo/bar", testNode2)
	if err != nil {
		t.Fatalf("could not get /foo/bar node: %s", err)
	}

	if testNode.Name != testNode2.Name {
		t.Fatalf("expected testNodes to match %s  --- %s", testNode.Name, testNode2.Name)
	}

	err = conn.Get("/foo/bar", testNode2)
	testNode2.Name = "abc"
	if err := conn.Set("/foo/bar", testNode2); err != nil {
		t.Fatalf("Could not update testNode: %s", err)
	}

	err = conn.Delete("/foo")
	if err != nil {
		t.Fatalf("delete of /foo should work: %s", err)
	}

	err = conn.CreateDir("/fum/bar/baz/echo/p/q")
	if err != nil {
		t.Fatalf("creating /fum/bar/baz/echo/p/q should work")
	}
	exists, err = conn.Exists("/fum/bar/baz/echo/p/q")
	if err != nil {
		t.Fatalf("could not call exists: %s", err)
	}
	if !exists {
		t.Fatal("/fum/bar/baz/echo/p/q should exist")
	}
	err = conn.Delete("/fum")
	if err != nil {
		t.Fatalf("delete of /fum should work: %s", err)
	}

	conn.Close()
}

func TestZkDriver_Multi(t *testing.T) {
	zzkServer := &zzktest.ZZKServer{}
	if err := zzkServer.Start(); err != nil {
		t.Fatalf("Could not start zookeeper: %s", err)
	}
	defer zzkServer.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", zzkServer.Port)}

	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}

	dsn := string(dsnBytes)

	basePath := "/basePath"
	conn, err := drv.GetConnection(dsn, basePath)
	defer conn.Close()
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	conn.CreateDir("/basePath")

	//
	// Test creating a new node and setting a non-existent node. Should not commit.
	//
	testNode0 := &testNodeT{
		Name: "test0",
	}
	testNode1 := &testNodeT{
		Name: "test1",
	}

	multi := conn.NewTransaction()
	multi.Create("/test0", testNode0)
	multi.Set("/test1", testNode1)
	if err = multi.Commit(); err == nil {
		t.Fatalf("creating /test0 and setting /test1 should have failed")
	}

	exists, err := conn.Exists("/test0")
	if err != nil {
		t.Fatalf("Error testing for existence of /test0: %s", err)
	}
	if exists {
		t.Fatalf("/test0 should not have been created")
	}

	//
	// Test creating two new nodes. Should commit.
	//
	multi = conn.NewTransaction()
	multi.Create("/test0", testNode0)
	multi.Create("/test1", testNode1)
	if err = multi.Commit(); err != nil {
		t.Fatalf("creating /test0 and /test1 should work: %s", err)
	}

	out := &testNodeT{
		Name: "luffydmonkey",
	}

	err = conn.Get("/test0", out)
	if err != nil {
		t.Fatalf("getting /test0 should work: %s", err)
	}

	if out.Name != "test0" {
		t.Fatalf("expected test0, got %s", out.Name)
	}

	err = conn.Get("/test1", out)
	if err != nil {
		t.Fatalf("getting /test1 should work: %s", err)
	}

	if out.Name != "test1" {
		t.Fatalf("expected test1, got %s", out.Name)
	}

	//
	// Test setting the newly created nodes. Should commit.
	//
	testNode0.Name = "test0b"
	testNode1.Name = "test1b"

	multi = conn.NewTransaction()
	multi.Set("/test0", testNode0)
	multi.Set("/test1", testNode1)
	if err = multi.Commit(); err != nil {
		t.Fatalf("setting test0 and test1 should work: %s", err)
	}

	out.Name = "luffydmonkey"

	err = conn.Get("/test0", out)
	if err != nil {
		t.Fatalf("getting /test0 should work: %s", err)
	}

	if out.Name != "test0b" {
		t.Fatalf("expected test0b, got %s", out.Name)
	}

	out.Name = "luffydmonkey"

	err = conn.Get("/test1", out)
	if err != nil {
		t.Fatalf("getting /test1 should work: %s", err)
	}

	if out.Name != "test1b" {
		t.Fatalf("expected test1b, got %s", out.Name)
	}

	//
	// Attempt to delete the same node twice in the same transaction. Should not commit.
	//
	multi = conn.NewTransaction()
	multi.Delete("/test0")
	multi.Delete("/test0")
	if err = multi.Commit(); err == nil {
		t.Fatalf("expected error trying to delete the same node twice")
	}

	exists, err = conn.Exists("/test0")
	if err != nil {
		t.Fatalf("Error testing for existence of /test0: %s", err)
	}
	if !exists {
		t.Fatalf("/test0 should not have been deleted")
	}

	//
	// Attempt to delete two nodes in a transaction. Should commit.
	//
	multi = conn.NewTransaction()
	multi.Delete("/test0")
	multi.Delete("/test1")
	if err = multi.Commit(); err != nil {
		t.Fatalf("deleting /test0 and /test1 should work: %s", err)
	}

	exists, err = conn.Exists("/test0")
	if err != nil {
		t.Fatalf("Error testing for existence of /test0: %s", err)
	}
	if exists {
		t.Fatalf("/test0 should have been deleted")
	}

	exists, err = conn.Exists("/test1")
	if err != nil {
		t.Fatalf("Error testing for existence of /test1: %s", err)
	}
	if exists {
		t.Fatalf("/test1 should have been deleted")
	}

	//
	// Attempt to create the same node twice in the same transaction. Should not commit.
	//
	multi = conn.NewTransaction()
	multi.Create("/test0", testNode0)
	multi.Create("/test0", testNode1)
	if err = multi.Commit(); err == nil {
		t.Fatalf("expected error trying to create an existing node")
	}

}

func TestZkDriver_Ephemeral(t *testing.T) {
	zzkServer := &zzktest.ZZKServer{}
	if err := zzkServer.Start(); err != nil {
		t.Fatalf("Could not start zookeeper: %s", err)
	}
	defer zzkServer.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", zzkServer.Port)}

	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}

	dsn := string(dsnBytes)

	basePath := "/basePath"
	conn, err := drv.GetConnection(dsn, basePath)
	defer conn.Close()
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	node := &testNodeT{Name: "ephemeral"}
	epath, err := conn.CreateEphemeral("/ephemeral", node)
	if err != nil {
		t.Fatalf("creating /ephemeral should work: %s", err)
	}
	// The returned is from the root, so it has to be trimmed down to the
	// relative location
	ename := "/" + path.Base(epath)

	if ok, err := conn.Exists(ename); err != nil {
		t.Fatalf("could not find path to ephemeral %s: %s", ename, err)
	} else if !ok {
		t.Fatalf("ephemeral %s not created", ename)
	}

	// Close connection and verify the node was deleted
	conn.Close()
	conn, err = drv.GetConnection(dsn, basePath)

	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	if ok, err := conn.Exists(ename); err != nil && err != coordclient.ErrNoNode {
		t.Fatalf("should be able to check path %s: %s", ename, err)
	} else if ok {
		t.Errorf("ephemeral %s should have been deleted", ename)
	}

	// Adding and deleting
	node = &testNodeT{Name: "ephemeral"}
	epath, err = conn.CreateEphemeral("/ephemeral", node)
	if err != nil {
		t.Fatalf("creating /ephemeral should work: %s", err)
	}
	ename = "/" + path.Base(epath)

	if ok, err := conn.Exists(ename); err != nil {
		t.Fatalf("could not find path to ephemeral %s: %s", ename, err)
	} else if !ok {
		t.Fatalf("ephemeral %s not created", ename)
	}
	if err := conn.Delete(ename); err != nil {
		t.Fatalf("could not delete path %s to ephemeral: %s", ename, err)
	}

	if ok, err := conn.Exists(ename); err != nil && err != coordclient.ErrNoNode {
		t.Fatalf("should be able to check path %s: %s", ename, err)
	} else if ok {
		t.Errorf("ephemeral %s should have been deleted", ename)
	}
}

func TestZkDriver_Watch(t *testing.T) {
	zzkServer := &zzktest.ZZKServer{}
	if err := zzkServer.Start(); err != nil {
		t.Fatalf("Could not start zookeeper: %s", err)
	}
	defer zzkServer.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", zzkServer.Port)}

	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	basePath := "/basePath"
	conn, err := drv.GetConnection(dsn, basePath)
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	err = conn.CreateDir("/foo")
	if err != nil {
		t.Fatalf("creating /foo should work: %s", err)
	}
	err = conn.Get("/foo", &testNodeT{})
	if err != coordclient.ErrEmptyNode {
		t.Fatalf("expected empty node, got %s", err)
	}

	childWDone1 := make(chan struct{})
	defer close(childWDone1)
	_, w1, err := conn.ChildrenW("/foo", childWDone1)
	if err != nil {
		t.Fatalf("should be able to acquire watch for /foo: %s", err)
	}

	childWDone2 := make(chan struct{})
	defer close(childWDone2)
	_, w2, err := conn.ChildrenW("/foo", childWDone2)
	if err != nil {
		t.Fatalf("should be able to acquire watch for /foo: %s", err)
	}

	go func() {
		for w1 != nil || w2 != nil {
			select {
			case e := <-w1:
				if e.Type != coordclient.EventNodeChildrenChanged {
					t.Errorf("expected %v; actual: %v (w1)", coordclient.EventNodeChildrenChanged, e.Type)
				}
				w1 = nil
			case e := <-w2:
				if e.Type != coordclient.EventNodeChildrenChanged {
					t.Errorf("expected %v; actual: %v (w2)", coordclient.EventNodeChildrenChanged, e.Type)
				}
				w2 = nil
			}
		}
	}()

	<-time.After(time.Second)
	testNode := &testNodeT{
		Name: "test",
	}
	err = conn.Create("/foo/bar", testNode)
	if err != nil {
		t.Fatalf("creating /foo/bar should work: %s", err)
	}
	t.Logf("testNode version: %v", testNode.Version())
}
