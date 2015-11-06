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
	"github.com/control-center/serviced/volume"

	volumemocks "github.com/control-center/serviced/volume/mocks"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestGetSnapshotWithTag_NoVolume(c *C) {
	s.disk.On("Get", "tenant").Return(&volumemocks.Volume{}, ErrTestVolumeNotFound)
	snapshot, err := s.dfs.GetSnapshotWithTag("tenant", "someTag")
	c.Assert(snapshot, Equals, "")
	c.Assert(err, Equals, ErrTestVolumeNotFound)
}

func (s *DFSTestSuite) TestGetSnapshotWithTag_NoSnapshot(c *C) {
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "tenant").Return(vol, nil)
	vol.On("GetSnapshotWithTag", "someTag").Return(nil, ErrTestGetSnapshotByTagFailed)
	snapshot, err := s.dfs.GetSnapshotWithTag("tenant", "someTag")
	c.Assert(snapshot, Equals, "")
	c.Assert(err, Equals, ErrTestGetSnapshotByTagFailed)
}

func (s *DFSTestSuite) TestGetSnapshotWithTag_Success(c *C) {
	vol := &volumemocks.Volume{}
	snapshotInfo := &volume.SnapshotInfo{Name: "mySnapID"}
	s.disk.On("Get", "tenant").Return(vol, nil)
	vol.On("GetSnapshotWithTag", "someTag").Return(snapshotInfo, nil)
	snapshot, err := s.dfs.GetSnapshotWithTag("tenant", "someTag")
	c.Assert(snapshot, Equals, "mySnapID")
	c.Assert(err, IsNil)
}
