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
	"errors"
	"time"

	"github.com/control-center/serviced/dfs/dfs"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	volumemocks "github.com/control-center/serviced/volume/mocks"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestInfo_NoSnapshot(c *C) {
	s.disk.On("Get", "test-label").Return(nil, errors.New("snapshot not found"))
	info, err := s.dfs.Info("test-label")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, errors.New("snapshot not found"))
	snapshot := &volumemocks.Volume{}
	s.disk.On("Get", "test-label-2").Return(snapshot, nil)
	snapshot.On("Tenant").Return("tenant-service")
	s.disk.On("Get", "tenant-service").Return(nil, errors.New("volume not found"))
	info, err = s.dfs.Info("test-label-2")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, errors.New("volume not found"))
	snapshot = &volumemocks.Volume{}
	s.disk.On("Get", "test-label-3").Return(snapshot, nil)
	s.disk.On("Tenant").Return("tenant-service-2")
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "tenant-service-2").Return(vol, nil)
	vol.On("SnapshotInfo", "test-label-3").Return(nil, errors.New("info not found"))
	info, err = s.dfs.Info("test-label-3")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, errors.New("info not found"))
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
	vol.On("ReadMetadata", "test-snapshot-label", dfs.ServicesMetadataFile).Return(svcbuffer, nil)
	vol.On("ReadMetadata", "test-snapshot-label", dfs.ImagesMetadataFile).Return(nil, errors.New("no images metadata"))
	info, err := s.dfs.Info("test-snapshot-label")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, errors.New("no images metadata"))
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
	vol.On("ReadMetadata", "test-snapshot-label", dfs.ServicesMetadataFile).Return(nil, errors.New("no services metadata"))
	vol.On("ReadMetadata", "test-snapshot-label", dfs.ImagesMetadataFile).Return(imgbuffer, nil)
	info, err := s.dfs.Info("test-snapshot-label")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, errors.New("no services metadata"))
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
			ID: "test-service-1",
		}, {
			ID: "test-service-2",
		}, {
			ID: "test-service-3",
		},
	}
	svcsbuffer := bytes.NewBufferString("")
	err := json.NewEncoder(svcsbuffer).Encode(svcs)
	c.Assert(err, IsNil)
	vol.On("ReadMetadata", "test-snapshot-label", dfs.ServicesMetadataFile).Return(svcsbuffer, nil)
	imgs := []string{"test-tenant/repo:snapshot-label"}
	imgsbuffer := bytes.NewBufferString("")
	err = json.NewEncoder(imgsbuffer).Encode(imgs)
	c.Assert(err, IsNil)
	vol.On("ReadMetadata", "test-snapshot-label", dfs.ServicesMetadataFile).Return(imgsbuffer, nil)
	info, err := s.dfs.Info("test-snapshot-label")
	c.Assert(err, IsNil)
	c.Assert(info, DeepEquals, &dfs.SnapshotInfo{vinfo, imgs, svcs})
}

func (s *DFSTestSuite) getVolumeFromSnapshot(snapshotID, tenantID string) *volumemocks.Volume {
	snapshot := &volumemocks.Volume{}
	s.disk.On("Get", snapshot).Return(snapshotID, nil)
	snapshot.On("Tenant").Return(tenantID)
	volume := &volumemocks.Volume{}
	s.disk.On("Get", tenantID).Return(volume, nil)
	return volume
}
