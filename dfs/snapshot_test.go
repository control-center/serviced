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
	"bytes"
	"encoding/json"
	"strings"
	"time"

	. "github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	volumemocks "github.com/control-center/serviced/volume/mocks"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestSnapshot_VolumeNotFound(c *C) {
	data := SnapshotInfo{
		SnapshotInfo: &volume.SnapshotInfo{
			TenantID: "BASE",
			Tags:     []string{"tag1"},
		},
	}
	s.disk.On("Get", "BASE").Return(&volumemocks.Volume{}, ErrTestVolumeNotFound)
	id, err := s.dfs.Snapshot(data)
	c.Assert(id, Equals, "")
	c.Assert(err, Equals, ErrTestVolumeNotFound)
}

func (s *DFSTestSuite) TestSnapshot_NoPush(c *C) {
	data := SnapshotInfo{
		SnapshotInfo: &volume.SnapshotInfo{
			TenantID: "BASE",
			Tags:     []string{"tag1"},
		},
		Images: []string{"BASE/repo:latest"},
	}
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "BASE").Return(vol, nil)
	s.index.On("FindImage", "BASE/repo:latest").Return(&registry.Image{}, ErrTestImageNotInRegistry).Once()
	id, err := s.dfs.Snapshot(data)
	c.Assert(id, Equals, "")
	c.Assert(err, Equals, ErrTestImageNotInRegistry)
	rImage := &registry.Image{
		Library: "BASE",
		Repo:    "repo",
		Tag:     "latest",
		UUID:    "testuuid",
		Hash:    "hashvalue",
	}
	s.index.On("FindImage", "BASE/repo:latest").Return(rImage, nil)
	s.registry.On("FindImage", rImage).Return(&dockerclient.Image{}, nil).Once()
	s.index.On("PushImage", mock.AnythingOfType("string"), "testuuid", "hashvalue").Return(ErrTestNoPush).Run(func(a mock.Arguments) {
		newRegistryImage := a.Get(0).(string)
		c.Assert(strings.HasPrefix(newRegistryImage, "BASE/repo:"), Equals, true)
	})
	id, err = s.dfs.Snapshot(data)
	c.Assert(id, Equals, "")
	c.Assert(err, Equals, ErrTestNoPush)
}

func (s *DFSTestSuite) TestSnapshot_MissingImage(c *C) {
	data := SnapshotInfo{
		SnapshotInfo: &volume.SnapshotInfo{
			TenantID: "BASE",
			Tags:     []string{"tag1"},
		},
		Images: []string{"BASE/repo:latest"},
	}
	rImage := &registry.Image{
		Library: "BASE",
		Repo:    "repo",
		Tag:     "latest",
		UUID:    "testuuid",
		Hash:    "hashvalue",
	}

	vol := &volumemocks.Volume{}
	s.disk.On("Get", "BASE").Return(vol, nil)
	s.index.On("FindImage", "BASE/repo:latest").Return(rImage, nil).Once()
	s.registry.On("FindImage", rImage).Return(nil, ErrTestImageNotFound).Once()
	id, err := s.dfs.Snapshot(data)
	c.Assert(id, Equals, "")
	c.Assert(err, Equals, ErrTestImageNotFound)
}

