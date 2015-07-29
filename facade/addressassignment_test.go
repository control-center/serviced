// Copyright 2015 The Serviced Authors.
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

package facade

import (
	"testing"

	. "gopkg.in/check.v1"

	aa "github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
)

func Test_getPorts(t *testing.T) {
	getendpoints := func(plist []uint16) []service.ServiceEndpoint {
		endpoints := make([]service.ServiceEndpoint, len(plist))
		for i, p := range plist {
			endpoints[i].AddressConfig.Port = p
			endpoints[i].AddressConfig.Protocol = "tcp"
		}
		return endpoints
	}

	// case 1: duplicate ports
	endpoints := getendpoints([]uint16{1000, 1000})
	actual, err := getPorts(endpoints)
	if err != ErrMultiPorts {
		t.Fatalf("Expected %s; Got %s", ErrMultiPorts, err)
	}

	// case 2: success
	expected := []uint16{1000, 2000, 300, 405}
	endpoints = getendpoints(expected)
	actual, err = getPorts(endpoints)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if len(actual) != len(expected) {
		t.Errorf("Mismatch: expected %+v; got %+v", expected, actual)
	}
	for _, p := range expected {
		if _, ok := actual[p]; !ok {
			t.Errorf("Missing port %d", p)
		}
	}
}

func Test_Ports_List(t *testing.T) {
	expected := map[uint16]struct{}{
		1000: struct{}{},
		2000: struct{}{},
		300:  struct{}{},
		405:  struct{}{},
	}

	actual := Ports(expected).List()
	if len(actual) != len(expected) {
		t.Errorf("Mismatch: expected %+v; got %+v", expected, actual)
	}
	for _, p := range actual {
		if _, ok := expected[p]; !ok {
			t.Errorf("Missing port %d", p)
		}
	}
}

func Test_Ports_GetIP(t *testing.T) {
	allports := Ports(map[uint16]struct{}{
		1000: struct{}{},
		2000: struct{}{},
		300:  struct{}{},
		405:  struct{}{},
	})

	expected := map[uint16]struct{}{
		1000: struct{}{},
		2000: struct{}{},
	}
	assigns := []aa.AddressAssignment{
		{IPAddr: "10.20.1.2", Port: 300},
		{IPAddr: "10.20.1.2", Port: 405},
	}
	ipaddr, ports := allports.GetIP(assigns)
	if ipaddr != "10.20.1.2" {
		t.Errorf("Expected ip addr %s; Got %s", "10.20.1.2", ipaddr)
	}
	if len(ports) != len(expected) {
		t.Errorf("Mismatch: expected %+v; got %+v", expected, ports)
	}
	for _, p := range ports {
		if _, ok := expected[p]; !ok {
			t.Errorf("Missing port %d", p)
		}
	}

	expected = map[uint16]struct{}{
		1000: struct{}{},
		2000: struct{}{},
		300:  struct{}{},
		405:  struct{}{},
	}
	assigns = []aa.AddressAssignment{
		{IPAddr: "10.20.1.2", Port: 300},
		{IPAddr: "10.20.1.3", Port: 405},
	}
	ipaddr, ports = allports.GetIP(assigns)
	if ipaddr != "" {
		t.Errorf("Expected empty ip addr; Got %s", ipaddr)
	}
	if len(ports) != len(expected) {
		t.Errorf("Mismatch: expected %+v; got %+v", expected, ports)
	}
	for _, p := range ports {
		if _, ok := expected[p]; !ok {
			t.Errorf("Missing port %d", p)
		}
	}
	ipaddr, ports = allports.GetIP([]aa.AddressAssignment{})
	if ipaddr != "" {
		t.Errorf("Expected empty ip addr; Got %s", ipaddr)
	}
	if len(ports) != len(expected) {
		t.Errorf("Mismatch: expected %+v; got %+v", expected, ports)
	}
	for _, p := range ports {
		if _, ok := expected[p]; !ok {
			t.Errorf("Missing port %d", p)
		}
	}
}

func Test_Ports_SetIP(t *testing.T) {
	allports := Ports(map[uint16]struct{}{
		1000: struct{}{},
		2000: struct{}{},
		300:  struct{}{},
		405:  struct{}{},
	})

	expected := map[uint16]struct{}{
		1000: struct{}{},
		2000: struct{}{},
	}
	assigns := []aa.AddressAssignment{
		{IPAddr: "10.20.1.2", Port: 300},
		{IPAddr: "10.20.1.2", Port: 405},
	}
	ports := allports.SetIP("10.20.1.2", assigns)
	if len(ports) != len(expected) {
		t.Errorf("Mismatch: expected %+v; got %+v", expected, ports)
	}
	for _, p := range ports {
		if _, ok := expected[p]; !ok {
			t.Errorf("Missing port %d", p)
		}
	}

	expected = map[uint16]struct{}{
		1000: struct{}{},
		2000: struct{}{},
		300:  struct{}{},
		405:  struct{}{},
	}
	assigns = []aa.AddressAssignment{
		{IPAddr: "10.20.1.2", Port: 300},
		{IPAddr: "10.20.1.3", Port: 405},
	}
	ports = allports.SetIP("10.20.1.4", assigns)
	if len(ports) != len(expected) {
		t.Errorf("Mismatch: expected %+v; got %+v", expected, ports)
	}
	for _, p := range ports {
		if _, ok := expected[p]; !ok {
			t.Errorf("Missing port %d", p)
		}
	}
}

func (ft *FacadeTest) Test_addAddrAssignment(t *C) {
	// success
	expected := aa.AddressAssignment{
		AssignmentType: "static",
		HostID:         "hostid_1",
		PoolID:         "",
		IPAddr:         "10.20.1.2",
		Port:           2000,
		ServiceID:      "test_service_1",
		EndpointName:   "endpoint_name_1",
	}

	err := ft.Facade.addAddrAssignment(ft.CTX, expected)
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveAddrAssignmentsByService(ft.CTX, expected.ServiceID)

	// service and endpoint exists
	actual, err := ft.Facade.GetAddrAssignmentByServiceEndpoint(ft.CTX, expected.ServiceID, expected.EndpointName)
	t.Assert(err, IsNil)
	expected.ID = actual.ID
	expected.DatabaseVersion++
	t.Assert(actual, DeepEquals, &expected)
	err = ft.Facade.addAddrAssignment(ft.CTX, aa.AddressAssignment{
		AssignmentType: "virtual",
		HostID:         "",
		PoolID:         "test-pool",
		IPAddr:         "10.111.15.44",
		Port:           1234,
		ServiceID:      "test_service_1",
		EndpointName:   "endpoint_name_1",
	})
	t.Check(err, Equals, ErrAddrAssignExists)

	// ip and port exists
	actual, err = ft.Facade.GetAddrAssignmentByIPPort(ft.CTX, expected.IPAddr, expected.Port)
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, &expected)
	err = ft.Facade.addAddrAssignment(ft.CTX, aa.AddressAssignment{
		AssignmentType: "virtual",
		HostID:         "",
		PoolID:         "test-pool",
		IPAddr:         "10.20.1.2",
		Port:           2000,
		ServiceID:      "test_service_2",
		EndpointName:   "endpoint_name_2",
	})
	t.Check(err, Equals, ErrAddrAssignExists)
}