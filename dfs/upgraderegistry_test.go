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
	index "github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	dockerclient "github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"
)

// error when trying to find image
func (s *DFSTestSuite) TestUpgradeRegistry_FindImageFail(c *C) {
	imageName := "unknown/repo"
	svcs := []service.Service{
		service.Service{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", "unknown/repo").Return(nil, ErrTestGeneric)
	err := s.dfs.UpgradeRegistry(svcs, "test_tenantID", "", false)
	c.Assert(err, Equals, ErrTestGeneric)
}

// image already in registry
func (s *DFSTestSuite) TestUpgradeRegistry_FindImageSuccess(c *C) {
	imageName := "localhost:5000/goodtenant/repo"
	svcs := []service.Service{
		service.Service{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	rImage := &registry.Image{
		Library: "goodtenant",
		Repo:    "repo",
		Tag:     "latest",
		UUID:    "testuuid",
	}
	s.index.On("FindImage", imageName).Return(rImage, nil)
	err := s.dfs.UpgradeRegistry(svcs, "goodtenant", "", false)
	c.Assert(err, IsNil)
}

// image in registry and override enabled
func (s *DFSTestSuite) TestUpgradeRegistry_OverrideImageFound(c *C) {
	imageName := "localhost:5000/tenantid/reponame"
	svcs := []service.Service{
		{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	rImage := &registry.Image{
		Library: "tenantid",
		Repo:    "reponame",
		Tag:     "latest",
		UUID:    "uuidvalue",
	}
	s.index.On("FindImage", imageName).Return(rImage, nil)
	image := &dockerclient.Image{ID: "xyzabc123"}
	s.docker.On("GetImageHash", image.ID).Return("hashvalue", nil)
	s.docker.On("FindImage", imageName).Return(image, nil)
	s.index.On("PushImage", "tenantid/reponame:latest", "xyzabc123", "hashvalue").Return(nil)
	err := s.dfs.UpgradeRegistry(svcs, "tenantid", "", true)
	c.Assert(err, IsNil)
}

// image not in registry and override enabled
func (s *DFSTestSuite) TestUpgradeRegistry_OverrideImageNotFound(c *C) {
	imageName := "localhost:5000/tenantid/reponame"
	svcs := []service.Service{
		{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound)
	image := &dockerclient.Image{ID: "xyzabc123"}
	s.docker.On("GetImageHash", image.ID).Return("hashvalue", nil)
	s.docker.On("FindImage", imageName).Return(image, nil)
	s.index.On("PushImage", "tenantid/reponame:latest", "xyzabc123", "hashvalue").Return(nil)
	err := s.dfs.UpgradeRegistry(svcs, "tenantid", "", true)
	c.Assert(err, IsNil)
}

// image not present in docker library
func (s *DFSTestSuite) TestUpgradeRegistry_DockerImageNotFound(c *C) {
	imageName := "localhost:5000/goodtenant/repo"
	svcs := []service.Service{
		service.Service{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound)
	s.docker.On("FindImage", imageName).Return(nil, dockerclient.ErrNoSuchImage)
	err := s.dfs.UpgradeRegistry(svcs, "goodtenant", "", false)
	c.Assert(err, IsNil)
}

// error when looking for image in docker library
func (s *DFSTestSuite) TestUpgradeRegistry_DockerFindImageFail(c *C) {
	imageName := "localhost:5000/goodtenant/repo"
	svcs := []service.Service{
		service.Service{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound)
	s.docker.On("FindImage", imageName).Return(nil, ErrTestGeneric)
	err := s.dfs.UpgradeRegistry(svcs, "goodtenant", "", false)
	c.Assert(err, Equals, ErrTestGeneric)
}

// error when getting the image hash
func (s *DFSTestSuite) TestUpgradeRegistry_PushImageNoHash(c *C) {
	imageName := "localhost:5000/goodtenant/repo"
	svcs := []service.Service{
		service.Service{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound)
	image := &dockerclient.Image{ID: "youyoueyedee"}
	s.docker.On("FindImage", imageName).Return(image, nil)
	s.docker.On("GetImageHash", image.ID).Return("", ErrTestNoHash)
	err := s.dfs.UpgradeRegistry(svcs, "goodtenant", "", false)
	c.Assert(err, Equals, ErrTestNoHash)
}

// image successfully pushed to registry index
func (s *DFSTestSuite) TestUpgradeRegistry_PushImageSuccess(c *C) {
	imageName := "localhost:5000/goodtenant/repo"
	svcs := []service.Service{
		service.Service{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound)
	image := &dockerclient.Image{ID: "youyoueyedee"}
	s.docker.On("FindImage", imageName).Return(image, nil)
	s.docker.On("GetImageHash", image.ID).Return("hashvalue", nil)
	s.index.On("PushImage", "goodtenant/repo:latest", "youyoueyedee", "hashvalue").Return(nil)
	err := s.dfs.UpgradeRegistry(svcs, "goodtenant", "", false)
	c.Assert(err, IsNil)
}

// error pushing image to registry index
func (s *DFSTestSuite) TestUpgradeRegistry_PushImageFail(c *C) {
	imageName := "localhost:5000/goodtenant/repo"
	svcs := []service.Service{
		service.Service{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound)
	image := &dockerclient.Image{ID: "youyoueyedee"}
	s.docker.On("FindImage", imageName).Return(image, nil)
	s.docker.On("GetImageHash", image.ID).Return("hashvalue", nil)
	s.index.On("PushImage", "goodtenant/repo:latest", "youyoueyedee", "hashvalue").Return(ErrTestNoPush)
	err := s.dfs.UpgradeRegistry(svcs, "goodtenant", "", false)
	c.Assert(err, Equals, ErrTestNoPush)
}

// duplicate images are only migrated once
func (s *DFSTestSuite) TestUpgradeRegistry_NoDupes(c *C) {
	imageName := "localhost:5000/goodtenant/repo"
	svcs := []service.Service{
		{
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		}, {
			Name:    "service",
			ID:      "service_id",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound).Once()
	image := &dockerclient.Image{ID: "youyoueyedee"}
	s.docker.On("FindImage", imageName).Return(image, nil).Once()
	s.docker.On("GetImageHash", image.ID).Return("hashvalue", nil)
	s.index.On("PushImage", "goodtenant/repo:latest", "youyoueyedee", "hashvalue").Return(nil).Once()
	err := s.dfs.UpgradeRegistry(svcs, "goodtenant", "", false)
	c.Assert(err, IsNil)
}

// no image in the old docker registry
func (s *DFSTestSuite) TestUpgradeRegistry_MigrateNoImage(c *C) {
	imageName := "test-server:5000/tenantid/reponame"
	svcs := []service.Service{
		{
			Name:    "servicename",
			ID:      "serviceid",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound)
	s.docker.On("PullImage", "old-server:5001/tenantid/reponame:latest").Return(dockerclient.ErrNoSuchImage)
	image := &dockerclient.Image{ID: "uuidvalue"}
	s.docker.On("FindImage", imageName).Return(image, nil)
	s.docker.On("GetImageHash", image.ID).Return("hashvalue", nil)
	s.index.On("PushImage", "tenantid/reponame:latest", "uuidvalue", "hashvalue").Return(nil)
	err := s.dfs.UpgradeRegistry(svcs, "tenantid", "old-server:5001", false)
	c.Assert(err, IsNil)
}

// could not retag image
func (s *DFSTestSuite) TestUpgradeRegistry_MigrateTagFail(c *C) {
	imageName := "test-server:5000/tenantid/reponame"
	svcs := []service.Service{
		{
			Name:    "servicename",
			ID:      "serviceid",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound)
	s.docker.On("PullImage", "old-server:5001/tenantid/reponame:latest").Return(nil)
	s.docker.On("TagImage", "old-server:5001/tenantid/reponame:latest", "test-server:5000/tenantid/reponame").Return(ErrTestNoTag)
	err := s.dfs.UpgradeRegistry(svcs, "tenantid", "old-server:5001", false)
	c.Assert(err, Equals, ErrTestNoTag)
}

// migrate successful
func (s *DFSTestSuite) TestUpgradeRegistry_MigrateSuccess(c *C) {
	imageName := "test-server:5000/tenantid/reponame"
	svcs := []service.Service{
		{
			Name:    "servicename",
			ID:      "serviceid",
			ImageID: imageName,
		},
	}
	s.index.On("FindImage", imageName).Return(nil, index.ErrImageNotFound)
	s.docker.On("PullImage", "old-server:5001/tenantid/reponame:latest").Return(nil)
	s.docker.On("TagImage", "old-server:5001/tenantid/reponame:latest", "test-server:5000/tenantid/reponame").Return(nil)
	image := &dockerclient.Image{ID: "uuidvalue"}
	s.docker.On("FindImage", imageName).Return(image, nil)
	s.docker.On("GetImageHash", image.ID).Return("hashvalue", nil)
	s.index.On("PushImage", "tenantid/reponame:latest", "uuidvalue", "hashvalue").Return(nil)
	err := s.dfs.UpgradeRegistry(svcs, "tenantid", "old-server:5001", false)
	c.Assert(err, IsNil)
}
