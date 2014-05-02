// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package pool

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
	ps  *Store
}

func (s *S) SetUpTest(c *C) {
	s.ElasticTest.SetUpTest(c)
	datastore.Register(s.Driver())
	s.ctx = datastore.Get()
	s.ps = NewStore()
}

func (s *S) Test_PoolCRUD(t *C) {
	defer s.ps.Delete(s.ctx, Key("testid"))

	pool := New("testID")
	pool2 := ResourcePool{}

	time.Sleep(1000 * time.Millisecond)
	if err := s.ps.Get(s.ctx, Key(pool.ID), &pool2); !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Expected ErrNoSuchEntity, got: %v", err)
	}

	err := s.ps.Put(s.ctx, Key(pool.ID), pool)
	if err != nil {
		t.Errorf("Unexpected failure creating pool %-v", pool)
	}

	//Test update
	pool.CoreLimit = 1024
	err = s.ps.Put(s.ctx, Key(pool.ID), pool)
	err = s.ps.Get(s.ctx, Key(pool.ID), &pool2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if pool.CoreLimit != pool2.CoreLimit {
		t.Errorf("pools did not match after update")
	}

	//test delete
	err = s.ps.Delete(s.ctx, Key(pool.ID))
	err = s.ps.Get(s.ctx, Key(pool.ID), &pool2)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}

}

func (s *S) Test_GetPools(t *C) {
	defer s.ps.Delete(s.ctx, Key("Test_GetPools1"))
	defer s.ps.Delete(s.ctx, Key("Test_GetPools2"))

	pools, err := s.ps.GetResourcePools(s.ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(pools) != 0 {
		t.Errorf("Expected %v results, got %v :%#v", 2, len(pools), pools)
	}

	pool := New("Test_GetPools1")
	err = s.ps.Put(s.ctx, Key(pool.ID), pool)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	time.Sleep(2000 * time.Millisecond)
	pools, err = s.ps.GetResourcePools(s.ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(pools) != 1 {
		t.Errorf("Expected %v results, got %v :%v", 1, len(pools), pools)
	}

	pool.ID = "Test_GetHosts2"
	err = s.ps.Put(s.ctx, Key(pool.ID), pool)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	time.Sleep(2000 * time.Millisecond)
	pools, err = s.ps.GetResourcePools(s.ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(pools) != 2 {
		t.Errorf("Expected %v results, got %v :%v", 2, len(pools), pools)
	}

}
