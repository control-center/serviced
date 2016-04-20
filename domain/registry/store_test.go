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

// +build integration

package registry

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
	store ImageRegistryStore
}

func (s *S) SetUpTest(c *C) {
	s.ElasticTest.SetUpTest(c)
	datastore.Register(s.Driver())
	s.ctx = datastore.Get()
	s.store = NewStore()
}

func (s *S) Test_ImageCRUD(c *C) {
	expected := &Image{
		Library: "tenantID",
		Repo:    "reponame",
		Tag:     "latest",
		UUID:    "abs123",
	}
	c.Logf("Key %s: %v", expected.String(), expected.key())
	actual, err := s.store.Get(s.ctx, expected.String())
	c.Assert(err, NotNil)
	c.Assert(actual, IsNil)
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
	err = s.store.Put(s.ctx, expected)
	expected.DatabaseVersion++
	c.Assert(err, IsNil)
	actual, err = s.store.Get(s.ctx, expected.String())
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
	err = s.store.Put(s.ctx, expected)
	expected.DatabaseVersion++
	c.Assert(err, IsNil)
	actual, err = s.store.Get(s.ctx, expected.String())
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
	err = s.store.Delete(s.ctx, expected.String())
	c.Assert(err, IsNil)
	actual, err = s.store.Get(s.ctx, expected.String())
	c.Assert(err, NotNil)
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
}

func (s *S) GetImages(c *C) {
	expected := []Image{}
	actual, err := s.store.GetImages(s.ctx)
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
	expected = []Image{
		{Library: "test", Repo: "busybox", Tag: "latest", UUID: "123abc"},
		{Library: "test", Repo: "centos", Tag: "latest", UUID: "4567dsfdsg"},
		{Library: "test2", Repo: "ogden", Tag: "latest", UUID: "4567dfsdsg"},
		{Library: "test", Repo: "ogden", Tag: "tuesday", UUID: "5654gge"},
	}
	for _, image := range expected {
		err = s.store.Put(s.ctx, &image)
		c.Assert(err, IsNil)
	}
	actual, err = s.store.GetImages(s.ctx)
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
}

func (s *S) Test_SearchLibraryByTag(c *C) {
	expected := []Image{}
	actual, err := s.store.SearchLibraryByTag(s.ctx, "test", "latest")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)

	image := &Image{
		Library: "test",
		Repo:    "busybox",
		Tag:     "latest",
		UUID:    "123abc",
	}
	err = s.store.Put(s.ctx, image)
	image.DatabaseVersion++
	c.Assert(err, IsNil)
	expected = append(expected, *image)
	actual, err = s.store.SearchLibraryByTag(s.ctx, "test", "latest")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)

	image = &Image{
		Library: "test",
		Repo:    "centos",
		Tag:     "latest",
		UUID:    "4567dsfdsg",
	}
	err = s.store.Put(s.ctx, image)
	image.DatabaseVersion++
	c.Assert(err, IsNil)
	expected = append(expected, *image)
	actual, err = s.store.SearchLibraryByTag(s.ctx, "test", "latest")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)

	image = &Image{
		Library: "test2",
		Repo:    "ogden",
		Tag:     "latest",
		UUID:    "4567dsfdsg",
	}
	err = s.store.Put(s.ctx, image)
	image.DatabaseVersion++
	c.Assert(err, IsNil)
	actual, err = s.store.SearchLibraryByTag(s.ctx, "test", "latest")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
	actual, err = s.store.SearchLibraryByTag(s.ctx, "test2", "latest")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, []Image{*image})

	image = &Image{
		Library: "test",
		Repo:    "ogden",
		Tag:     "tuesday",
		UUID:    "5654gge",
	}
	err = s.store.Put(s.ctx, image)
	image.DatabaseVersion++
	actual, err = s.store.SearchLibraryByTag(s.ctx, "test", "latest")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
	actual, err = s.store.SearchLibraryByTag(s.ctx, "test", "tuesday")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, []Image{*image})
}
