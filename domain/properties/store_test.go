// Copyright 2016 The Serviced Authors.
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

package properties

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

func (s *S) Test_GetEmpty(c *C) {
	actual, err := s.store.Get(s.ctx)
	c.Assert(err, NotNil)
	c.Assert(actual, IsNil)
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
}
func (s *S) Test_Put(c *C) {
	props := New()
	props.SetCCVersion("9.9.8")
	err := s.store.Put(s.ctx, props)
	c.Assert(err, IsNil)

	actual, err := s.store.Get(s.ctx)
	c.Assert(err, IsNil)
	// update version since it is incremented when stored
	props.DatabaseVersion = 1
	props.IfPrimaryTerm = 1
	c.Assert(actual, DeepEquals, props)

	props.SetCCVersion("9.9.9")
	props.Props["blam"] = "test"
	err = s.store.Put(s.ctx, props)
	c.Assert(err, IsNil)
	actual, err = s.store.Get(s.ctx)
	c.Assert(err, IsNil)
	props.DatabaseVersion = 2
	props.IfSeqNo = 1
	c.Assert(actual, DeepEquals, props)

}
