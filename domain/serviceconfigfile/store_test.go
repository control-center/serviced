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

package serviceconfigfile

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/domain/servicedefinition"
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

func (s *S) Test_ConfigFileCRUD(t *C) {
	configFile, err := New("tenant_id", "/testpath", servicedefinition.ConfigFile{Content: "Test content", Filename: "testname"})
	t.Assert(err, IsNil)
	configFile2 := SvcConfigFile{}

	if err := s.ps.Get(s.ctx, Key(configFile.ID), &configFile2); !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Expected ErrNoSuchEntity, got: %v", err)
	}

	err = s.ps.Put(s.ctx, Key(configFile.ID), configFile)
	if err != nil {
		t.Errorf("Unexpected failure creating configFile %-v", configFile)
	}

	//Test update
	configFile.ConfFile.Owner = "newowner"
	err = s.ps.Put(s.ctx, Key(configFile.ID), configFile)
	err = s.ps.Get(s.ctx, Key(configFile.ID), &configFile2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if configFile.ConfFile.Owner != configFile2.ConfFile.Owner {
		t.Errorf("configFiles did not match after update")
	}

	//test delete
	err = s.ps.Delete(s.ctx, Key(configFile.ID))
	err = s.ps.Get(s.ctx, Key(configFile.ID), &configFile2)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}

}

func (s *S) Test_GetConfigFiles(t *C) {

	tenant := "test_tenant"
	path := "/testPath/parts"

	configFiles, err := s.ps.GetConfigFiles(s.ctx, tenant, path)
	t.Assert(err, IsNil)
	t.Assert(0, Equals, len(configFiles))

	configFile, err := New(tenant, path, servicedefinition.ConfigFile{Content: "Test content", Filename: "testname"})
	t.Assert(err, IsNil)
	err = s.ps.Put(s.ctx, Key(configFile.ID), configFile)
	t.Assert(err, IsNil)

	configFiles, err = s.ps.GetConfigFiles(s.ctx, "wrong_tenant", path)
	t.Assert(err, IsNil)
	t.Assert(0, Equals, len(configFiles))

	configFiles, err = s.ps.GetConfigFiles(s.ctx, tenant, path)
	t.Assert(err, IsNil)
	t.Assert(1, Equals, len(configFiles))
	// Need to bump the DB version to make this pass the test
	configFile.DatabaseVersion = 1
	t.Assert(*configFile, Equals, *configFiles[0])

	//	configFile.ID = "Test_GetHosts2"
	//	err = s.ps.Put(s.ctx, Key(configFile.ID), configFile)
	//	if err != nil {
	//		t.Errorf("Unexpected error: %v", err)
	//	}
	//
	//	configFiles, err = s.ps.GetConfigFiles(s.ctx)
	//	if err != nil {
	//		t.Errorf("Unexpected error: %v", err)
	//	} else if len(configFiles) != 2 {
	//		t.Errorf("Expected %v results, got %v :%v", 2, len(configFiles), configFiles)
	//	}

}
