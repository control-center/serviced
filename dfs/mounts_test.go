// Copyright 2017 The Serviced Authors.
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
	"fmt"
	"path"

	. "gopkg.in/check.v1"
	volumemocks "github.com/control-center/serviced/volume/mocks"
)


// This test returns a different device for the volume and export paths.
func (s *DFSTestSuite) TestInfo_VerifyTenantMounts_InvalidMounts(c *C) {
	tenantID := "test-tenant"
	exportPath := "export-path"

	// set up the test.
	vol := &volumemocks.Volume{}
	vol.On("Path").Return("volume-path")

	s.disk.On("Get", "test-tenant").Return(vol, nil)
	s.net.On("ExportNamePath").Return(exportPath)
	exportPath = path.Join(exportPath, tenantID)

	s.net.On("GetDevice", "volume-path").Return(uint64(12345), nil)
	s.net.On("GetDevice", exportPath).Return(uint64(23456), nil)

	err := s.dfs.VerifyTenantMounts(tenantID)
	c.Assert(err, NotNil)
}

// This test returns the same device for the volume and export paths.
func (s *DFSTestSuite) TestInfo_VerifyTenantMounts_ValidMounts(c *C) {
	tenantID := "test-tenant"
	exportPath := "export-path"

	// set up the test.
	vol := &volumemocks.Volume{}
	vol.On("Path").Return("volume-path")

	s.disk.On("Get", "test-tenant").Return(vol, nil)
	s.net.On("ExportNamePath").Return(exportPath)
	exportPath = path.Join(exportPath, tenantID)

	s.net.On("GetDevice", "volume-path").Return(uint64(12345), nil)
	s.net.On("GetDevice", exportPath).Return(uint64(12345), nil)

	err := s.dfs.VerifyTenantMounts(tenantID)
	c.Assert(err, IsNil)
}

// This test returns an error from GetDevice on the export path; we expect VerifyTenantMounts
// to return this error.
func (s *DFSTestSuite) TestInfo_VerifyTenantMounts_ErrorOnGetDevice(c *C) {
	tenantID := "test-tenant"
	exportPath := "export-path"

	// set up the test.
	vol := &volumemocks.Volume{}
	vol.On("Path").Return("volume-path")

	s.disk.On("Get", "test-tenant").Return(vol, nil)
	s.net.On("ExportNamePath").Return(exportPath)
	exportPath = path.Join(exportPath, tenantID)

	s.net.On("GetDevice", "volume-path").Return(uint64(12345), nil)
	s.net.On("GetDevice", exportPath).Return(uint64(0), fmt.Errorf("some error"))

	err := s.dfs.VerifyTenantMounts(tenantID)
	c.Assert(err, NotNil)
}
