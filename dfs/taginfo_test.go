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
	"bytes"
	"encoding/json"
	"time"

	. "github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	volumemocks "github.com/control-center/serviced/volume/mocks"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestTagInfo_NoSnapshot(c *C) {
	// volume not found
	s.disk.On("Get", "Base").Return(&volumemocks.Volume{}, volume.ErrVolumeNotExists).Once()
	info, err := s.dfs.TagInfo("Base", "tagA")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, volume.ErrVolumeNotExists)
	// tag not found
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "Base").Return(vol, nil)
	vol.On("GetSnapshotWithTag", "tagA").Return(&volume.SnapshotInfo{}, ErrTestInfoNotFound).Once()
	info, err = s.dfs.TagInfo("Base", "tagA")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, ErrTestInfoNotFound)
}

func (s *DFSTestSuite) TestTagInfo_Success(c *C) {
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "Base").Return(vol, nil)
	vinfo := &volume.SnapshotInfo{
		Name:     "Base_Snap",
		TenantID: "Base",
		Label:    "Snap",
		Message:  "this is a snapshot",
		Tags:     []string{"tagA"},
		Created:  time.Now().UTC(),
	}
	vol.On("GetSnapshotWithTag", "tagA").Return(vinfo, nil)
	svcs := []service.Service{
		{
			ID:        "test-service-1",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}, {
			ID:        "test-service-2",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}, {
			ID:        "test-service-3",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	svcsbuffer := bytes.NewBufferString("")
	err := json.NewEncoder(svcsbuffer).Encode(svcs)
	c.Assert(err, IsNil)
	vol.On("ReadMetadata", "Snap", ServicesMetadataFile).Return(&NopCloser{svcsbuffer}, nil)
	imgs := []string{"Base/repo:Snap"}
	imgsbuffer := bytes.NewBufferString("")
	err = json.NewEncoder(imgsbuffer).Encode(imgs)
	c.Assert(err, IsNil)
	vol.On("ReadMetadata", "Snap", ImagesMetadataFile).Return(&NopCloser{imgsbuffer}, nil)
	info, err := s.dfs.TagInfo("Base", "tagA")
	c.Assert(err, IsNil)
	c.Assert(info, DeepEquals, &SnapshotInfo{vinfo, imgs, svcs})
}
