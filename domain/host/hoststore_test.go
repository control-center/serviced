// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"

	"testing"
	"time"
)

var hs HostStore
var ctx datastore.Context

func init() {
	esDriver := elastic.New("localhost", 9200, "controlplane")
	esDriver.AddMappingFilie("host", "./host_mapping.json")
	err := esDriver.Initialize()
	if err != nil {
		glog.Infof("Error initializing db driver %v", err)
		return
	}
	hs = NewStore()
	ctx = datastore.NewContext(esDriver)
}

func Test_HostCRUD(t *testing.T) {

	if hs == nil {
		t.Fatalf("Test failed to initialize")
	}
	defer hs.Delete(ctx, "testid")

	host := New()

	err := hs.Put(ctx, host)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
	}

	host.Id = "testid"
	err = hs.Put(ctx, host)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
	}

	//fill host with required values
	host, err = Build("", "pool-id", []string{}...)
	host.Id = "testid"
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	err = hs.Put(ctx, host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	host2, err := hs.Get(ctx, "testid")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	hostEquals(host, host2, t)

	//Test update
	host.Memory = 1024
	err = hs.Put(ctx, host)
	host2, err = hs.Get(ctx, "testid")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	hostEquals(host, host2, t)

	//test delete
	err = hs.Delete(ctx, "testid")
	host2, err = hs.Get(ctx, "testid")
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}

}

func Test_GetHosts(t *testing.T) {
	if hs == nil {
		t.Fatalf("Test failed to initialize")
	}
	defer hs.Delete(ctx, "Test_GetHosts1")
	defer hs.Delete(ctx, "Test_GetHosts2")

	host, err := Build("", "pool-id", []string{}...)
	host.Id = "Test_GetHosts1"
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	err = hs.Put(ctx, host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	time.Sleep(1000 * time.Millisecond)
	hosts, err := hs.GetUpTo(ctx, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 1 {
		t.Errorf("Expected %v results, got %v :%v", 1, len(hosts), hosts)
	}

	host.Id = "Test_GetHosts2"
	err = hs.Put(ctx, host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	time.Sleep(1000 * time.Millisecond)
	hosts, err = hs.GetUpTo(ctx, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 2 {
		t.Errorf("Expected %v results, got %v :%v", 2, len(hosts), hosts)
	}

}

func hostEquals(h1 *Host, h2 *Host, t *testing.T) {
	if h1.Id != h2.Id {
		t.Errorf("Host name %v did not equal %v", h1.Id, h2.Id)
	}
	if h1.Name != h2.Name {
		t.Errorf("Host id %v did not equal %v", h1.Name, h2.Name)
	}
	if h1.PoolId != h2.PoolId {
		t.Errorf("Host PoolId %v did not equal %v", h1.PoolId, h2.PoolId)
	}
	if h1.IpAddr != h2.IpAddr {
		t.Errorf("Host IpAddr %v did not equal %v", h1.IpAddr, h2.IpAddr)
	}

	if h1.Cores != h2.Cores {
		t.Errorf("Host Cores %v did not equal %v", h1.Cores, h2.Cores)
	}
	if h1.Memory != h2.Memory {
		t.Errorf("Host Memory %v did not equal %v", h1.Memory, h2.Memory)
	}
	if h1.PrivateNetwork != h2.PrivateNetwork {
		t.Errorf("Host PrivateNetwork %v did not equal %v", h1.PrivateNetwork, h2.PrivateNetwork)
	}
}
