// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package addressassignment

import (
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"
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

//func (s *S) Test_GetAddressAssignments(t *C) {
//
//	assignments, err := s.ps.GetResourcePools(s.ctx)
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	} else if len(assignments) != 0 {
//		t.Errorf("Expected %v results, got %v :%#v", 0, len(assignments), assignments)
//	}
//
//	assignment := New("Test_GetPools1")
//	err = s.ps.Put(s.ctx, Key(assignment.ID), assignment)
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	}
//	assignments, err = s.ps.GetResourcePools(s.ctx)
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	} else if len(assignments) != 1 {
//		t.Errorf("Expected %v results, got %v :%v", 1, len(assignments), assignments)
//	}
//
//	assignment.ID = "Test_GetHosts2"
//	err = s.ps.Put(s.ctx, Key(assignment.ID), assignment)
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	}
//
//	assignments, err = s.ps.GetResourcePools(s.ctx)
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	} else if len(assignments) != 2 {
//		t.Errorf("Expected %v results, got %v :%v", 2, len(assignments), assignments)
//	}
//
//}
