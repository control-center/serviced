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

// +build unit

package registry

import (
	"errors"
	"testing"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs/registry/mocks"
	"github.com/control-center/serviced/domain/registry"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

var (
	ErrTestUnknownError   = errors.New("test unknown error")
	ErrTestGeneratingHash = errors.New("test error generating hash")
)

func TestRegistryIndex(t *testing.T) { TestingT(t) }

type RegistryIndexSuite struct {
	ctx    datastore.Context
	facade *mocks.MockFacade
	index  *RegistryIndexClient
}

var _ = Suite(&RegistryIndexSuite{})

func (s *RegistryIndexSuite) SetUpTest(c *C) {
	s.ctx = datastore.Get()
	s.facade = &mocks.MockFacade{}
	s.index = &RegistryIndexClient{s.ctx, s.facade}
}

func (s *RegistryIndexSuite) TestFindImage(c *C) {
	// unknown error
	s.facade.On("GetRegistryImage", s.ctx, "unknownlibrary/repo:tagname").Return(nil, ErrTestUnknownError)
	actual, err := s.index.FindImage("test-host:5000/unknownlibrary/repo:tagname")
	c.Assert(actual, IsNil)
	c.Assert(err, Equals, ErrTestUnknownError)
	// image not in registry
	s.facade.On("GetRegistryImage", s.ctx, "libnotfound/repo:latest").Return(nil, datastore.ErrNoSuchEntity{registry.Key("libnotfound/repo:latest")})
	actual, err = s.index.FindImage("libnotfound/repo")
	c.Assert(actual, IsNil)
	c.Assert(err, Equals, ErrImageNotFound)
	// success
	expected := &registry.Image{
		Library: "library",
		Repo:    "repo",
		Tag:     "latest",
		UUID:    "uuidvalue",
	}
	s.facade.On("GetRegistryImage", s.ctx, "library/repo:latest").Return(expected, nil)
	actual, err = s.index.FindImage("test-host:5000/library/repo:")
	c.Assert(actual, DeepEquals, expected)
	c.Assert(err, IsNil)
}

func (s *RegistryIndexSuite) TestPushImage(c *C) {
	hashValue := "123456789abcdef"
	s.facade.On("SetRegistryImage", s.ctx, mock.AnythingOfType("*registry.Image")).Return(nil)
	expected := &registry.Image{
		Library: "libraryname",
		Repo:    "reponame",
		Tag:     "tagname",
		UUID:    "uuidvalue",
		Hash:    hashValue,
	}
	err := s.index.PushImage("localhost:5000/libraryname/reponame:tagname", "uuidvalue", hashValue)
	c.Assert(err, IsNil)
	s.facade.AssertCalled(c, "SetRegistryImage", s.ctx, expected)
}

func (s *RegistryIndexSuite) TestRemoveImage(c *C) {
	s.facade.On("DeleteRegistryImage", s.ctx, "libraryname/reponame:tagname").Return(nil).Twice()
	err := s.index.RemoveImage("libraryname/reponame:tagname")
	c.Assert(err, IsNil)
	err = s.index.RemoveImage("localhost:5000/libraryname/reponame:tagname")
	c.Assert(err, IsNil)
	s.facade.AssertExpectations(c)
}

func (s *RegistryIndexSuite) TestSearchLibraryByTag(c *C) {
	expected := []registry.Image{
		{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "latest",
			UUID:    "uuidvalue",
		},
	}
	s.facade.On("SearchRegistryLibraryByTag", s.ctx, "libraryname", "latest").Return(expected, nil).Once()
	actual, err := s.index.SearchLibraryByTag("libraryname", "latest")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
	s.facade.AssertExpectations(c)
}
