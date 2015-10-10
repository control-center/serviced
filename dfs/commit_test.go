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

package dfs_test

import (
	"errors"

	"github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestCommit_NotFound(c *C) {
	s.docker.On("FindContainer", "testcontainer").Return(nil, errors.New("ctr not found"))
	err := s.dfs.Commit("testcontainer")
	c.Assert(err, Equals, errors.New("ctr not found"))
}

func (s *DFSTestSuite) TestCommit_Running(c *C) {
	ctr := &dockerclient.Container{
		ID:    "testcontainer",
		Image: "testimage",
		Config: &dockerclient.Config{
			Image: "localhost:5000/libraryname/reponame:tagname",
		},
		State: dockerclient.State{
			Running: true,
		},
	}
	s.docker.On("FindContainer", "testcontainer").Return(ctr, nil)
	err := s.dfs.Commit("testcontainer")
	c.Assert(err, Equals, dfs.ErrRunningContainer)
}

func (s *DFSTestSuite) TestCommit_ImageNotFound(c *C) {
	ctr := &dockerclient.Container{
		ID:    "testcontainer",
		Image: "testimage",
		Config: &dockerclient.Config{
			Image: "localhost:5000/libraryname/reponame:tagname",
		},
		State: dockerclient.State{
			Running: false,
		},
	}
	s.docker.On("FindContainer", "testcontainer").Return(ctr, nil)
	s.registry.On("GetImage", "localhost:5000/libraryname/reponame:tagname").Return(nil, errors.New("img not found"))
	err := s.dfs.Commit("testcontainer")
	c.Assert(err, Equals, errors.New("image not found"))
}

func (s *DFSTestSuite) TestCommit_Stale(c *C) {
	// tag is not latest
	ctr := &dockerclient.Container{
		ID:    "testcontainer",
		Image: "testimage",
		Config: &dockerclient.Config{
			Image: "localhost:5000/libraryname/reponame:tagname",
		},
		State: dockerclient.State{
			Running: false,
		},
	}
	rImg := &registry.Image{
		Library: "libraryname",
		Repo:    "reponame",
		Tag:     "tagname",
		UUID:    "testimage",
	}
	s.docker.On("FindContainer", "testcontainer").Return(ctr, nil)
	s.registry.On("GetImage", "localhost:5000/libraryname/reponame:tagname").Return(nil, rImg)
	err := s.dfs.Commit("testcontainer")
	c.Assert(err, Equals, dfs.ErrStaleContainer)
	// uuid is outdated
	ctr2 := &dockerclient.Container{
		ID:    "testcontainer2",
		Image: "testimage",
		Config: &dockerclient.Config{
			Image: "localhost:5000/libraryname/reponame:latest",
		},
		State: dockerclient.State{
			Running: false,
		},
	}
	rImg2 := &registry.Image{
		Library: "libraryname",
		Repo:    "reponame",
		Tag:     "latest",
		UUID:    "testimage2",
	}
	s.docker.On("FindContainer", "testcontainer2").Return(ctr2, nil)
	s.registry.On("GetImage", "localhost:5000/libraryname/reponame:latest").Return(nil, rImg2)
	err = s.dfs.Commit("testcontainer2")
	c.Assert(err, Equals, dfs.ErrStaleContainer)
}

func (s *DFSTestSuite) TestCommit_NoCommit(c *C) {
	ctr := &dockerclient.Container{
		ID:    "testcontainer",
		Image: "testimage",
		Config: &dockerclient.Config{
			Image: "localhost:5000/libraryname/reponame:latest",
		},
		State: dockerclient.State{
			Running: false,
		},
	}
	rImg := &registry.Image{
		Library: "libraryname",
		Repo:    "reponame",
		Tag:     "latest",
		UUID:    "testimage",
	}
	s.docker.On("FindContainer", "testcontainer").Return(ctr, nil)
	s.registry.On("GetImage", "localhost:5000/libraryname/reponame:latest").Return(nil, rImg)
	s.docker.On("CommitContainer", "testcontainer", "localhost:5000/libraryname/reponame:latest").Return(nil, errors.New("no commit"))
	err := s.dfs.Commit("testcontainer")
	c.Assert(err, Equals, errors.New("no commit"))
}

func (s *DFSTestSuite) TestCommit_NoPush(c *C) {
	ctr := &dockerclient.Container{
		ID:    "testcontainer",
		Image: "testimage",
		Config: &dockerclient.Config{
			Image: "localhost:5000/libraryname/reponame:latest",
		},
		State: dockerclient.State{
			Running: false,
		},
	}
	rImg := &registry.Image{
		Library: "libraryname",
		Repo:    "reponame",
		Tag:     "latest",
		UUID:    "testimage",
	}
	img := &dockerclient.Image{
		ID: "testimage2",
	}
	s.docker.On("FindContainer", "testcontainer").Return(ctr, nil)
	s.registry.On("GetImage", "localhost:5000/libraryname/reponame:latest").Return(nil, rImg)
	s.docker.On("CommitContainer", "testcontainer", "localhost:5000/libraryname/reponame:latest").Return(img, nil)
	s.docker.On("PushImage", "localhost:5000/libraryname/reponame:latest", "testimage2").Return(errors.New("no push"))
	err := s.dfs.Commit("testcontainer")
	c.Assert(err, Equals, errors.New("no push"))
}

func (s *DFSTestSuite) TestCommit_Success(c *C) {
	ctr := &dockerclient.Container{
		ID:    "testcontainer",
		Image: "testimage",
		Config: &dockerclient.Config{
			Image: "localhost:5000/libraryname/reponame:latest",
		},
		State: dockerclient.State{
			Running: false,
		},
	}
	rImg := &registry.Image{
		Library: "libraryname",
		Repo:    "reponame",
		Tag:     "latest",
		UUID:    "testimage",
	}
	img := &dockerclient.Image{
		ID: "testimage2",
	}
	s.docker.On("FindContainer", "testcontainer").Return(ctr, nil)
	s.registry.On("GetImage", "localhost:5000/libraryname/reponame:latest").Return(nil, rImg)
	s.docker.On("CommitContainer", "testcontainer", "localhost:5000/libraryname/reponame:latest").Return(img, nil)
	s.docker.On("PushImage", "localhost:5000/libraryname/reponame:latest", "testimage2").Return(nil)
	err := s.dfs.Commit("testcontainer")
	c.Assert(err, Equals, nil)
}
