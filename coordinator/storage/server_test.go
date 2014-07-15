// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package storage

import (
	zklib "github.com/samuel/go-zookeeper/zk"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/utils"
	"github.com/zenoss/serviced/zzk"

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

func TestServer(t *testing.T) {
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
	if err != nil {
		t.Fatal("unexpected error getting zk client")
	}
	zzk.InitializeGlobalCoordClient(zClient)

	defer func(orig func(string, string) error) {
		nfsMount = orig
	}(nfsMount)

	var local, remote string
	nfsMount = func(a, b string) error {
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

	s, err := NewServer(mockNfsDriver, hostServer, zClient)
	if err != nil {
		t.Fatalf("unexpected error creating Server: %s", err)
	}
	defer s.Close()

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
