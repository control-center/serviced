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

package facade

import (
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
	. "gopkg.in/check.v1"
)

func (s *FacadeIntegrationTest) Test_HostCRUD(t *C) {
	testid := "deadb10f"
	poolid := "pool-id"
	defer s.Facade.RemoveHost(s.CTX, testid)

	//fill host with required values
	h, err := host.Build("", "65535", poolid, "", []string{}...)
	h.ID = testid
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	glog.Infof("Facade test add host %v", h)
	err = s.Facade.AddHost(s.CTX, h)
	//should fail since pool doesn't exist
	if err == nil {
		t.Errorf("Expected error: %v", err)
	}

	//create pool for test
	rp := pool.New(poolid)
	if err := s.Facade.AddResourcePool(s.CTX, rp); err != nil {
		t.Fatalf("Could not add pool for test: %v", err)
	}
	defer s.Facade.RemoveResourcePool(s.CTX, poolid)

	err = s.Facade.AddHost(s.CTX, h)
	if err != nil {
		t.Errorf("Unexpected error adding host: %v", err)
	}

	//Test re-add fails
	err = s.Facade.AddHost(s.CTX, h)
	if err == nil {
		t.Errorf("Expected already exists error: %v", err)
	}

	h2, err := s.Facade.GetHost(s.CTX, testid)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if h2 == nil {
		t.Error("Unexpected nil host")

	} else if !host.HostEquals(t, h, h2) {
		t.Error("Hosts did not match")
	}

	//Test update
	h.Memory = 1024
	err = s.Facade.UpdateHost(s.CTX, h)
	h2, err = s.Facade.GetHost(s.CTX, testid)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !host.HostEquals(t, h, h2) {
		t.Error("Hosts did not match")
	}

	//test delete
	err = s.Facade.RemoveHost(s.CTX, testid)
	h2, err = s.Facade.GetHost(s.CTX, testid)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func (s *FacadeIntegrationTest) TestRestoreHosts(c *C) {
	// create pool for testing
	resourcePool := pool.New("poolid")
	s.Facade.AddResourcePool(s.CTX, resourcePool)
	defer s.Facade.RemoveResourcePool(s.CTX, "poolid")

	// success
	hosts1 := []host.Host{
		{
			ID:      "deadb11f",
			PoolID:  "poolid",
			Name:    "h1",
			IPAddr:  "192.168.0.1",
			RPCPort: 65535,
			IPs: []host.HostIPResource{
				{
					HostID:    "deadb11f",
					IPAddress: "192.168.0.1",
				},
			},
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
		},
	}
	defer s.Facade.RemoveHost(s.CTX, "deadb11f")
	err := s.Facade.RestoreHosts(s.CTX, hosts1)
	c.Assert(err, IsNil)
	actual, err := s.Facade.GetHosts(s.CTX)
	c.Assert(err, IsNil)
	for i := range actual {
		actual[i].DatabaseVersion = 0
		actual[i].CreatedAt = time.Time{}
		actual[i].UpdatedAt = time.Time{}
	}
	c.Assert(actual, DeepEquals, hosts1)

	// different host with the same ip
	hosts2 := []host.Host{
		{
			ID:      "deadb11e",
			PoolID:  "poolid",
			Name:    "h2",
			IPAddr:  "192.168.1.1",
			RPCPort: 65535,
			IPs: []host.HostIPResource{
				{
					HostID:    "deadb11e",
					IPAddress: "192.168.1.1",
				}, {
					HostID:    "deadb11e",
					IPAddress: "192.168.0.1",
				},
			},
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
		},
	}
	err = s.Facade.RestoreHosts(s.CTX, hosts2)
	c.Assert(err, IsNil)
	hosts2[0].DatabaseVersion++
	actual, err = s.Facade.GetHosts(s.CTX)
	c.Assert(err, IsNil)
	for i := range actual {
		actual[i].DatabaseVersion = 0
		actual[i].CreatedAt = time.Time{}
		actual[i].UpdatedAt = time.Time{}
	}
	c.Assert(actual, DeepEquals, hosts1)

	// host in different pool
	resourcePool = pool.New("poolid2")
	s.Facade.AddResourcePool(s.CTX, resourcePool)
	defer s.Facade.RemoveResourcePool(s.CTX, "poolid2")
	hosts3 := []host.Host{
		{
			ID:      "deadb11f",
			PoolID:  "poolid2",
			Name:    "h1",
			IPAddr:  "192.168.0.1",
			RPCPort: 65535,
			IPs: []host.HostIPResource{
				{
					HostID:    "deadb11f",
					IPAddress: "192.168.0.1",
				},
			},
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
		},
	}
	err = s.Facade.RestoreHosts(s.CTX, hosts3)
	c.Assert(err, NotNil)
	actual, err = s.Facade.GetHosts(s.CTX)
	c.Assert(err, IsNil)
	for i := range actual {
		actual[i].DatabaseVersion = 0
		actual[i].CreatedAt = time.Time{}
		actual[i].UpdatedAt = time.Time{}
	}
	c.Assert(actual, DeepEquals, hosts1)
}

func (s *FacadeIntegrationTest) Test_HostRemove(t *C) {
	//create pool for testing
	resoucePool := pool.New("poolid")
	s.Facade.AddResourcePool(s.CTX, resoucePool)
	defer s.Facade.RemoveResourcePool(s.CTX, "poolid")

	//add host1
	h1 := host.Host{
		ID:      "deadb11f",
		PoolID:  "poolid",
		Name:    "h1",
		IPAddr:  "192.168.0.1",
		RPCPort: 65535,
		IPs: []host.HostIPResource{
			host.HostIPResource{
				HostID:    "deadb11f",
				IPAddress: "192.168.0.1",
			},
		},
	}
	err := s.Facade.AddHost(s.CTX, &h1)
	if err != nil {
		t.Fatalf("Failed to add host %+v: %s", h1, err)
	}

	//add host2
	h2 := host.Host{
		ID:      "deadb12f",
		PoolID:  "poolid",
		Name:    "h2",
		IPAddr:  "192.168.0.2",
		RPCPort: 65535,
		IPs: []host.HostIPResource{
			host.HostIPResource{
				HostID:    "deadb12f",
				IPAddress: "192.168.0.2",
			},
		},
	}
	err = s.Facade.AddHost(s.CTX, &h2)
	if err != nil {
		t.Fatalf("Failed to add host %+v: %s", h2, err)
	}
	defer s.Facade.RemoveHost(s.CTX, "host2")

	//add service with address assignment
	s1, _ := service.NewService()
	s1.Name = "name"
	s1.PoolID = "poolid"
	s1.DeploymentID = "deployment_id"
	s1.Launch = "manual"
	s1.Endpoints = []service.ServiceEndpoint{
		service.ServiceEndpoint{},
	}
	s1.Endpoints[0].Name = "name"
	s1.Endpoints[0].AddressConfig = servicedefinition.AddressResourceConfig{Port: 123, Protocol: "tcp"}
	aa := addressassignment.AddressAssignment{ID: "id", HostID: h1.ID}
	s1.Endpoints[0].SetAssignment(aa)
	err = s.Facade.AddService(s.CTX, *s1)
	if err != nil {
		t.Fatalf("Failed to add service %+v: %s", s1, err)
	}
	defer s.Facade.RemoveService(s.CTX, s1.ID)

	request := addressassignment.AssignmentRequest{ServiceID: s1.ID, IPAddress: h1.IPAddr, AutoAssignment: false}
	if err = s.Facade.AssignIPs(s.CTX, request); err != nil {
		t.Fatalf("Failed assigning ip to service: %s", err)
	}

	var serviceRequest dao.ServiceRequest
	services, _ := s.Facade.GetServices(s.CTX, serviceRequest)
	if len(services) <= 0 {
		t.Fatalf("Expected one service in context")
	}
	if len(services[0].Endpoints) <= 0 {
		t.Fatalf("Expected service with one endpoint in context")
	}
	ep := services[0].Endpoints[0]
	if ep.AddressAssignment.IPAddr != h1.IPAddr && ep.AddressAssignment.HostID != h1.ID {
		t.Fatalf("Incorrect IPAddress and HostID before remove host")
	}

	//remove host1
	err = s.Facade.RemoveHost(s.CTX, h1.ID)
	if err != nil {
		t.Fatalf("Failed to remove host: %s", err)
	}
	services, _ = s.Facade.GetServices(s.CTX, serviceRequest)

	if len(services) <= 0 {
		t.Fatalf("Expected one service in context")
	}

	if len(services[0].Endpoints) <= 0 {
		t.Fatalf("Expected service with one endpoint in context")
	}

	ep = services[0].Endpoints[0]
	if ep.AddressAssignment.IPAddr == h2.IPAddr || ep.AddressAssignment.HostID == h2.ID {
		t.Fatalf("Incorrect IPAddress and HostID after remove host")
	}
}
