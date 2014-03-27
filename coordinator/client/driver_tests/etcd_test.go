package drivertest

import (
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/coordinator/client/etcd"
	"github.com/zenoss/serviced/coordinator/client/retry"
)

func TestEtcd(t *testing.T) {
	tc, err := etcd.NewTestCluster()
	if err != nil {
		t.Fatalf("Could not create Test Cluster: %s", err)
	}

	defer tc.Stop()

	driver, err := etcd.NewDriver(tc.Machines(), time.Second*10)
	if err != nil {
		t.Fatalf("could not create new driver")
	}
	cClient, err := client.New(driver, retry.NTimes(10, time.Millisecond*30))
	if err != nil {
		t.Fatalf("Could not create create coordinator client: %s", err)
	}
	defer cClient.Close()

	conn, err := cClient.GetConnection()
	if err != nil {
		t.Fatalf("could not create connection: %s")
	}
	conn.Close()
}
