package storage

import (
	zklib "github.com/samuel/go-zookeeper/zk"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/domain/host"

	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

type mockNfsServerT struct {
	clients    []string
	syncCalled bool
	exportPath string
}

func (m *mockNfsServerT) ExportPath() string {
	return m.exportPath
}

func (m *mockNfsServerT) SetClients(client ...string) {
	m.clients = client
}
func (m *mockNfsServerT) Sync() error {
	m.syncCalled = true
	return nil
}

func TestServer(t *testing.T) {
	zookeeper.EnsureZkFatjar()
	basePath := "/basePath"
	tc, err := zklib.StartTestCluster(1)
	if err != nil {
		t.Fatalf("could not start test zk cluster: %s", err)
	}
	defer os.RemoveAll(tc.Path)
	defer tc.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", tc.Servers[0].Port)}

	drv := zookeeper.Driver{}
	dsnBytes, err := json.Marshal(zookeeper.DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatal("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	conn, err := drv.GetConnection(dsn, basePath)
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}

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

	h := host.New()
	h.ID = "nodeID"
	h.IPAddr = "192.168.1.5"

	hc1 := host.New()
	hc1.ID = "nodeID_client1"
	hc1.IPAddr = "192.168.1.10"

	mockNfsServer := &mockNfsServerT{
		exportPath: fmt.Sprintf("%s:/serviced_var", h.IPAddr),
	}

	s, err := NewServer(mockNfsServer, h, conn)
	if err != nil {
		t.Fatalf("unexpected error creating Server: %s", err)
	}
	defer s.Close()

	// give it some time
	time.Sleep(time.Second * 5)

	if !mockNfsServer.syncCalled {
		t.Fatalf("sync() should have been called by now")
	}
	if len(mockNfsServer.clients) != 0 {
		t.Fatalf("there should be no clients yet")
	}
	mockNfsServer.syncCalled = false
	c1 := NewClient(hc1, conn)
	// give it some time
	time.Sleep(time.Second * 2)
	if !mockNfsServer.syncCalled {
		t.Fatalf("sync() should have been called by now")
	}

	if len(mockNfsServer.clients) != 1 {
		t.Fatalf("expecting 1 client, got %d", len(mockNfsServer.clients))
	}
	if mockNfsServer.clients[0] != hc1.IPAddr {
		t.Fatalf("expecting '%s', got '%s'", h.IPAddr, mockNfsServer.clients[0])
	}

	if remote != mockNfsServer.exportPath {
		t.Fatalf("remote should be %s", remote)
	}

	c1.Close()
}

func assertNoError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatalf(msg+": %s", err)
	}
}
