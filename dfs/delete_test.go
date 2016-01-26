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
	"time"

	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/volume"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestDelete_NoImages(c *C) {
	vol := s.getVolumeFromSnapshot("Base_Snapshot", "Base")
	vinfo := &volume.SnapshotInfo{
		Name:     "Base_Snapshot",
		TenantID: "Base",
		Label:    "Snapshot",
		Created:  time.Now().UTC(),
	}
	vol.On("SnapshotInfo", "Base_Snapshot").Return(vinfo, nil)
	s.index.On("SearchLibraryByTag", "Base", "Snapshot").Return(nil, ErrTestImageNotFound)
	err := s.dfs.Delete("Base_Snapshot")
	c.Assert(err, Equals, ErrTestImageNotFound)
}

func (s *DFSTestSuite) TestDelete_NoRemove(c *C) {
	// image won't delete
	vol := s.getVolumeFromSnapshot("Base_Snapshot", "Base")
	vinfo := &volume.SnapshotInfo{
		Name:     "Base_Snapshot",
		TenantID: "Base",
		Label:    "Snapshot",
		Created:  time.Now().UTC(),
	}
	vol.On("SnapshotInfo", "Base_Snapshot").Return(vinfo, nil)
	rImages := []registry.Image{
		{
			Library: "Base",
			Repo:    "repo",
			Tag:     "Snapshot",
		},
	}
	s.index.On("SearchLibraryByTag", "Base", "Snapshot").Return(rImages, nil)
	s.index.On("RemoveImage", "Base/repo:Snapshot").Return(ErrTestImageNotRemoved)
	err := s.dfs.Delete("Base_Snapshot")
	c.Assert(err, Equals, ErrTestImageNotRemoved)
	// volume won't delete
	vol = s.getVolumeFromSnapshot("Base2_Snapshot2", "Base2")
	vinfo = &volume.SnapshotInfo{
		Name:     "Base2_Snapshot2",
		TenantID: "Base2",
		Label:    "Snapshot2",
		Created:  time.Now().UTC(),
	}
	vol.On("SnapshotInfo", "Base2_Snapshot2").Return(vinfo, nil)
	rImages = []registry.Image{
		{
			Library: "Base2",
			Repo:    "repo",
			Tag:     "Snapshot2",
		},
	}
	s.index.On("SearchLibraryByTag", "Base2", "Snapshot2").Return(rImages, nil)
	s.index.On("RemoveImage", "Base2/repo:Snapshot2").Return(nil)
	vol.On("RemoveSnapshot", "Base2_Snapshot2").Return(ErrTestVolumeNotRemoved)
	err = s.dfs.Delete("Base2_Snapshot2")
	c.Assert(err, Equals, ErrTestVolumeNotRemoved)
}

func (s *DFSTestSuite) TestDelete_Success(c *C) {
	vol := s.getVolumeFromSnapshot("Base_Snapshot", "Base")
	vinfo := &volume.SnapshotInfo{
		Name:     "Base_Snapshot",
		TenantID: "Base",
		Label:    "Snapshot",
		Created:  time.Now().UTC(),
	}
	vol.On("SnapshotInfo", "Base_Snapshot").Return(vinfo, nil)
	rImages := []registry.Image{
		{
			Library: "Base",
			Repo:    "repo",
			Tag:     "Snapshot",
		},
	}
	s.index.On("SearchLibraryByTag", "Base", "Snapshot").Return(rImages, nil)
	s.index.On("RemoveImage", "Base/repo:Snapshot").Return(nil)
	vol.On("RemoveSnapshot", "Base_Snapshot").Return(nil)
	err := s.dfs.Delete("Base_Snapshot")
	c.Assert(err, IsNil)
}

func (s *DFSTestSuite) TestDelete_NoInfo_Success(c *C) {
	vol := s.getVolumeFromSnapshot("Base_Snapshot", "Base")
	vol.On("SnapshotInfo", "Base_Snapshot").Return(nil, volume.ErrInvalidSnapshot)
	vol.On("RemoveSnapshot", "Base_Snapshot").Return(nil)
	err := s.dfs.Delete("Base_Snapshot")
	c.Assert(err, IsNil)
}