func (s *DFSTestSuite) TestSnapshot_NoWriteMetadata(c *C) {
	data := SnapshotInfo{
		SnapshotInfo: &volume.SnapshotInfo{
			TenantID: "BASE",
			Tags:     []string{"tag1"},
		},
		Images: []string{"BASE/repo:latest"},
		Services: []service.Service{
			{ID: "test-service", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		},
	}
	vol := &volumemocks.Volume{}
	rImage := &registry.Image{
		Library: "BASE",
		Repo:    "repo",
		Tag:     "latest",
		UUID:    "testuuid",
		Hash:    "hashvalue",
	}
	s.disk.On("Get", "BASE").Return(vol, nil)
	s.index.On("FindImage", "BASE/repo:latest").Return(rImage, nil)
	s.registry.On("FindImage", rImage).Return(&dockerclient.Image{}, nil)
	s.index.On("PushImage", mock.AnythingOfType("string"), "testuuid", "hashvalue").Return(nil).Run(func(a mock.Arguments) {
		newRegistryImage := a.Get(0).(string)
		c.Assert(strings.HasPrefix(newRegistryImage, "BASE/repo:"), Equals, true)
		s.registry.On("ImagePath", newRegistryImage).Return("test:5000/"+newRegistryImage, nil)
	})
	vol.On("WriteMetadata", mock.AnythingOfType("string"), ImagesMetadataFile).Return(&NopCloser{}, ErrTestNoImagesMetadata)
	vol.On("WriteMetadata", mock.AnythingOfType("string"), ServicesMetadataFile).Return(&NopCloser{bytes.NewBufferString("")}, nil)
	id, err := s.dfs.Snapshot(data)
	c.Assert(id, Equals, "")
	c.Assert(err, Equals, ErrTestNoImagesMetadata)
	vol.On("WriteMetadata", mock.AnythingOfType("string"), ServicesMetadataFile).Return(&NopCloser{}, ErrTestNoServicesMetadata)
	vol.On("WriteMetadata", mock.AnythingOfType("string"), ImagesMetadataFile).Return(&NopCloser{bytes.NewBufferString("")}, nil)
	id, err = s.dfs.Snapshot(data)
	c.Assert(id, Equals, "")
	c.Assert(err, Equals, ErrTestNoImagesMetadata)
}

func (s *DFSTestSuite) TestSnapshot_NoSnapshot(c *C) {
	data := SnapshotInfo{
		SnapshotInfo: &volume.SnapshotInfo{
			TenantID: "BASE",
			Tags:     []string{"tag1"},
		},
		Images: []string{"BASE/repo:latest"},
		Services: []service.Service{
			{ID: "test-service", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		},
	}
	vol := &volumemocks.Volume{}
	rImage := &registry.Image{
		Library: "BASE",
		Repo:    "repo",
		Tag:     "latest",
		UUID:    "testuuid",
		Hash:    "hashvalue",
	}
	s.disk.On("Get", "BASE").Return(vol, nil)
	s.index.On("FindImage", "BASE/repo:latest").Return(rImage, nil)
	s.registry.On("FindImage", rImage).Return(&dockerclient.Image{}, nil)
	s.index.On("PushImage", mock.AnythingOfType("string"), "testuuid", "hashvalue").Return(nil).Run(func(a mock.Arguments) {
		newRegistryImage := a.Get(0).(string)
		c.Assert(strings.HasPrefix(newRegistryImage, "BASE/repo:"), Equals, true)
		s.registry.On("ImagePath", newRegistryImage).Return("test:5000/"+newRegistryImage, nil)
	})
	vol.On("WriteMetadata", mock.AnythingOfType("string"), ImagesMetadataFile).Return(&NopCloser{bytes.NewBufferString("")}, nil)
	vol.On("WriteMetadata", mock.AnythingOfType("string"), ServicesMetadataFile).Return(&NopCloser{bytes.NewBufferString("")}, nil)
	vol.On("Snapshot", mock.AnythingOfType("string"), data.Message, data.Tags).Return(ErrTestSnapshotNotCreated).Once()
	id, err := s.dfs.Snapshot(data)
	c.Assert(id, Equals, "")
	c.Assert(err, Equals, ErrTestSnapshotNotCreated)
	vol.On("Snapshot", mock.AnythingOfType("string"), data.Message, data.Tags).Return(nil).Once()
	vol.On("SnapshotInfo", mock.AnythingOfType("string")).Return(&volume.SnapshotInfo{}, ErrTestInfoNotFound)
	id, err = s.dfs.Snapshot(data)
	c.Assert(id, Equals, "")
	c.Assert(err, Equals, ErrTestInfoNotFound)
}

func (s *DFSTestSuite) TestSnapshot_Success(c *C) {
	data := SnapshotInfo{
		SnapshotInfo: &volume.SnapshotInfo{
			TenantID: "BASE",
			Tags:     []string{"tag1"},
		},
		Images: []string{"BASE/repo:latest"},
		Services: []service.Service{
			{ID: "test-service", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		},
	}
	vol := &volumemocks.Volume{}
	rImage := &registry.Image{
		Library: "BASE",
		Repo:    "repo",
		Tag:     "latest",
		UUID:    "testuuid",
		Hash:    "hashvalue",
	}
	s.disk.On("Get", "BASE").Return(vol, nil)
	s.index.On("FindImage", "BASE/repo:latest").Return(rImage, nil)
	s.registry.On("FindImage", rImage).Return(&dockerclient.Image{}, nil).Once()
	s.index.On("PushImage", mock.AnythingOfType("string"), "testuuid", "hashvalue").Return(nil).Run(func(a mock.Arguments) {
		newRegistryImage := a.Get(0).(string)
		c.Assert(strings.HasPrefix(newRegistryImage, "BASE/repo:"), Equals, true)
		s.registry.On("ImagePath", newRegistryImage).Return("test:5000/"+newRegistryImage, nil)
	})
	imagesBuffer := bytes.NewBufferString("")
	servicesBuffer := bytes.NewBufferString("")
	vol.On("WriteMetadata", mock.AnythingOfType("string"), ImagesMetadataFile).Return(&NopCloser{imagesBuffer}, nil)
	vol.On("WriteMetadata", mock.AnythingOfType("string"), ServicesMetadataFile).Return(&NopCloser{servicesBuffer}, nil)
	var name string
	vol.On("Snapshot", mock.AnythingOfType("string"), data.Message, data.Tags).Return(nil).Run(func(a mock.Arguments) {
		label := a.Get(0).(string)
		name = "BASE_" + label
		var actualImages []string
		err := json.NewDecoder(imagesBuffer).Decode(&actualImages)
		c.Assert(err, IsNil)
		c.Assert(actualImages, DeepEquals, []string{"test:5000/BASE/repo:" + label})
		var actualServices []service.Service
		err = json.NewDecoder(servicesBuffer).Decode(&actualServices)
		c.Assert(err, IsNil)
		c.Assert(actualServices, DeepEquals, data.Services)

		sInfo := volume.SnapshotInfo{
			Name:     name,
			TenantID: "BASE",
			Label:    label,
			Message:  a.Get(1).(string),
			Tags:     a.Get(2).([]string),
			Created:  time.Now().UTC(),
		}
		vol.On("SnapshotInfo", label).Return(&sInfo, nil)
	})
	id, err := s.dfs.Snapshot(data)
	c.Assert(id, Equals, name)
	c.Assert(err, IsNil)
}
