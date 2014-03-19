package etcd

import (
	"testing"
	"time"
)

func TestEtcdDriver(t *testing.T) {

	tc, err := NewTestCluster()
	if err != nil {
		t.Fatalf("Could not create a test etcd cluster: %s", err)
	}
	defer tc.Stop()

	drv, err := NewEtcdDriver(tc.Machines(), time.Second)
	if err != nil {
		t.Fatalf("Could not create a client: %s", err)
	}

	exists, err := drv.Exists("/foo")
	if err != nil {
		t.Fatalf("err calling exists: %s", err)
	}
	if exists {
		t.Fatal("foo should not exist")
	}

	err = drv.Delete("/foo")
	if err == nil {
		t.Fatalf("delete on non-existent object should fail")
	}

	err = drv.CreateDir("/foo")
	if err != nil {
		t.Fatalf("creating /foo should work: %s", err)
	}

	err = drv.Create("/foo/bar", []byte("test"))
	if err != nil {
		t.Fatalf("creating /foo/bar should work: %s", err)
	}

	exists, err = drv.Exists("/foo/bar")
	if err != nil {
		t.Fatalf("could not call exists: %s", err)
	}

	if !exists {
		t.Fatal("/foo/bar should not exist")
	}

	err = drv.Delete("/foo")
	if err != nil {
		t.Fatalf("delete of /foo should work: %s", err)
	}
}
