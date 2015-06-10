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

package govpool

import (
	"testing"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	. "gopkg.in/check.v1"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&S{
	ElasticTest: elastic.ElasticTest{
		Index:    "controlplane",
		Mappings: []elastic.Mapping{MAPPING},
	},
})

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

func (s *S) TestGovernedPool_CRUD(t *C) {
	expectedPool := &GovernedPool{
		PoolID:        "test_pool_id",
		RemotePoolID:  "test_remote_pool_id",
		RemoteAddress: "test_remote_address",
	}

	// insert
	err := s.store.Put(s.ctx, expectedPool)
	t.Assert(err, IsNil)
	defer s.store.Delete(s.ctx, expectedPool.PoolID)
	expectedPool.DatabaseVersion++
	actualPool, err := s.store.Get(s.ctx, expectedPool.PoolID)
	t.Assert(err, IsNil)
	t.Assert(actualPool, DeepEquals, expectedPool)

	// update
	expectedPool.RemotePoolID = "test_remote_pool_id 2"
	expectedPool.RemoteAddress = "test_remote_address 2"
	err = s.store.Put(s.ctx, expectedPool)
	t.Assert(err, IsNil)
	expectedPool.DatabaseVersion++
	actualPool, err = s.store.Get(s.ctx, expectedPool.PoolID)
	t.Assert(err, IsNil)
	t.Assert(actualPool, DeepEquals, expectedPool)

	// delete
	err = s.store.Delete(s.ctx, expectedPool.PoolID)
	t.Assert(err, IsNil)
	actualPool, err = s.store.Get(s.ctx, expectedPool.PoolID)
	t.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
}

func (s *S) TestGovernedPool_GetGovernedPools(t *C) {
	expectedPools := []GovernedPool{}

	actualPools, err := s.store.GetGovernedPools(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(actualPools, DeepEquals, expectedPools)

	pool := New("test_pool_1", "test_remote_1", "remote_address_1")
	err = s.store.Put(s.ctx, pool)
	t.Assert(err, IsNil)
	defer s.store.Delete(s.ctx, pool.PoolID)
	pool.DatabaseVersion++
	expectedPools = append(expectedPools, *pool)
	actualPools, err = s.store.GetGovernedPools(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(actualPools, DeepEquals, expectedPools)

	pool = New("test_pool_2", "test_remote_1", "remote_address_1")
	err = s.store.Put(s.ctx, pool)
	t.Assert(err, IsNil)
	defer s.store.Delete(s.ctx, pool.PoolID)
	pool.DatabaseVersion++
	expectedPools = append(expectedPools, *pool)
	actualPools, err = s.store.GetGovernedPools(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(actualPools, DeepEquals, expectedPools)
}