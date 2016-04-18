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

package service

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	. "gopkg.in/check.v1"

	"testing"
	"time"

	"github.com/control-center/serviced/domain/servicedefinition"
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
	store Store
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

func (s *S) Test_FindChildService(t *C) {
	svcIn := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto", ParentServiceID: "parent_svc_id", DeploymentID: "deployment_id"}
	err := s.store.Put(s.ctx, svcIn)
	t.Assert(err, IsNil)

	svcOut, err := s.store.FindChildService(s.ctx, "deployment_id", "parent_svc_id", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, NotNil)

	svcOut, err = s.store.FindChildService(s.ctx, "not_deployment", "parent_svc_id", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	svcOut, err = s.store.FindChildService(s.ctx, "deployment_id", "parent_svc_id", "not_svc")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	svcOut, err = s.store.FindChildService(s.ctx, "deployment_id", "not_parent", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	svcOut, err = s.store.FindChildService(s.ctx, "deployment_id", "not_parent", "not_svc")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)
}

func (s *S) Test_FindTenantByDeploymentID(t *C) {
	svcIn := &Service{ID: "svc_test_id1", PoolID: "testPool", Name: "svc_name", Launch: "auto", ParentServiceID: "parent_svc_id", DeploymentID: "deployment"}
	err := s.store.Put(s.ctx, svcIn)
	t.Assert(err, IsNil)

	// Case 1: no service exists with deployment ID
	svcOut, err := s.store.FindTenantByDeploymentID(s.ctx, "dummy_deployment", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	// Case 2: service exists with deployment ID, but is not tenant
	svcOut, err = s.store.FindTenantByDeploymentID(s.ctx, "deployment", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	svcIn = &Service{ID: "svc_test_id2", PoolID: "testPool", Name: "svc_name", Launch: "auto", ParentServiceID: "", DeploymentID: "deployment2"}
	err = s.store.Put(s.ctx, svcIn)
	t.Assert(err, IsNil)

	// Case 3: service is tenant, but does not have deployment ID
	svcOut, err = s.store.FindTenantByDeploymentID(s.ctx, "deployment", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	// Case 4: success
	svcOut, err = s.store.FindTenantByDeploymentID(s.ctx, "deployment2", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut.ID, Equals, svcIn.ID)
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

func (s *S) Test_GetUpdatedServices(t *C) {
	svcs, err := s.store.GetUpdatedServices(s.ctx, time.Duration(1)*time.Hour)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 0)

	svc := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto", UpdatedAt: time.Now().Add(-time.Duration(10) * time.Second)}
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetUpdatedServices(s.ctx, time.Duration(1)*time.Hour)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 1)
}

func (s *S) Test_VersionConflicts(t *C) {
	svc := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}
	err := s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err := s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 1)
	t.Assert(svcs[0].DatabaseVersion, Equals, 1)

	svc2 := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}
	svc2.DatabaseVersion = 1
	err = s.store.Put(s.ctx, svc2)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 1)
	t.Assert(svcs[0].DatabaseVersion, Equals, 2)

	svc3 := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}
	svc3.DatabaseVersion = 1
	err = s.store.Put(s.ctx, svc3)
	t.Assert(err, Not(IsNil))
}
