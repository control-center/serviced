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

package storage

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/dfs/nfs"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/test"
	"github.com/zenoss/glog"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

type mockNfsDriverT struct {
	clients    []string
	syncCalled bool
	exportName string
	exportPath string
}

func (m *mockNfsDriverT) ExportPath() string {
	return path.Join(m.exportPath, m.exportName)
}

func (m *mockNfsDriverT) SetClients(client ...string) {
	m.clients = client
}
func (m *mockNfsDriverT) Sync() error {
	m.syncCalled = true
	return nil
}

func (m *mockNfsDriverT) Restart() error {
	return nil
}

func (m *mockNfsDriverT) Stop() error {
	return nil
}

func (m *mockNfsDriverT) AddVolume(path string) error {
	return nil
}

func (m *mockNfsDriverT) RemoveVolume(path string) error {
	return nil
}

func TestServer(t *testing.T) {
	t.Skip() // the zookeeper part doesnt work in this test, but does work in real life
	zzkServer := &zzktest.ZZKServer{}
	if err := zzkServer.Start(); err != nil {
		t.Fatalf("Could not start zookeeper: %s", err)
	}
	defer zzkServer.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", zzkServer.Port)}

	dsnBytes, err := json.Marshal(zookeeper.DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatalf("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	basePath := ""
	zClient, err := client.New("zookeeper", dsn, basePath, nil)
	if err != nil {
		t.Fatal("unexpected error getting zk client")
	}
	zzk.InitializeLocalClient(zClient)

	defer func(orig func(nfs.Driver, string, string) error) {
		nfsMount = orig
	}(nfsMount)

	var local, remote string
	nfsMount = func(driver nfs.Driver, a, b string) error {
		glog.Infof("client is mounting %s to %s", a, b)
		remote = a
		local = b
		return nil
	}

	// creating a UUID in order to make a unique poolID
	// the poolID is somehow being saved on the filesystem (zookeeper config somewhere?)
	// making the poolID unique on every run will ensure it is stateless
	uuid, err := utils.NewUUID()
	if err != nil {
		t.Fatal("New UUID could not be created")
	}

	hostServer := host.New()
	hostServer.ID = "nodeID"
	hostServer.IPAddr = "192.168.1.50"
	hostServer.PoolID = uuid

	hostClient1 := host.New()
	hostClient1.ID = "nodeID_client1"
	hostClient1.IPAddr = "192.168.1.100"
	hostClient1.PoolID = uuid

	mockNfsDriver := &mockNfsDriverT{
		exportPath: "/exports",
		exportName: "serviced_var",
	}

	// TODO: this gets stuck at server.go:90 call to conn.CreateDir hangs
	s, err := NewServer(mockNfsDriver, hostServer, path.Join(mockNfsDriver.exportPath, mockNfsDriver.exportName))
	if err != nil {
		t.Fatalf("unexpected error creating Server: %s", err)
	}

	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		t.Fatalf("unexpected error getting connection: %s", err)
	}
	shutdown := make(chan interface{})
	defer close(shutdown)
	go s.Run(shutdown, conn)

	// give it some time
	time.Sleep(time.Second * 5)

	if !mockNfsDriver.syncCalled {
		t.Fatalf("sync() should have been called by now")
	}
	if len(mockNfsDriver.clients) != 0 {
		t.Fatalf("Expected number of clients: 0 --- Found: %v (%v)", len(mockNfsDriver.clients), mockNfsDriver.clients)
	}
	mockNfsDriver.syncCalled = false
	tmpVar, err := ioutil.TempDir("", "serviced_var")
	if err != nil {
		t.Fatalf("could not create tempdir: %s", err)
	}
	defer os.RemoveAll(tmpVar)
	c1, err := NewClient(hostClient1, tmpVar)
	if err != nil {
		t.Fatalf("could not create client: %s", err)
	}
	// give it some time
	time.Sleep(time.Second * 2)
	if !mockNfsDriver.syncCalled {
		t.Fatalf("sync() should have been called by now")
	}

	if len(mockNfsDriver.clients) != 1 {
		t.Fatalf("expecting 1 client, got %d", len(mockNfsDriver.clients))
	}
	if mockNfsDriver.clients[0] != hostClient1.IPAddr {
		t.Fatalf("expecting '%s', got '%s'", hostServer.IPAddr, mockNfsDriver.clients[0])
	}

	shareName := fmt.Sprintf("%s:%s", hostServer.IPAddr, mockNfsDriver.ExportPath())
	if remote != shareName {
		t.Fatalf("remote should be %s, not %s", remote, shareName)
	}

	glog.Info("about to call c1.Close()")
	c1.Close()
}

func assertNoError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatalf(msg+": %s", err)
	}
}
