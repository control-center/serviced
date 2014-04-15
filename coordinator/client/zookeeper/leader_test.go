package zk_driver

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
)

func TestLeader(t *testing.T) {

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
		t.Fatal("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	// create a connection
	conn, err := drv.GetConnection(dsn, "/bossPath")
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	// create  a leader and TakeLead
	leader1Str := "leader1"
	leader1 := conn.NewLeader("/like/a/boss", []byte(leader1Str))
	err = leader1.TakeLead()
	if err != nil {
		t.Fatal("could not take lead! %s", err)
	}

	leader2Str := "leader2"
	leader2 := conn.NewLeader("/like/a/boss", []byte(leader2Str))
	leader2Response := make(chan error)
	go func() {
		leader2Response <- leader2.TakeLead()
	}()

	select {
	case err = <-leader2Response:
		t.Fatalf("expected leader2 to block!: %s", err)
	case <-time.After(time.Second):
	}

	// get current Leader
	currentLeader := conn.NewLeader("/like/a/boss", []byte{})
	currentLeaderBytes, err := currentLeader.Current()
	if err != nil {
		t.Fatalf("unexpected error getting current leader:%s", err)
	}

	if string(currentLeaderBytes) != leader1Str {
		t.Fatalf("expected leader %s , got %s", leader1Str, string(currentLeaderBytes))
	}

	// let the first leader go
	err = leader1.ReleaseLead()
	if err != nil {
		t.Fatal("unexpected error releasing leader1 ")
	}

	select {
	case err = <-leader2Response:
		if err != nil {
			t.Fatalf("unexpected error when leader 1 was release and waiting on leader2: %s", err)

		}
	case <-time.After(time.Second * 3):
		t.Fatalf("expected leader2 to take over but we blocked")
	}

}
