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

package servicetemplate

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicedefinition/testutils"
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
	store Store
}

func (s *S) SetUpTest(c *C) {
	s.ElasticTest.SetUpTest(c)
	datastore.Register(s.Driver())
	s.ctx = datastore.Get()
	s.store = NewStore()
}

func (s *S) Test_ServiceTemplateCRUD(t *C) {

	id := "st_test_id"
	st := ServiceTemplate{ID: id}

	_, err := s.store.Get(s.ctx, st.ID)
	t.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)

	err = s.store.Put(s.ctx, st)
	t.Assert(err, NotNil)

	st.Name = testutils.ValidSvcDef.Name
	st.Services = []servicedefinition.ServiceDefinition{*testutils.ValidSvcDef}
	err = s.store.Put(s.ctx, st)
	t.Assert(err, IsNil)

	st2, err := s.store.Get(s.ctx, st.ID)
	t.Assert(err, IsNil)
	t.Assert(st2, NotNil)
	t.Assert(st2.Equals(&st), Equals, true)

	//Test update
	st.Name = "newName"
	st.Services[0].Command = "blam"
	err = s.store.Put(s.ctx, st)

	st2, err = s.store.Get(s.ctx, st.ID)
	t.Assert(err, IsNil)
	t.Assert(st2, NotNil)
	t.Assert(st2.Equals(&st), Equals, true)

}

func (s *S) Test_GetServiceTemplates(t *C) {

	servicetemplates, err := s.store.GetServiceTemplates(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(servicetemplates), Equals, 0)

	st := ServiceTemplate{ID: "st_test_id"}
	st.Name = testutils.ValidSvcDef.Name
	st.Services = []servicedefinition.ServiceDefinition{*testutils.ValidSvcDef}
	err = s.store.Put(s.ctx, st)
	t.Assert(err, IsNil)

	servicetemplates, err = s.store.GetServiceTemplates(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(servicetemplates), Equals, 1)

	st.ID = "st_test_id_2"
	err = s.store.Put(s.ctx, st)
	t.Assert(err, IsNil)

	servicetemplates, err = s.store.GetServiceTemplates(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(servicetemplates), Equals, 2)

}
