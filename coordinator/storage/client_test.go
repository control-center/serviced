package storage

import (
	zklib "github.com/samuel/go-zookeeper/zk"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/zzk"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	zookeeper.EnsureZkFatjar()
	basePath := ""
	tc, err := zklib.StartTestCluster(1)
	if err != nil {
		t.Fatalf("could not start test zk cluster: %s", err)
	}
	defer os.RemoveAll(tc.Path)
	defer tc.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", tc.Servers[0].Port)}

	dsnBytes, err := json.Marshal(zookeeper.DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatal("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)
	zClient, err := client.New("zookeeper", dsn, basePath, nil)

	zzk.InitializeGlobalCoordClient(zClient)

	conn, err := zzk.GetBasePathConnection("/")
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	h := host.New()
	h.ID = "nodeID"
	h.IPAddr = "192.168.1.5"
	defer func(old func(string, os.FileMode) error) {
		mkdirAll = old
	}(mkdirAll)
	dir, err := ioutil.TempDir("", "serviced_var_")
	if err != nil {
		t.Fatalf("could not create tempdir: %s", err)
	}
	defer os.RemoveAll(dir)
	c, err := NewClient(h, dir)
	if err != nil {
		t.Fatalf("unexpected error creating client: %s", err)
	}
	defer c.Close()
	time.Sleep(time.Second * 10)

	nodePath := fmt.Sprintf("/storage/clients/%s", h.IPAddr)
	glog.Infof("about to check for %s", nodePath)
	if exists, err := conn.Exists(nodePath); err != nil {
		t.Fatalf("did not expect error checking for existence of %s: %s", nodePath, err)
	} else {
		if !exists {
			t.Fatalf("could not find %s", nodePath)
		}
	}
}
