// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"
	. "gopkg.in/check.v1"

	"testing"
	"time"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&S{
	ElasticTest: elastic.ElasticTest{
		Index:    "controlplane",
		Mappings: []elastic.Mapping{MAPPING},
	}})

type S struct {
	elastic.ElasticTest
	ctx datastore.Context
	hs  *HostStore
}

func (s *S) SetUpTest(c *C) {
	s.ElasticTest.SetUpTest(c)
	datastore.Register(s.Driver())
	s.ctx = datastore.Get()
	s.hs = NewStore()
}

func (s *S) Test_HostCRUD(t *C) {
	defer s.hs.Delete(s.ctx, HostKey("testid"))

	var host2 Host

	if err := s.hs.Get(s.ctx, HostKey("testid"), &host2); !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Expected ErrNoSuchEntity, got: %v", err)
	}

	host := New()

	err := s.hs.Put(s.ctx, HostKey("testid"), host)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
	}

	host.ID = "testid"
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
	}

	//fill host with required values
	host, err = Build("", "pool-id", []string{}...)
	host.ID = "testid"
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	err = s.hs.Put(s.ctx, HostKey("testid"), host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = s.hs.Get(s.ctx, HostKey("testid"), &host2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !HostEquals(t, host, &host2) {
		t.Error("Hosts did not match")
	}

	//Test update
	host.Memory = 1024
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	err = s.hs.Get(s.ctx, HostKey("testid"), &host2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !HostEquals(t, host, &host2) {
		t.Error("Hosts did not match")
	}

	//test delete
	err = s.hs.Delete(s.ctx, HostKey("testid"))
	err = s.hs.Get(s.ctx, HostKey("testid"), &host2)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}

}

func (s *S) TestDaoGetHostWithIPs(t *C) {
	//Add host to test scenario where host exists but no IP resource registered
	h, err := Build("", "pool-id", []string{}...)
	h.ID = "TestDaoGetHostWithIPs"
	h.IPs = []HostIPResource{
		HostIPResource{h.ID, "testip", "ifname"},
		HostIPResource{h.ID, "testip2", "ifname"},
	}
	err = s.hs.Put(s.ctx, HostKey(h.ID), h)
	defer s.hs.Delete(s.ctx, HostKey(h.ID))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	var resultHost Host
	err = s.hs.Get(s.ctx, HostKey(h.ID), &resultHost)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if len(resultHost.IPs) != 2 {
		t.Errorf("Expected %v IPs, got %v", 2, len(resultHost.IPs))
	}
	if !HostEquals(t, h, &resultHost) {
		t.Error("Hosts did not match")
	}
}

func (s *S) Test_GetHosts(t *C) {
	defer s.hs.Delete(s.ctx, HostKey("Test_GetHosts1"))
	defer s.hs.Delete(s.ctx, HostKey("Test_GetHosts2"))

	host, err := Build("", "pool-id", []string{}...)
	host.ID = "Test_GetHosts1"
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	time.Sleep(1000 * time.Millisecond)
	hosts, err := s.hs.GetN(s.ctx, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 1 {
		t.Errorf("Expected %v results, got %v :%v", 1, len(hosts), hosts)
	}

	host.ID = "Test_GetHosts2"
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	time.Sleep(1000 * time.Millisecond)
	hosts, err = s.hs.GetN(s.ctx, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 2 {
		t.Errorf("Expected %v results, got %v :%v", 2, len(hosts), hosts)
	}

}

func (s *S) Test_FindHostsInPool(t *C) {
	id1 := "Test_FindHostsInPool1"
	id2 := "Test_FindHostsInPool2"
	id3 := "Test_FindHostsInPool3"

	defer s.hs.Delete(s.ctx, HostKey(id1))
	defer s.hs.Delete(s.ctx, HostKey(id2))
	defer s.hs.Delete(s.ctx, HostKey(id3))

	host, err := Build("", "pool1", []string{}...)
	host.ID = id1
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)

	host.ID = id2
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//add one with different pool
	host.ID = id3
	host.PoolID = "pool2"
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	time.Sleep(1000 * time.Millisecond)
	hosts, err := s.hs.FindHostsWithPoolID(s.ctx, "blam")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 0 {
		t.Errorf("Expected %v results, got %v :%v", 0, len(hosts), hosts)
	}

	hosts, err = s.hs.FindHostsWithPoolID(s.ctx, "pool2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 1 {
		t.Errorf("Expected %v results, got %v :%v", 1, len(hosts), hosts)
	}

	hosts, err = s.hs.FindHostsWithPoolID(s.ctx, "pool1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 2 {
		t.Errorf("Expected %v results, got %v :%v", 2, len(hosts), hosts)
	}
}
