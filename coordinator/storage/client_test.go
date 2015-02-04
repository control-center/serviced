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

package storage

import (
	zklib "github.com/control-center/go-zookeeper/zk"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	t.Skipf("Test cluster is not set up properly")
	zookeeper.EnsureZkFatjar()
	basePath := ""
	tc, err := zklib.StartTestCluster(1, nil, nil)
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

	zzk.InitializeLocalClient(zClient)

	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

	h := host.New()
	h.ID = "nodeID"
	h.IPAddr = "192.168.1.5"
	h.PoolID = "default1"
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
	time.Sleep(time.Second * 5)

	// therefore, we need to check that the client was added under the pool from root
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
