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
	ctx   datastore.Context
	store *Store
}

func (s *S) SetUpTest(c *C) {
	s.ElasticTest.SetUpTest(c)
	datastore.Register(s.Driver())
	s.ctx = datastore.Get()
	s.store = NewStore()
}

func (s *S) Test_AddressAssignmentCRUD(t *C) {
	defer s.store.Delete(s.ctx, Key("testID"))

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

	if err := s.store.Get(s.ctx, Key(assignment.ID), &assignment2); !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Expected ErrNoSuchEntity, got: %v", err)
	}

	err := s.store.Put(s.ctx, Key(assignment.ID), assignment)
	t.Assert(err, IsNil)

	//Test update
	assignment.EndpointName = "BLAM"
	err = s.store.Put(s.ctx, Key(assignment.ID), assignment)
	t.Assert(err, IsNil)

	err = s.store.Get(s.ctx, Key(assignment.ID), &assignment2)
	t.Assert(err, IsNil)

	if assignment.EndpointName != assignment2.EndpointName {
		t.Errorf("assignments did not match after update")
	}

	//test delete
	err = s.store.Delete(s.ctx, Key(assignment.ID))
	t.Assert(err, IsNil)
	err = s.store.Get(s.ctx, Key(assignment.ID), &assignment2)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}

}

func (s *S) Test_GetAllAddressAssignments(t *C) {
	defer s.store.Delete(s.ctx, Key("testID1"))
	defer s.store.Delete(s.ctx, Key("testID2"))
	defer s.store.Delete(s.ctx, Key("testID3"))

	assignment1 := AddressAssignment{
		ID:             "testID1",
		AssignmentType: "static",
		ServiceID:      "svcID1",
		EndpointName:   "epname1",
		IPAddr:         "10.0.1.5",
		HostID:         "hostid",
		Port:           500,
	}
	assignment2 := assignment1
	assignment2.ID = "testID2"
	assignment2.ServiceID = "svcID2"
	assignment3 := assignment1
	assignment3.ID = "testID3"
	assignment3.ServiceID = "svcID3"
	assignment3.EndpointName = "epname2"

	// Verify get-all on an empty DB returns correct values
	addrs, err := s.store.GetAllAddressAssignments(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(addrs), Equals, 0)

	err = s.store.Put(s.ctx, Key(assignment1.ID), &assignment1)
	t.Assert(err, IsNil)

	// Verify get-all on a db w/just 1 entry is correct
	addrs, err = s.store.GetAllAddressAssignments(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(addrs), Equals, 1)
	assignment1.IfPrimaryTerm = 1
	t.Assert(addrs[0], Equals, assignment1)

	err = s.store.Put(s.ctx, Key(assignment2.ID), &assignment2)
	t.Assert(err, IsNil)
	err = s.store.Put(s.ctx, Key(assignment3.ID), &assignment3)
	t.Assert(err, IsNil)

	// Verify get-all on a db w/multiple entries is correct
	addrs, err = s.store.GetAllAddressAssignments(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(addrs), Equals, 3)

	addrMap := make(map[string]AddressAssignment)
	for _, addr := range addrs {
		addrMap[addr.ID] = addr
	}
	result, ok := addrMap[assignment1.ID]
	t.Assert(ok, Equals, true)
	t.Assert(result, Equals, assignment1)

	assignment2.IfPrimaryTerm = 1
	assignment2.IfSeqNo = 1
	result, ok = addrMap[assignment2.ID]
	t.Assert(ok, Equals, true)
	t.Assert(result, Equals, assignment2)

	assignment3.IfPrimaryTerm = 1
	assignment3.IfSeqNo = 2
	result, ok = addrMap[assignment3.ID]
	t.Assert(ok, Equals, true)
	t.Assert(result, Equals, assignment3)
}
