// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build integration

package facade

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/registry"
	. "gopkg.in/check.v1"
	"sort"
)

func (ft *FacadeIntegrationTest) TestGetRegistryImage(c *C) {
	expected := &registry.Image{
		Library: "library",
		Repo:    "reponame",
		Tag:     "tagname",
		UUID:    "uuidvalue",
	}
	err := ft.Facade.registryStore.Put(ft.CTX, expected)
	c.Assert(err, IsNil)
	actual, err := ft.Facade.GetRegistryImage(ft.CTX, "library/reponame:tagname")
	c.Assert(err, IsNil)
	expected.IfPrimaryTerm = actual.IfPrimaryTerm
	expected.IfSeqNo = actual.IfSeqNo
	c.Assert(actual, DeepEquals, expected)
}

func (ft *FacadeIntegrationTest) TestGetRegistryImage_NotFound(c *C) {
	result, err := ft.Facade.GetRegistryImage(ft.CTX, "someImageID")
	c.Assert(err, NotNil)
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
	c.Assert(result, IsNil)
}

func (ft *FacadeIntegrationTest) TestSetRegistryImage(c *C) {
	expected := &registry.Image{
		Library: "library",
		Repo:    "reponame",
		Tag:     "tagname",
		UUID:    "uuidvalue",
	}
	err := ft.Facade.SetRegistryImage(ft.CTX, expected)
	c.Assert(err, IsNil)
	actual, err := ft.Facade.registryStore.Get(ft.CTX, "library/reponame:tagname")
	c.Assert(err, IsNil)
	expected.IfPrimaryTerm = actual.IfPrimaryTerm
	expected.IfSeqNo = actual.IfSeqNo
	c.Assert(actual, DeepEquals, expected)

	err = ft.Facade.SetRegistryImage(ft.CTX, expected)
	c.Assert(err, IsNil)
	actual, err = ft.Facade.registryStore.Get(ft.CTX, "library/reponame:tagname")
	expected.IfSeqNo = actual.IfSeqNo
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)

	expected = &registry.Image{
		Library: "library",
		Repo:    "reponame",
		Tag:     "tagname",
		UUID:    "anotheruuidvalue",
	}
	err = ft.Facade.SetRegistryImage(ft.CTX, expected)
	c.Assert(err, IsNil)
	actual, err = ft.Facade.registryStore.Get(ft.CTX, "library/reponame:tagname")
	c.Assert(err, IsNil)
	expected.IfSeqNo = actual.IfSeqNo
	expected.IfPrimaryTerm = actual.IfPrimaryTerm
	c.Assert(actual, DeepEquals, expected)

	expected2 := &registry.Image{
		Library: "library",
		Repo:    "reponame",
		Tag:     "anothertagname",
		UUID:    "anotheruuidvalue",
	}
	err = ft.Facade.SetRegistryImage(ft.CTX, expected2)
	c.Assert(err, IsNil)
	actual, err = ft.Facade.registryStore.Get(ft.CTX, "library/reponame:tagname")
	c.Assert(err, IsNil)
	expected2.IfPrimaryTerm = actual.IfPrimaryTerm
	expected2.IfSeqNo = actual.IfSeqNo
	c.Assert(actual, DeepEquals, expected)
	actual, err = ft.Facade.registryStore.Get(ft.CTX, "library/reponame:anothertagname")
	c.Assert(err, IsNil)
	expected2.IfPrimaryTerm = actual.IfPrimaryTerm
	expected2.IfSeqNo = actual.IfSeqNo
	c.Assert(actual, DeepEquals, expected2)
}

func (ft *FacadeIntegrationTest) TestDeleteRegistryImage(c *C) {
	expected := &registry.Image{
		Library: "library",
		Repo:    "reponame",
		Tag:     "tagname",
		UUID:    "uuidvalue",
	}
	err := ft.Facade.registryStore.Put(ft.CTX, expected)
	c.Assert(err, IsNil)
	actual, err := ft.Facade.registryStore.Get(ft.CTX, "library/reponame:tagname")
	c.Assert(err, IsNil)
	expected.IfPrimaryTerm = actual.IfPrimaryTerm
	expected.IfSeqNo = actual.IfSeqNo
	c.Assert(actual, DeepEquals, expected)
	err = ft.Facade.DeleteRegistryImage(ft.CTX, "library/reponame:tagname")
	c.Assert(err, IsNil)
	actual, err = ft.Facade.registryStore.Get(ft.CTX, "library/reponame:tagname")
	c.Assert(err, NotNil)
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
}

