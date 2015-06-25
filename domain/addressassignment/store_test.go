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

package addressassignment

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
	ps  *Store
}

func (s *S) SetUpTest(c *C) {
	s.ElasticTest.SetUpTest(c)
	datastore.Register(s.Driver())
	s.ctx = datastore.Get()
	s.ps = NewStore()
}

func (s *S) Test_AddressAssignmentCRUD(t *C) {
	defer s.ps.Delete(s.ctx, Key("testid"))

	assignment := &AddressAssignment{
		ID:             "testID",
		AssignmentType: "static",
		ServiceID:      "svcid",
		EndpointName:   "epname",
		IPAddr:         "10.0.1.5",
		PoolID:         "",
		HostID:         "hostid",
		Port:           500,
	}
	assignment2 := AddressAssignment{}

	if err := s.ps.Get(s.ctx, Key(assignment.ID), &assignment2); !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Expected ErrNoSuchEntity, got: %v", err)
	}

	err := s.ps.Put(s.ctx, Key(assignment.ID), assignment)
	t.Assert(err, IsNil)

	//Test update
	assignment.EndpointName = "BLAM"
	err = s.ps.Put(s.ctx, Key(assignment.ID), assignment)
	t.Assert(err, IsNil)

	err = s.ps.Get(s.ctx, Key(assignment.ID), &assignment2)
	t.Assert(err, IsNil)

	if assignment.EndpointName != assignment2.EndpointName {
		t.Errorf("assignments did not match after update")
	}

	//test delete
	err = s.ps.Delete(s.ctx, Key(assignment.ID))
	t.Assert(err, IsNil)
	err = s.ps.Get(s.ctx, Key(assignment.ID), &assignment2)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}

}

