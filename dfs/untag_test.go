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

	"github.com/control-center/serviced/volume"
	volumemocks "github.com/control-center/serviced/volume/mocks"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestUntag_NoVolume(c *C) {
	s.disk.On("Get", "Base").Return(&volumemocks.Volume{}, volume.ErrVolumeNotExists)
	label, err := s.dfs.Untag("Base", "tagA")
	c.Assert(err, Equals, volume.ErrVolumeNotExists)
	c.Assert(label, Equals, "")
}

func (s *DFSTestSuite) TestUntag_NoSnapshot(c *C) {
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "Base").Return(vol, nil)
	vol.On("UntagSnapshot", "tagA").Return("", volume.ErrSnapshotDoesNotExist)
	label, err := s.dfs.Untag("Base", "tagA")
	c.Assert(err, Equals, volume.ErrSnapshotDoesNotExist)
	c.Assert(label, Equals, "")
}

func (s *DFSTestSuite) TestUntag_NoInfo(c *C) {
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "Base").Return(vol, nil)
	vol.On("UntagSnapshot", "tagA").Return("Snap", nil)
	vol.On("SnapshotInfo", "Snap").Return(nil, ErrTestInfoNotFound)
	label, err := s.dfs.Untag("Base", "tagA")
	c.Assert(err, Equals, ErrTestInfoNotFound)
	c.Assert(label, Equals, "")
}

func (s *DFSTestSuite) TestUntag_Success(c *C) {
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "Base").Return(vol, nil)
	vol.On("UntagSnapshot", "tagA").Return("Snap", nil)
	info := &volume.SnapshotInfo{
		Name:     "Base_Snap",
		TenantID: "Base",
		Label:    "Snap",
		Message:  " this is a snapshot",
		Tags:     []string{},
		Created:  time.Now().UTC(),
	}
	vol.On("SnapshotInfo", "Snap").Return(info, nil)
	label, err := s.dfs.Untag("Base", "tagA")
	c.Assert(err, IsNil)
	c.Assert(label, Equals, "Base_Snap")
}
