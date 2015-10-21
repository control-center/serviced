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
	volumemocks "github.com/control-center/serviced/volume/mocks"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestList_NoVolume(c *C) {
	s.disk.On("Get", "tenant").Return(&volumemocks.Volume{}, ErrTestVolumeNotFound)
	snapshots, err := s.dfs.List("tenant")
	c.Assert(snapshots, IsNil)
	c.Assert(err, Equals, ErrTestVolumeNotFound)
}

func (s *DFSTestSuite) TestList_NoSnapshots(c *C) {
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "tenant").Return(vol, nil)
	vol.On("Snapshots").Return(nil, ErrTestNoSnapshots)
	snapshots, err := s.dfs.List("tenant")
	c.Assert(snapshots, IsNil)
	c.Assert(err, Equals, ErrTestNoSnapshots)
}

func (s *DFSTestSuite) TestList_Success(c *C) {
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "tenant").Return(vol, nil)
	snaps := []string{"tenant_label1", "tenant_label2"}
	vol.On("Snapshots").Return(snaps, nil)
	snapshots, err := s.dfs.List("tenant")
	c.Assert(snapshots, DeepEquals, snaps)
	c.Assert(err, IsNil)
}
