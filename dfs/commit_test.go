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
	. "github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestCommit_NotFound(c *C) {
	s.docker.On("FindContainer", "testcontainer").Return(nil, ErrTestContainerNotFound)
	tenantID, err := s.dfs.Commit("testcontainer")
	c.Assert(tenantID, Equals, "")
	c.Assert(err, Equals, ErrTestContainerNotFound)
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
	tenantID, err := s.dfs.Commit("testcontainer")
	c.Assert(tenantID, Equals, "")
	c.Assert(err, Equals, ErrRunningContainer)
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
	s.index.On("FindImage", "localhost:5000/libraryname/reponame:tagname").Return(nil, ErrTestImageNotFound)
	tenantID, err := s.dfs.Commit("testcontainer")
	c.Assert(tenantID, Equals, "")
	c.Assert(err, Equals, ErrTestImageNotFound)
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
		Tag:     "tagname",		// not "latest"
		UUID:    "testimage",
	}
	s.docker.On("FindContainer", "testcontainer").Return(ctr, nil)
	s.index.On("FindImage", "localhost:5000/libraryname/reponame:tagname").Return(rImg, nil)
	tenantID, err := s.dfs.Commit("testcontainer")
	c.Assert(tenantID, Equals, "")
	c.Assert(err, Equals, ErrStaleContainer)
	// if uuids are different and the hashes are different, then the container is out of date relative to the registry
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
		Hash:    "testImage2-hash",
	}
	s.docker.On("FindContainer", "testcontainer2").Return(ctr2, nil)
	s.docker.On("GetImageHash", ctr2.Image).Return("NOT testImage2-hash", nil)
	s.index.On("FindImage", "localhost:5000/libraryname/reponame:latest").Return(rImg2, nil)
	tenantID, err = s.dfs.Commit("testcontainer2")
	c.Assert(tenantID, Equals, "")
	c.Assert(err, Equals, ErrStaleContainer)
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
	s.index.On("FindImage", "localhost:5000/libraryname/reponame:latest").Return(rImg, nil)
	s.docker.On("CommitContainer", "testcontainer", "localhost:5000/libraryname/reponame:latest").Return(nil, ErrTestNoCommit)
	tenantID, err := s.dfs.Commit("testcontainer")
	c.Assert(tenantID, Equals, "")
	c.Assert(err, Equals, ErrTestNoCommit)
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
	s.index.On("FindImage", "localhost:5000/libraryname/reponame:latest").Return(rImg, nil)
	s.docker.On("CommitContainer", "testcontainer", "localhost:5000/libraryname/reponame:latest").Return(img, nil)
	s.docker.On("GetImageHash", img.ID).Return("hashvalue", nil)
	s.index.On("PushImage", "localhost:5000/libraryname/reponame:latest", "testimage2", "hashvalue").Return(ErrTestNoPush)
	s.index.On("PushImage", "libraryname/reponame:latest", "testimage2", "hashvalue").Return(ErrTestNoPush)
	tenantID, err := s.dfs.Commit("testcontainer")
	c.Assert(tenantID, Equals, "")
	c.Assert(err, Equals, ErrTestNoPush)
}

func (s *DFSTestSuite) TestCommit_NoHash(c *C) {
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
	s.index.On("FindImage", "localhost:5000/libraryname/reponame:latest").Return(rImg, nil)
	s.docker.On("CommitContainer", "testcontainer", "localhost:5000/libraryname/reponame:latest").Return(img, nil)
	s.docker.On("GetImageHash", img.ID).Return("", ErrTestNoHash)
	tenantID, err := s.dfs.Commit("testcontainer")
	c.Assert(tenantID, Equals, "")
	c.Assert(err, Equals, ErrTestNoHash)
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
	s.index.On("FindImage", "localhost:5000/libraryname/reponame:latest").Return(rImg, nil)
	s.docker.On("CommitContainer", "testcontainer", "localhost:5000/libraryname/reponame:latest").Return(img, nil)
	s.docker.On("GetImageHash", img.ID).Return("hashvalue", nil)
	s.index.On("PushImage", "localhost:5000/libraryname/reponame:latest", "testimage2", "hashvalue").Return(nil)
	s.index.On("PushImage", "libraryname/reponame:latest", "testimage2", "hashvalue").Return(nil)
	tenantID, err := s.dfs.Commit("testcontainer")
	c.Assert(tenantID, Equals, "libraryname")
	c.Assert(err, IsNil)
}
