// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package zookeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	coordclient "github.com/control-center/serviced/coordinator/client"
	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
)

func init() {
	EnsureZkFatjar()
}

func TestEnsureZkFatjar(t *testing.T) {
	EnsureZkFatjar()
}

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
	basePath := "/basePath"
	tc, err := zklib.StartTestCluster(1)
	if err != nil {
		t.Fatalf("could not start test zk cluster: %s", err)
	}
	defer os.RemoveAll(tc.Path)
	defer tc.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", tc.Servers[0].Port)}

	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

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
	t.Logf("testNode version: %v", testNode2.Version().(*zklib.Stat).Version)
	testNode2.Name = "abc"
	if err := conn.Set("/foo/bar", testNode2); err != nil {
		t.Fatalf("Could not update testNode: %s", err)
	}

	err = conn.Delete("/foo")
	if err != nil {
		t.Fatalf("delete of /foo should work: %s", err)
	}

	conn.Close()
}

func TestZkDriver_Ephemeral(t *testing.T) {
	basePath := "/basePath"
	tc, err := zklib.StartTestCluster(1)
	if err != nil {
		t.Fatalf("could not start test zk cluster: %s", err)
	}
	defer os.RemoveAll(tc.Path)
	defer tc.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", tc.Servers[0].Port)}

	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}

	dsn := string(dsnBytes)

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
		t.Fatalf("could not delete path %s to ephemeral: %s", err)
	}

	if ok, err := conn.Exists(ename); err != nil && err != coordclient.ErrNoNode {
		t.Fatalf("should be able to check path %s: %s", ename, err)
	} else if ok {
		t.Errorf("ephemeral %s should have been deleted", ename)
	}
}

func TestZkDriver_Watch(t *testing.T) {
	basePath := "/basePath"
	tc, err := zklib.StartTestCluster(1)
	if err != nil {
		t.Fatalf("could not start test zk cluster: %s", err)
	}
	defer os.RemoveAll(tc.Path)
	defer tc.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", tc.Servers[0].Port)}

	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	conn, err := drv.GetConnection(dsn, basePath)
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	err = conn.CreateDir("/foo")
	if err != nil {
		t.Fatalf("creating /foo should work: %s", err)
	}

	_, w1, err := conn.ChildrenW("/foo")
	if err != nil {
		t.Fatalf("should be able to acquire watch for /foo: %s", err)
	}

	_, w2, err := conn.ChildrenW("/foo")
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
