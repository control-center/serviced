// Copyright 2016 The Serviced Authors.
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

package dfs_test

import (
	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"
)

var (
	oldImage = registry.Image{Library: "testlib", Repo: "testoldimage", Tag: "testtag"}
	newImage = dockerclient.Image{ID: "newimageid"}
)

func (s *DFSTestSuite) TestOverride_OldImageNotFound(c *C) {
	s.index.On("FindImage", "oldimage").Return(nil, ErrTestImageNotInRegistry)
	err := s.dfs.Override("newimage", "oldimage")
	c.Assert(err, Equals, ErrTestImageNotInRegistry)
}

func (s *DFSTestSuite) TestOverride_NewImageNotFound(c *C) {
	s.index.On("FindImage", "oldimage").Return(&oldImage, nil)
	s.docker.On("FindImage", "newimage").Return(nil, ErrTestImageNotFound)
	err := s.dfs.Override("newimage", "oldimage")
	c.Assert(err, Equals, ErrTestImageNotFound)
}

func (s *DFSTestSuite) TestOverride_ErrOnHash(c *C) {
	s.index.On("FindImage", "oldimage").Return(&oldImage, nil)
	s.docker.On("FindImage", "newimage").Return(&newImage, nil)
	s.docker.On("GetImageHash", newImage.ID).Return("", ErrTestNoHash)
	err := s.dfs.Override("newimage", "oldimage")
	c.Assert(err, Equals, ErrTestNoHash)
}

func (s *DFSTestSuite) TestOverride_ErrOnPush(c *C) {
	s.index.On("FindImage", "oldimage").Return(&oldImage, nil)
	s.docker.On("FindImage", "newimage").Return(&newImage, nil)
	s.docker.On("GetImageHash", newImage.ID).Return("newimagehash", nil)
	s.index.On("PushImage", oldImage.String(), newImage.ID, "newimagehash").Return(ErrTestNoPush)
	err := s.dfs.Override("newimage", "oldimage")
	c.Assert(err, Equals, ErrTestNoPush)
}

func (s *DFSTestSuite) TestOverride_Success(c *C) {
	s.index.On("FindImage", "oldimage").Return(&oldImage, nil)
	s.docker.On("FindImage", "newimage").Return(&newImage, nil)
	s.docker.On("GetImageHash", newImage.ID).Return("newimagehash", nil)
	s.index.On("PushImage", oldImage.String(), newImage.ID, "newimagehash").Return(nil)
	err := s.dfs.Override("newimage", "oldimage")
	c.Assert(err, IsNil)
}
