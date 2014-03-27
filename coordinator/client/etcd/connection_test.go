package etcd

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEtcdDriver(t *testing.T) {

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

	drv := EtcdDriver{}

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
}
