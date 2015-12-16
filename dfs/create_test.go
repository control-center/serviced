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

func (s *DFSTestSuite) TestCreate_NoShare(c *C) {
	// TODO: this is a placeholder for nfs sharing
}

func (s *DFSTestSuite) TestCreate_ShareNotEnabled(c *C) {
	// TODO: this is a placeholder for nfs sharing
}

func (s *DFSTestSuite) TestCreate_Success(c *C) {
	vol := &volumemocks.Volume{}
	vol.On("Path").Return("testPath")
	s.disk.On("Create", "tenantid").Return(vol, nil)
	s.net.On("AddVolume", "testPath").Return(nil)
	s.net.On("Sync").Return(nil)
	err := s.dfs.Create("tenantid")
	c.Assert(err, IsNil)
}
