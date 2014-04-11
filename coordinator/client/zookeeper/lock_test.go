package zk_driver

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
)

func TestLock(t *testing.T) {
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
		t.Fatal("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	conn, err := drv.GetConnection(dsn)
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	lock, err := conn.NewLock("/foo/bar")
	if err != nil {
		t.Fatalf("unexpected error getting lock: %s", err)
	}

	if err = lock.Lock(); err != nil {
		t.Fatalf("unexpected error aquiring lock: %s", err)
	}
	if err = lock.Unlock(); err != nil {
		t.Fatalf("unexpected error releasing lock: %s", err)
	}

}
