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
	"path"
	"time"

	. "github.com/control-center/serviced/dfsnew"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/volume"
	volumemocks "github.com/control-center/serviced/volume/mocks"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestRestore_NoMetadata(c *C) {
	buf := bytes.NewBufferString("")
	tarfile := tar.NewWriter(buf)
	err := tarfile.WriteHeader(&tar.Header{Name: "IGNORED", Size: 0})
	c.Assert(err, IsNil)
	tarfile.Close()
	backupInfo, err := s.dfs.Restore(buf)
	c.Assert(backupInfo, IsNil)
	c.Assert(err, Equals, ErrRestoreNoInfo)
}

func (s *DFSTestSuite) TestRestore_LoadImages(c *C) {
	buf := bytes.NewBufferString("")
	tarfile := tar.NewWriter(buf)
	backupInfo := BackupInfo{
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
		Snapshots: []string{"BASE_LABEL"},
		Timestamp: time.Now().UTC(),
	}
	s.writeBackupInfo(c, tarfile, backupInfo)
	err := tarfile.WriteHeader(&tar.Header{Name: DockerImagesFile, Size: 0})
	c.Assert(err, IsNil)
	tarfile.Close()
	s.docker.On("LoadImage", mock.AnythingOfType("*tar.Reader")).Return(nil)
	actual, err := s.dfs.Restore(buf)
	c.Assert(actual, DeepEquals, &backupInfo)
	c.Assert(err, IsNil)
	s.docker.AssertExpectations(c)
}

func (s *DFSTestSuite) TestRestore_ImportSnapshot(c *C) {
	buf := bytes.NewBufferString("")
	tarfile := tar.NewWriter(buf)
	backupInfo := BackupInfo{
		Templates: []servicetemplate.ServiceTemplate{
			{ID: "test-template-1"},
		},
		BaseImages: []string{},
		Pools: []pool.ResourcePool{
			{ID: "test-pool-1", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		},
		Hosts: []host.Host{
			{ID: "test-host-1", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		},
		Snapshots: []string{"BASE_LABEL"},
		Timestamp: time.Now().UTC(),
	}
	s.writeBackupInfo(c, tarfile, backupInfo)
	err := tarfile.WriteHeader(&tar.Header{Name: path.Join(SnapshotsMetadataDir, "BASE", "LABEL"), Size: 0})
	c.Assert(err, IsNil)
	tarfile.Close()
	s.disk.On("Create", "BASE").Return(&volumemocks.Volume{}, volume.ErrVolumeExists)
	vol := &volumemocks.Volume{}
	s.disk.On("Get", "BASE").Return(vol, nil)
	vol.On("Import", "LABEL", mock.AnythingOfType("*tar.Reader")).Return(nil)
	actual, err := s.dfs.Restore(buf)
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, &backupInfo)
	s.disk.AssertExpectations(c)
	vol.AssertExpectations(c)
}

func (s *DFSTestSuite) writeBackupInfo(c *C, tarfile *tar.Writer, backupInfo BackupInfo) {
	bytedata, err := json.Marshal(backupInfo)
	c.Assert(err, IsNil)
	err = tarfile.WriteHeader(&tar.Header{Name: BackupMetadataFile, Size: int64(len(bytedata))})
	c.Assert(err, IsNil)
	_, err = tarfile.Write(bytedata)
	c.Assert(err, IsNil)
}
