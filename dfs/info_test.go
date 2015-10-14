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

func (s *DFSTestSuite) TestInfo_NoSnapshot(c *C) {
	s.disk.On("Get", "test-label").Return(&volumemocks.Volume{}, ErrTestSnapshotNotFound)
	info, err := s.dfs.Info("test-label")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, ErrTestSnapshotNotFound)
	snapshot := &volumemocks.Volume{}
	s.disk.On("Get", "test-label-2").Return(snapshot, nil)
	snapshot.On("Tenant").Return("tenant-service")
	s.disk.On("Get", "tenant-service").Return(&volumemocks.Volume{}, ErrTestVolumeNotFound)
	info, err = s.dfs.Info("test-label-2")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, ErrTestVolumeNotFound)
	snapshot = &volumemocks.Volume{}
	s.disk.On("Get", "test-label-3").Return(snapshot, nil)
	snapshot.On("Tenant").Return("tenant-service-2")
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "tenant-service-2").Return(vol, nil)
	vol.On("SnapshotInfo", "test-label-3").Return(&volume.SnapshotInfo{}, ErrTestInfoNotFound)
	info, err = s.dfs.Info("test-label-3")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, ErrTestInfoNotFound)
}

func (s *DFSTestSuite) TestInfo_NoImages(c *C) {
	vol := s.getVolumeFromSnapshot("test-snapshot-label", "test-tenant")
	vinfo := &volume.SnapshotInfo{
		Name:     "test-snapshot-label",
		TenantID: "test-tenant",
		Label:    "snaphot-label",
		Message:  "this is a snapshot",
		Tags:     []string{"tag1", "tag2"},
		Created:  time.Now().UTC(),
	}
	vol.On("SnapshotInfo", "test-snapshot-label").Return(vinfo, nil)
	svcbuffer := bytes.NewBufferString("")
	err := json.NewEncoder(svcbuffer).Encode([]service.Service{})
	c.Assert(err, IsNil)
	vol.On("ReadMetadata", "test-snapshot-label", ServicesMetadataFile).Return(&NopCloser{svcbuffer}, nil)
	vol.On("ReadMetadata", "test-snapshot-label", ImagesMetadataFile).Return(&NopCloser{}, ErrTestNoImagesMetadata)
	info, err := s.dfs.Info("test-snapshot-label")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, ErrTestNoImagesMetadata)
}

func (s *DFSTestSuite) TestInfo_NoServices(c *C) {
	vol := s.getVolumeFromSnapshot("test-snapshot-label", "test-tenant")
	vinfo := &volume.SnapshotInfo{
		Name:     "test-snapshot-label",
		TenantID: "test-tenant",
		Label:    "snaphot-label",
		Message:  "this is a snapshot",
		Tags:     []string{"tag1", "tag2"},
		Created:  time.Now().UTC(),
	}
	vol.On("SnapshotInfo", "test-snapshot-label").Return(vinfo, nil)
	imgbuffer := bytes.NewBufferString("")
	err := json.NewEncoder(imgbuffer).Encode([]string{})
	c.Assert(err, IsNil)
	vol.On("ReadMetadata", "test-snapshot-label", ServicesMetadataFile).Return(&NopCloser{}, ErrTestNoServicesMetadata)
	vol.On("ReadMetadata", "test-snapshot-label", ImagesMetadataFile).Return(&NopCloser{imgbuffer}, nil)
	info, err := s.dfs.Info("test-snapshot-label")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, ErrTestNoServicesMetadata)
}

func (s *DFSTestSuite) TestInfo_Success(c *C) {
	vol := s.getVolumeFromSnapshot("test-snapshot-label", "test-tenant")
	vinfo := &volume.SnapshotInfo{
		Name:     "test-snapshot-label",
		TenantID: "test-tenant",
		Label:    "snaphot-label",
		Message:  "this is a snapshot",
		Tags:     []string{"tag1", "tag2"},
		Created:  time.Now().UTC(),
	}
	vol.On("SnapshotInfo", "test-snapshot-label").Return(vinfo, nil)
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
	vol.On("ReadMetadata", "test-snapshot-label", ServicesMetadataFile).Return(&NopCloser{svcsbuffer}, nil)
	imgs := []string{"test-tenant/repo:snapshot-label"}
	imgsbuffer := bytes.NewBufferString("")
	err = json.NewEncoder(imgsbuffer).Encode(imgs)
	c.Assert(err, IsNil)
	vol.On("ReadMetadata", "test-snapshot-label", ImagesMetadataFile).Return(&NopCloser{imgsbuffer}, nil)
	info, err := s.dfs.Info("test-snapshot-label")
	c.Assert(err, IsNil)
	c.Assert(info, DeepEquals, &SnapshotInfo{vinfo, imgs, svcs})
}

func (s *DFSTestSuite) getVolumeFromSnapshot(snapshotID, tenantID string) *volumemocks.Volume {
	snapshot := &volumemocks.Volume{}
	s.disk.On("Get", snapshotID).Return(snapshot, nil)
	snapshot.On("Tenant").Return(tenantID)
	volume := &volumemocks.Volume{}
	s.disk.On("Get", tenantID).Return(volume, nil)
	return volume
}
