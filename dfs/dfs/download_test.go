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
	"errors"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs/dfs"
	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestDownload_NoImage(c *C) {
	s.docker.On("FindImage", "library/repo:tag").Return(nil, errors.New("image not found"))
	img, err := s.dfs.Download("library/repo:tag", "tenant", false)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, errors.New("image not found"))
	img, err = s.dfs.Download("library/repo:tag", "tenant", true)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, errors.New("image not found"))
	s.docker.On("FindImage", "library/repo2:tag").Return(nil, dockerclient.ErrNoSuchImage)
	s.docker.On("PullImage", "library/repo2:tag").Return(errors.New("could not pull image"))
	img, err = s.dfs.Download("library/repo2:tag", "tenant", false)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, errors.New("could not pull image"))
	s.docker.On("FindImage", "library/repo3:tag").Return(nil, dockerclient.ErrNoSuchImage).Once()
	s.docker.On("PullImage", "library/repo3:tag").Return(nil)
	s.docker.On("FindImage", "library/repo3:tag").Return(nil, errors.New("image not found"))
	img, err = s.dfs.Download("library/repo3:tag", "tenant", false)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, errors.New("image not found"))
	s.docker.On("FindImage", "library/repo4:tag").Return(nil, dockerclient.ErrNoSuchImage).Once()
	s.docker.On("PullImage", "library/repo4:tag").Return(nil)
	image := &dockerclient.Image{ID: "testimage"}
	s.docker.On("FindImage", "library/repo4:tag").Return(image, nil)
	s.index.On("FindImage", "tenant/repo4:latest").Return(nil, errors.New("image not in registry"))
	img, err = s.dfs.Download("library/repo4:tag", "tenant", true)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, errors.New("image not in registry"))
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
	s.index.On("PushImage", "tenant/repo:latest").Return(errors.New("could not push image into registry")).Once()
	img, err := s.dfs.Download("library/repo:tag", "tenant", true)
	c.Assert(img, Equals, "")
	c.Assert(err, Equals, errors.New("could not push image into registry"))
	s.index.On("PushImage", "tenant/repo:latest").Return(nil).Once()
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
	c.Assert(err, Equals, dfs.ErrImageCollision)
	image = &dockerclient.Image{ID: "testimage2"}
	s.docker.On("FindImage", "library/repo2:tag").Return(image, nil)
	rImage = &registry.Image{
		Library: "tenant",
		Repo:    "repo2",
		Tag:     "latest",
		UUID:    "testimage2",
	}
	s.index.On("FindImage", "tenant/repo2:latest").Return(rImage, nil)
	s.index.On("PushImage", "tenant/repo2:latest").Return(nil)
	img, err = s.dfs.Download("library/repo2:tag", "tenant", false)
	c.Assert(img, Equals, "tenant/repo2:latest")
	c.Assert(err, IsNil)
	image = &dockerclient.Image{ID: "testimage3"}
	s.docker.On("FindImage", "library/repo3:tag").Return(image, nil)
	s.index.On("FindImage", "tenant/repo3:latest").Return(nil, &datastore.ErrNoSuchEntity{})
	s.index.On("PushImage", "tenant/repo3:latest").Return(nil)
	img, err = s.dfs.Download("library/repo3:tag", "tenant", false)
	c.Assert(img, Equals, "tenant/repo3:latest")
	c.Assert(err, IsNil)
}
