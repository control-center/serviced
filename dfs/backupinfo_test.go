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
	"archive/tar"
	"bytes"
	"encoding/json"
	"time"

	. "github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/servicetemplate"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestBackupInfo_NotATar(c *C) {
	buf := bytes.NewBufferString("this is not a tarfile")
	binfo, err := s.dfs.BackupInfo(buf)
	c.Assert(binfo, IsNil)
	c.Assert(err, NotNil)
}

func (s *DFSTestSuite) TestBackupInfo_NoMetadataFile(c *C) {
	marshal := []byte("this is just a file")
	buf := bytes.NewBufferString("")
	tarfile := tar.NewWriter(buf)
	hdr := &tar.Header{
		Name: "somefile",
		Size: int64(len(marshal)),
	}
	err := tarfile.WriteHeader(hdr)
	c.Assert(err, IsNil)
	n, err := tarfile.Write(marshal)
	c.Assert(err, IsNil)
	c.Assert(int64(n), Equals, hdr.Size)
	err = tarfile.Close()
	c.Assert(err, IsNil)
	binfo, err := s.dfs.BackupInfo(buf)
	c.Assert(binfo, IsNil)
	c.Assert(err, NotNil)
}

func (s *DFSTestSuite) TestBackupInfo_BadData(c *C) {
	marshal := []byte("this is bad data")
	buf := bytes.NewBufferString("")
	tarfile := tar.NewWriter(buf)
	hdr := &tar.Header{
		Name: BackupMetadataFile,
		Size: int64(len(marshal)),
	}
	err := tarfile.WriteHeader(hdr)
	c.Assert(err, IsNil)
	n, err := tarfile.Write(marshal)
	c.Assert(err, IsNil)
	c.Assert(int64(n), Equals, hdr.Size)
	binfo, err := s.dfs.BackupInfo(buf)
	c.Assert(binfo, IsNil)
	c.Assert(err, NotNil)
}

func (s *DFSTestSuite) TestBackupInfo_Success(c *C) {
	expected := BackupInfo{
		Templates: []servicetemplate.ServiceTemplate{
			{ID: "test-template-1"},
		},
		BaseImages: []string{"library/repo:tag"},
		Pools: []pool.ResourcePool{
			{ID: "test-pool-1", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		},
		Hosts: []host.Host{
			{ID: "test-host-1", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		},
		Snapshots: []string{"testtenant_testlabel"},
		Timestamp: time.Now().UTC(),
	}
	marshal, err := json.Marshal(expected)
	c.Assert(err, IsNil)
	buf := bytes.NewBufferString("")
	tarfile := tar.NewWriter(buf)
	hdr := &tar.Header{
		Name: BackupMetadataFile,
		Size: int64(len(marshal)),
	}
	err = tarfile.WriteHeader(hdr)
	c.Assert(err, IsNil)
	n, err := tarfile.Write(marshal)
	c.Assert(err, IsNil)
	c.Assert(int64(n), Equals, hdr.Size)
	actual, err := s.dfs.BackupInfo(buf)
	c.Assert(actual, DeepEquals, &expected)
	c.Assert(err, IsNil)
}