func (s *S) Test_GetServiceAddressAssignments(t *C) {
	expected := []AddressAssignment{}
	actual, err := s.ps.GetServiceAddressAssignments(s.ctx, "test_service_1")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign := AddressAssignment{
		ID:             "test_id_1",
		AssignmentType: "static",
		ServiceID:      "test_service_1",
		EndpointName:   "endpoint_name_1",
		IPAddr:         "10.0.1.5",
		PoolID:         "test_pool_1",
		HostID:         "hostid_1",
		Port:           1000,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	expected = append(expected, assign)
	actual, err = s.ps.GetServiceAddressAssignments(s.ctx, "test_service_1")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign = AddressAssignment{
		ID:             "test_id_2",
		AssignmentType: "virtual",
		ServiceID:      "test_service_1",
		EndpointName:   "endpoint_name_2",
		IPAddr:         "10.0.2.22",
		PoolID:         "test_pool_1",
		HostID:         "",
		Port:           1100,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	expected = append(expected, assign)
	actual, err = s.ps.GetServiceAddressAssignments(s.ctx, "test_service_1")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign = AddressAssignment{
		ID:             "test_id_3",
		AssignmentType: "static",
		ServiceID:      "test_service_2",
		EndpointName:   "endpoint_name_3",
		IPAddr:         "10.0.15.20",
		PoolID:         "",
		HostID:         "hostid_2",
		Port:           2222,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	actual, err = s.ps.GetServiceAddressAssignments(s.ctx, "test_service_1")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)
}

func (s *S) Test_GetServiceAddressAssignmentsByIP(t *C) {
	expected := []AddressAssignment{}
	actual, err := s.ps.GetServiceAddressAssignmentsByIP(s.ctx, "8.8.8.8")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign := AddressAssignment{
		ID:             "test_id_1",
		AssignmentType: "static",
		ServiceID:      "test_service_1",
		EndpointName:   "endpoint_name_1",
		IPAddr:         "8.8.8.8",
		PoolID:         "",
		HostID:         "hostid_1",
		Port:           1000,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	expected = append(expected, assign)
	actual, err = s.ps.GetServiceAddressAssignmentsByIP(s.ctx, "8.8.8.8")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign = AddressAssignment{
		ID:             "test_id_2",
		AssignmentType: "static",
		ServiceID:      "test_service_2",
		EndpointName:   "endpoint_name_2",
		IPAddr:         "8.8.8.8",
		PoolID:         "",
		HostID:         "hostid_1",
		Port:           1100,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	expected = append(expected, assign)
	actual, err = s.ps.GetServiceAddressAssignmentsByIP(s.ctx, "8.8.8.8")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign = AddressAssignment{
		ID:             "test_id_3",
		AssignmentType: "virtual",
		ServiceID:      "test_service_3",
		EndpointName:   "endpoint_name_3",
		IPAddr:         "10.22.15.5",
		PoolID:         "test_pool_1",
		HostID:         "",
		Port:           2222,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	actual, err = s.ps.GetServiceAddressAssignmentsByIP(s.ctx, "8.8.8.8")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)
}

func (s *S) Test_GetServiceAddressAssignmentsByHost(t *C) {
	expected := []AddressAssignment{}
	actual, err := s.ps.GetServiceAddressAssignmentsByHost(s.ctx, "hostid_1")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign := AddressAssignment{
		ID:             "test_id_1",
		AssignmentType: "static",
		ServiceID:      "test_service_1",
		EndpointName:   "endpoint_name_1",
		IPAddr:         "8.8.8.8",
		PoolID:         "",
		HostID:         "hostid_1",
		Port:           1000,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	expected = append(expected, assign)
	actual, err = s.ps.GetServiceAddressAssignmentsByHost(s.ctx, "hostid_1")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign = AddressAssignment{
		ID:             "test_id_2",
		AssignmentType: "static",
		ServiceID:      "test_service_2",
		EndpointName:   "endpoint_name_2",
		IPAddr:         "10.15.22.33",
		PoolID:         "",
		HostID:         "hostid_1",
		Port:           1100,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	expected = append(expected, assign)
	actual, err = s.ps.GetServiceAddressAssignmentsByHost(s.ctx, "hostid_1")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign = AddressAssignment{
		ID:             "test_id_3",
		AssignmentType: "virtual",
		ServiceID:      "test_service_3",
		EndpointName:   "endpoint_name_3",
		IPAddr:         "10.22.15.5",
		PoolID:         "test_pool_1",
		HostID:         "",
		Port:           2222,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	actual, err = s.ps.GetServiceAddressAssignmentsByHost(s.ctx, "hostid_1")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)
}

func (s *S) Test_GetServiceAddressAssignmentsByPort(t *C) {
	expected := []AddressAssignment{}
	actual, err := s.ps.GetServiceAddressAssignmentsByPort(s.ctx, 1000)
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign := AddressAssignment{
		ID:             "test_id_1",
		AssignmentType: "static",
		ServiceID:      "test_service_1",
		EndpointName:   "endpoint_name_1",
		IPAddr:         "8.8.8.8",
		PoolID:         "",
		HostID:         "hostid_1",
		Port:           1000,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	expected = append(expected, assign)
	actual, err = s.ps.GetServiceAddressAssignmentsByPort(s.ctx, 1000)
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign = AddressAssignment{
		ID:             "test_id_2",
		AssignmentType: "static",
		ServiceID:      "test_service_2",
		EndpointName:   "endpoint_name_2",
		IPAddr:         "10.15.22.33",
		PoolID:         "",
		HostID:         "hostid_1",
		Port:           1000,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	expected = append(expected, assign)
	actual, err = s.ps.GetServiceAddressAssignmentsByPort(s.ctx, 1000)
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)

	assign = AddressAssignment{
		ID:             "test_id_3",
		AssignmentType: "virtual",
		ServiceID:      "test_service_3",
		EndpointName:   "endpoint_name_3",
		IPAddr:         "10.22.15.5",
		PoolID:         "test_pool_1",
		HostID:         "",
		Port:           2222,
	}
	err = s.ps.Put(s.ctx, Key(assign.ID), &assign)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(assign.ID))
	assign.DatabaseVersion++
	actual, err = s.ps.GetServiceAddressAssignmentsByPort(s.ctx, 1000)
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)
}

func (s *S) Test_FindAssignmentByServiceEndpoint(t *C) {
	expected := &AddressAssignment{
		ID:             "test_id_1",
		AssignmentType: "static",
		ServiceID:      "test_service_1",
		EndpointName:   "endpoint_name_1",
		IPAddr:         "8.8.8.8",
		PoolID:         "",
		HostID:         "hostid_1",
		Port:           1000,
	}
	err := s.ps.Put(s.ctx, Key(expected.ID), expected)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(expected.ID))
	expected.DatabaseVersion++

	actual, err := s.ps.FindAssignmentByServiceEndpoint(s.ctx, "test_service_1", "endpoint_name_1")
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)
	actual, err = s.ps.FindAssignmentByServiceEndpoint(s.ctx, "test_service_1", "endpoint_name_2")
	t.Assert(err, IsNil)
	t.Assert(actual, IsNil)
	actual, err = s.ps.FindAssignmentByServiceEndpoint(s.ctx, "test_service_2", "endpoint_name_1")
	t.Assert(err, IsNil)
	t.Assert(actual, IsNil)
	actual, err = s.ps.FindAssignmentByServiceEndpoint(s.ctx, "test_service_2", "endpoint_name_2")
	t.Assert(err, IsNil)
	t.Assert(actual, IsNil)
}

func (s *S) Test_FindAssignmentByHostPort(t *C) {
	expected := &AddressAssignment{
		ID:             "test_id_1",
		AssignmentType: "static",
		ServiceID:      "test_service_1",
		EndpointName:   "endpoint_name_1",
		IPAddr:         "8.8.8.8",
		PoolID:         "",
		HostID:         "hostid_1",
		Port:           1000,
	}
	err := s.ps.Put(s.ctx, Key(expected.ID), expected)
	t.Assert(err, IsNil)
	defer s.ps.Delete(s.ctx, Key(expected.ID))
	expected.DatabaseVersion++

	actual, err := s.ps.FindAssignmentByHostPort(s.ctx, "8.8.8.8", 1000)
	t.Assert(err, IsNil)
	t.Assert(actual, DeepEquals, expected)
	actual, err = s.ps.FindAssignmentByHostPort(s.ctx, "8.8.8.8", 1100)
	t.Assert(err, IsNil)
	t.Assert(actual, IsNil)
	actual, err = s.ps.FindAssignmentByHostPort(s.ctx, "10.20.1.2", 1000)
	t.Assert(err, IsNil)
	t.Assert(actual, IsNil)
	actual, err = s.ps.FindAssignmentByHostPort(s.ctx, "10.20.1.2", 1100)
	t.Assert(err, IsNil)
	t.Assert(actual, IsNil)
}
