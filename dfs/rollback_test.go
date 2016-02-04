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
	"time"

	. "github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/volume"
	volumemock "github.com/control-center/serviced/volume/mocks"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestRollback_NoSnapshot(c *C) {
	s.disk.On("GetTenant", "BASE_LABEL").Return(&volumemock.Volume{}, volume.ErrVolumeNotExists)
	err := s.dfs.Rollback("BASE_LABEL")
	c.Assert(err, Equals, volume.ErrVolumeNotExists)
	vol := s.getVolumeFromSnapshot("BASE2_LABEL2", "BASE2")
	vol.On("SnapshotInfo", "BASE2_LABEL2").Return(&volume.SnapshotInfo{}, ErrTestInfoNotFound)
	err = s.dfs.Rollback("BASE2_LABEL2")
	c.Assert(err, Equals, ErrTestInfoNotFound)
}

func (s *DFSTestSuite) TestRollback_NoImageMetadata(c *C) {
	vinfo := &volume.SnapshotInfo{
		Name:     "BASE_LABEL",
		TenantID: "BASE",
		Label:    "LABEL",
		Created:  time.Now().UTC(),
	}
	vol := s.getVolumeFromSnapshot("BASE_LABEL", "BASE")
	vol.On("SnapshotInfo", "BASE_LABEL").Return(vinfo, nil)
	vol.On("ReadMetadata", "LABEL", ImagesMetadataFile).Return(&NopCloser{}, ErrTestNoImagesMetadata)
	err := s.dfs.Rollback("BASE_LABEL")
	c.Assert(err, Equals, ErrTestNoImagesMetadata)
}

func (s *DFSTestSuite) TestRollback_ImageNoPush(c *C) {
	vinfo := &volume.SnapshotInfo{
		Name:     "BASE_LABEL",
		TenantID: "BASE",
		Label:    "LABEL",
		Created:  time.Now().UTC(),
	}
	vimages := []string{"BASE/repo:LABEL"}
	vimagesbuf := bytes.NewBufferString("")
	err := json.NewEncoder(vimagesbuf).Encode(vimages)
	c.Assert(err, IsNil)
	vol := s.getVolumeFromSnapshot("BASE_LABEL", "BASE")
	vol.On("SnapshotInfo", "BASE_LABEL").Return(vinfo, nil)
	vol.On("ReadMetadata", "LABEL", ImagesMetadataFile).Return(&NopCloser{vimagesbuf}, nil)
	s.index.On("FindImage", "BASE/repo:LABEL").Return(&registry.Image{}, ErrTestImageNotInRegistry).Once()
	err = s.dfs.Rollback("BASE_LABEL")
	c.Assert(err, Equals, ErrTestImageNotInRegistry)
	err = json.NewEncoder(vimagesbuf).Encode(vimages)
	c.Assert(err, IsNil)
	rImage := &registry.Image{
		Library: "BASE",
		Repo:    "repo",
		Tag:     "LABEL",
		UUID:    "testuuid",
		Hash:    "hashvalue",
	}
	s.index.On("FindImage", "BASE/repo:LABEL").Return(rImage, nil).Once()
	s.index.On("PushImage", "BASE/repo:latest", "testuuid", "hashvalue").Return(ErrTestNoPush)
	err = s.dfs.Rollback("BASE_LABEL")
	c.Assert(err, Equals, ErrTestNoPush)
}

func (s *DFSTestSuite) TestRollback_Success(c *C) {
	vinfo := &volume.SnapshotInfo{
		Name:     "BASE_LABEL",
		TenantID: "BASE",
		Label:    "LABEL",
		Created:  time.Now().UTC(),
	}
	vimages := []string{"BASE/repo:LABEL"}
	vimagesbuf := bytes.NewBufferString("")
	err := json.NewEncoder(vimagesbuf).Encode(vimages)
	c.Assert(err, IsNil)
	rImage := &registry.Image{
		Library: "BASE",
		Repo:    "repo",
		Tag:     "LABEL",
		UUID:    "testuuid",
		Hash:    "hashvalue",
	}
	vol := s.getVolumeFromSnapshot("BASE_LABEL", "BASE")
	vol.On("Path").Return("/path/to/tenantID")
	vol.On("SnapshotInfo", "BASE_LABEL").Return(vinfo, nil)
	vol.On("ReadMetadata", "LABEL", ImagesMetadataFile).Return(&NopCloser{vimagesbuf}, nil)
	s.index.On("FindImage", "BASE/repo:LABEL").Return(rImage, nil)
	s.index.On("PushImage", "BASE/repo:latest", "testuuid", "hashvalue").Return(nil)
	s.net.On("AddVolume", "/path/to/tenantID").Return(nil)
	s.net.On("RemoveVolume", "/path/to/tenantID").Return(nil)
	s.net.On("Stop").Return(nil)
	s.net.On("Restart").Return(nil)
	s.net.On("Sync").Return(nil)
	vol.On("Rollback", "LABEL").Return(nil)
	err = s.dfs.Rollback("BASE_LABEL")
	c.Assert(err, IsNil)
}
