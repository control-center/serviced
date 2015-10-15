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

// +build unit

package dfs_test

import (
	"github.com/control-center/serviced/datastore"
	. "github.com/control-center/serviced/dfsnew"
	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestDownload_NoImage(c *C) {
	s.docker.On("FindImage", "library/repo:tag").Return(nil, ErrTestImageNotFound)
	img, err := s.dfs.Download("library/repo:tag", "tenant", false)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, ErrTestImageNotFound)
	img, err = s.dfs.Download("library/repo:tag", "tenant", true)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, ErrTestImageNotFound)
	s.docker.On("FindImage", "library/repo2:tag").Return(nil, dockerclient.ErrNoSuchImage)
	s.docker.On("PullImage", "library/repo2:tag").Return(ErrTestNoPull)
	img, err = s.dfs.Download("library/repo2:tag", "tenant", false)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, ErrTestNoPull)
	s.docker.On("FindImage", "library/repo3:tag").Return(nil, dockerclient.ErrNoSuchImage).Once()
	s.docker.On("PullImage", "library/repo3:tag").Return(nil)
	s.docker.On("FindImage", "library/repo3:tag").Return(nil, ErrTestImageNotFound)
	img, err = s.dfs.Download("library/repo3:tag", "tenant", false)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, ErrTestImageNotFound)
	s.docker.On("FindImage", "library/repo4:tag").Return(nil, dockerclient.ErrNoSuchImage).Once()
	s.docker.On("PullImage", "library/repo4:tag").Return(nil)
	image := &dockerclient.Image{ID: "testimage"}
	s.docker.On("FindImage", "library/repo4:tag").Return(image, nil)
	s.index.On("FindImage", "tenant/repo4:latest").Return(nil, ErrTestImageNotInRegistry)
	img, err = s.dfs.Download("library/repo4:tag", "tenant", true)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, ErrTestImageNotInRegistry)
	s.docker.AssertExpectations(c)
}

func (s *DFSTestSuite) TestDownload_Upgrade(c *C) {
	image := &dockerclient.Image{ID: "testimage1"}
	s.docker.On("FindImage", "library/repo:tag").Return(image, nil)
	rImage := &registry.Image{
		Library: "tenant",
		Repo:    "repo",
		Tag:     "latest",
		UUID:    "testimage2",
	}
	s.index.On("FindImage", "tenant/repo:latest").Return(rImage, nil)
	s.index.On("PushImage", "tenant/repo:latest", "testimage1").Return(ErrTestImageNotInRegistry).Once()
	img, err := s.dfs.Download("library/repo:tag", "tenant", true)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, ErrTestImageNotInRegistry)
	s.index.On("PushImage", "tenant/repo:latest", "testimage1").Return(nil).Once()
	img, err = s.dfs.Download("library/repo:tag", "tenant", true)
	c.Assert(img, Equals, "tenant/repo:latest")
	c.Assert(err, IsNil)
}

func (s *DFSTestSuite) TestDownload_NoUpgrade(c *C) {
	image := &dockerclient.Image{ID: "testimage1"}
	s.docker.On("FindImage", "library/repo:tag").Return(image, nil)
	rImage := &registry.Image{
		Library: "tenant",
		Repo:    "repo",
		Tag:     "latest",
		UUID:    "testimage2",
	}
	s.index.On("FindImage", "tenant/repo:latest").Return(rImage, nil)
	img, err := s.dfs.Download("library/repo:tag", "tenant", false)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, ErrImageCollision)
	image = &dockerclient.Image{ID: "testimage2"}
	s.docker.On("FindImage", "library/repo2:tag").Return(image, nil)
	rImage = &registry.Image{
		Library: "tenant",
		Repo:    "repo2",
		Tag:     "latest",
		UUID:    "testimage2",
	}
	s.index.On("FindImage", "tenant/repo2:latest").Return(rImage, nil)
	s.index.On("PushImage", "tenant/repo2:latest", "testimage2").Return(nil)
	img, err = s.dfs.Download("library/repo2:tag", "tenant", false)
	c.Assert(img, Equals, "tenant/repo2:latest")
	c.Assert(err, IsNil)
	image = &dockerclient.Image{ID: "testimage3"}
	s.docker.On("FindImage", "library/repo3:tag").Return(image, nil)
	expectedErr := datastore.ErrNoSuchEntity{registry.Key("tenant/repo3:latest")}
	c.Assert(datastore.IsErrNoSuchEntity(expectedErr), Equals, true)
	s.index.On("FindImage", "tenant/repo3:latest").Return(nil, expectedErr)
	s.index.On("PushImage", "tenant/repo3:latest", "testimage3").Return(nil)
	img, err = s.dfs.Download("library/repo3:tag", "tenant", false)
	c.Assert(img, Equals, "tenant/repo3:latest")
	c.Assert(err, IsNil)
}
