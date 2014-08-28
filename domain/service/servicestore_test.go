// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package service

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/domain/servicedefinition"
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

func (s *S) Test_ServiceCRUD(t *C) {
	svc := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}

	confFile := servicedefinition.ConfigFile{Content: "Test content", Filename: "testname"}
	svc.OriginalConfigs = map[string]servicedefinition.ConfigFile{"testname": confFile}

	svc2, err := s.store.Get(s.ctx, svc.ID)
	t.Assert(err, NotNil)
	if !datastore.IsErrNoSuchEntity(err) {
		t.Fatalf("unexpected error type: %v", err)
	}

	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	//Test update
	svc.Description = "new description"
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svc2, err = s.store.Get(s.ctx, svc.ID)
	t.Assert(err, IsNil)

	t.Assert(svc2.Description, Equals, svc.Description)
	t.Assert(len(svc2.ConfigFiles), Equals, len(svc.OriginalConfigs))
	t.Assert(svc2.ConfigFiles["testname"], Equals, svc.OriginalConfigs["testname"])

	//test delete
	err = s.store.Delete(s.ctx, svc.ID)
	t.Assert(err, IsNil)

	svc2, err = s.store.Get(s.ctx, svc.ID)
	t.Assert(err, NotNil)
	if !datastore.IsErrNoSuchEntity(err) {
		t.Fatalf("unexpected error type: %v", err)
	}

}

func (s *S) Test_GetServices(t *C) {
	svcs, err := s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 0)

	svc := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 1)

	svc.ParentServiceID = svc.ID
	svc.ID = "Test_GetHosts2"
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 2)

	svcs, err = s.store.GetServicesByPool(s.ctx, "testPool")
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 2)

	svc.ID = "Test_GetHosts3"
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetChildServices(s.ctx, "svc_test_id")
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 2)

}