func (ft *FacadeIntegrationTest) TestGetRegistryImages(c *C) {
	expected := []registry.Image{
		{
			Library: "library",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		}, {
			Library: "library",
			Repo:    "anotherreponame",
			Tag:     "tagname",
			UUID:    "anotheruuidvalue",
		},
		{
			Library: "library",
			Repo:    "reponame",
			Tag:     "anothertagname",
			UUID:    "uuidvalue",
		},
		{
			Library: "anotherlibrary",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	for i, image := range expected {
		err := ft.Facade.registryStore.Put(ft.CTX, &image)
		c.Assert(err, IsNil)
		expected[i].Type = image.GetType()
	}
	actual, err := ft.Facade.GetRegistryImages(ft.CTX)
	c.Assert(err, IsNil)
	for i, _ := range actual {
		expected[i].IfPrimaryTerm, actual[i].IfPrimaryTerm = 0, 0
		expected[i].IfSeqNo, actual[i].IfSeqNo = 0, 0
	}
	c.Assert(actual, DeepEquals, expected)
}

func (ft *FacadeIntegrationTest) TestSearchRegistryLibraryByTag(c *C) {
	expected1 := []registry.Image{
		{
			Library: "library",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		}, {
			Library: "library",
			Repo:    "anotherreponame",
			Tag:     "tagname",
			UUID:    "anotheruuidvalue",
		},
	}
	for i := range expected1 {
		err := ft.Facade.registryStore.Put(ft.CTX, &expected1[i])
		c.Assert(err, IsNil)
	}
	expected2 := []registry.Image{
		{
			Library: "library",
			Repo:    "reponame",
			Tag:     "anothertagname",
			UUID:    "uuidvalue",
		},
	}
	for i := range expected2 {
		err := ft.Facade.registryStore.Put(ft.CTX, &expected2[i])
		c.Assert(err, IsNil)
	}
	expected3 := []registry.Image{
		{
			Library: "anotherlibrary",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	for i := range expected3 {
		err := ft.Facade.registryStore.Put(ft.CTX, &expected3[i])
		c.Assert(err, IsNil)
	}
	actual, err := ft.Facade.SearchRegistryLibraryByTag(ft.CTX, "library", "tagname")
	c.Assert(err, IsNil)
	sort.Slice(expected1, func(i, j int) bool {
		return expected1[i].Repo < expected1[j].Repo
	})
	sort.Slice(actual, func(i, j int) bool {
		return actual[i].Repo < actual[j].Repo
	})
	for i := range expected1 {
		expected1[i].IfPrimaryTerm = actual[i].IfPrimaryTerm
		expected1[i].IfSeqNo = actual[i].IfSeqNo
	}
	c.Assert(actual, DeepEquals, expected1)
	actual, err = ft.Facade.SearchRegistryLibraryByTag(ft.CTX, "library", "anothertagname")
	c.Assert(err, IsNil)
	expected2[0].IfPrimaryTerm = actual[0].IfPrimaryTerm
	expected2[0].IfSeqNo = actual[0].IfSeqNo
	c.Assert(actual, DeepEquals, expected2)
	actual, err = ft.Facade.SearchRegistryLibraryByTag(ft.CTX, "anotherlibrary", "tagname")
	c.Assert(err, IsNil)
	expected3[0].IfPrimaryTerm = actual[0].IfPrimaryTerm
	expected3[0].IfSeqNo = actual[0].IfSeqNo
	c.Assert(actual, DeepEquals, expected3)
	actual, err = ft.Facade.SearchRegistryLibraryByTag(ft.CTX, "anotherlibrary", "anothertagname")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, []registry.Image{})
}
