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
	"github.com/control-center/serviced/auth"
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
	_, err = s.Facade.AddHost(s.CTX, h)
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

	_, err = s.Facade.AddHost(s.CTX, h)
	if err != nil {
		t.Errorf("Unexpected error adding host: %v", err)
	}

	//Test re-add fails
	_, err = s.Facade.AddHost(s.CTX, h)
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

func (s *FacadeIntegrationTest) Test_HostGetKey(t *C) {
	testid := "deadb10f"
	poolid := "pool-id"

	// confirm error on gethostkey for nonexistent host
	public, err := s.Facade.GetHostKey(s.CTX, testid)
	t.Assert(err, NotNil)

	//create pool for test
	rp := pool.New(poolid)
	err = s.Facade.AddResourcePool(s.CTX, rp)
	t.Assert(err, IsNil)
	defer s.Facade.RemoveResourcePool(s.CTX, poolid)

	// create host for test
	h, err := host.Build("", "65535", poolid, "", []string{}...)
	t.Assert(err, IsNil)
	h.ID = testid
	private, err := s.Facade.AddHost(s.CTX, h)
	t.Assert(err, IsNil)
	defer s.Facade.RemoveHost(s.CTX, testid)

	// get host key
	public, err = s.Facade.GetHostKey(s.CTX, testid)
	t.Assert(err, IsNil)

	// confirm that the public and private keys correspond
	signer, err := auth.RSASignerFromPEM(private)
	t.Assert(err, IsNil)
	verifier, err := auth.RSAVerifierFromPEM(public)
	t.Assert(err, IsNil)
	message := []byte("Now is the time for all good")
	signed, err := signer.Sign(message)
	t.Assert(err, IsNil)
	err = verifier.Verify(message, signed)
	t.Assert(err, IsNil)
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
	_, err := s.Facade.AddHost(s.CTX, &h1)
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
	_, err = s.Facade.AddHost(s.CTX, &h2)
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
	s1.ImageID = "image_id"
	s1.Endpoints = []service.ServiceEndpoint{
		service.ServiceEndpoint{},
	}
	s1.Endpoints[0].Name = "name"
	s1.Endpoints[0].Application = "Application"
	s1.Endpoints[0].Purpose = "export"
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
