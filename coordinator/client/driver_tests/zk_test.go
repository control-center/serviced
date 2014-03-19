package drivertest

import (
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/coordinator/client/etcd"
)

func TestEtcd(t *testing.T) {
	tc, err := etcd.NewTestCluster()
	if err != nil {
		t.Fatalf("Could not create Test Cluster: %s", err)
	}

	defer tc.Stop()

	cClient, err := client.New(tc.Machines(), time.Second*10, "etcd", nil)
	if err != nil {
		t.Fatalf("Could not create create coordinator client: %s", err)
	}
	defer cClient.Close()

}
