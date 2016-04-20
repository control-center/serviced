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

package host

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	. "gopkg.in/check.v1"

	"testing"
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
	hs  Store
}

func (s *S) SetUpTest(c *C) {
	s.ElasticTest.SetUpTest(c)
	datastore.Register(s.Driver())
	s.ctx = datastore.Get()
	s.hs = NewStore()
}

func (s *S) Test_HostCRUD(t *C) {
	hostID := "deadb40f"
	defer s.hs.Delete(s.ctx, HostKey(hostID))

	var host2 Host

	if err := s.hs.Get(s.ctx, HostKey(hostID), &host2); !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Expected ErrNoSuchEntity, got: %v", err)
	}

	host := New()

	err := s.hs.Put(s.ctx, HostKey(hostID), host)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
	}

	host.ID = hostID
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
	}

	//fill host with required values
	host, err = Build("", "65535", "pool-id", "", []string{}...)
	host.ID = hostID
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	err = s.hs.Put(s.ctx, HostKey(hostID), host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = s.hs.Get(s.ctx, HostKey(hostID), &host2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !HostEquals(t, host, &host2) {
		t.Error("Hosts did not match")
	}

	//Test update
	host.Memory = 1024
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	err = s.hs.Get(s.ctx, HostKey(hostID), &host2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !HostEquals(t, host, &host2) {
		t.Error("Hosts did not match")
	}

	//test delete
	err = s.hs.Delete(s.ctx, HostKey(hostID))
	err = s.hs.Get(s.ctx, HostKey(hostID), &host2)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}

}

func (s *S) TestDaoGetHostWithIPs(t *C) {
	//Add host to test scenario where host exists but no IP resource registered
	h, err := Build("", "65535", "pool-id", "", []string{}...)
	h.ID = "deadb41f"
	h.IPs = []HostIPResource{
		HostIPResource{h.ID, "testip", "ifname", "address1"},
		HostIPResource{h.ID, "testip2", "ifname", "address2"},
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

	host, err := Build("", "65535", "pool-id", "", []string{}...)
	host.ID = "deadb51f"
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	hosts, err := s.hs.GetN(s.ctx, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 1 {
		t.Errorf("Expected %v results, got %v :%v", 1, len(hosts), hosts)
	}

	host.ID = "deadb52f"
	err = s.hs.Put(s.ctx, HostKey(host.ID), host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	hosts, err = s.hs.GetN(s.ctx, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 2 {
		t.Errorf("Expected %v results, got %v :%v", 2, len(hosts), hosts)
	}

}

func (s *S) Test_FindHostsInPool(t *C) {
	id1 := "deadb61f"
	id2 := "deadb62f"
	id3 := "deadb63f"

	defer s.hs.Delete(s.ctx, HostKey(id1))
	defer s.hs.Delete(s.ctx, HostKey(id2))
	defer s.hs.Delete(s.ctx, HostKey(id3))

	host, err := Build("", "65535", "pool1", "", []string{}...)
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

func (s *S) Test_GetHostByIP(t *C) {
	host, err := Build("", "65535", "pool1", "")
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}

	host.ID = "deadb70f"
	host.IPs = append(host.IPs, HostIPResource{IPAddress: "111.22.333.4"})
	if err := s.hs.Put(s.ctx, HostKey(host.ID), host); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer s.hs.Delete(s.ctx, HostKey(host.ID))

	result, err := s.hs.GetHostByIP(s.ctx, "111.22.333.4")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if result == nil || result.ID != host.ID {
		t.Errorf("Expected %v, got %v", host, result)
	}

	result, err = s.hs.GetHostByIP(s.ctx, "abc123")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}
