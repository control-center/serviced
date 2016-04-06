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
	"io"
	"time"

	. "github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/volume"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *DFSTestSuite) TestBackup_ImageNotFound(c *C) {
	buf := bytes.NewBufferString("")
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
		Snapshots: []string{"testtenant_testlabel"},
		Timestamp: time.Now().UTC(),
	}
	s.docker.On("FindImage", "library/repo:tag").Return(&dockerclient.Image{}, ErrTestImageNotFound).Once()
	err := s.dfs.Backup(backupInfo, buf)
	c.Assert(err, Equals, ErrTestImageNotFound)
	buf.Reset()
	s.docker.On("FindImage", "library/repo:tag").Return(&dockerclient.Image{}, dockerclient.ErrNoSuchImage).Once()
	s.docker.On("PullImage", "library/repo:tag").Return(ErrTestNoPull).Once()
	err = s.dfs.Backup(backupInfo, buf)
	c.Assert(err, Equals, ErrTestNoPull)
}

func (s *DFSTestSuite) TestBackup_SkipTemplateImage(c *C) {
	buf := bytes.NewBufferString("")
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
	s.docker.On("FindImage", "library/repo:tag").Return(&dockerclient.Image{}, dockerclient.ErrNoSuchImage).Once()
	s.docker.On("PullImage", "library/repo:tag").Return(dockerclient.ErrNoSuchImage)
	vol := s.getVolumeFromSnapshot("BASE_LABEL", "BASE")
	info := &volume.SnapshotInfo{
		Name:     "BASE_LABEL",
		TenantID: "BASE",
		Label:    "LABEL",
		Created:  time.Now().UTC(),
	}
	imagesbuf := bytes.NewBufferString("")
	err := json.NewEncoder(imagesbuf).Encode([]string{"BASE/repo:tag"})
	c.Assert(err, IsNil)
	vol.On("SnapshotInfo", "BASE_LABEL").Return(info, nil)
	vol.On("ReadMetadata", "LABEL", ImagesMetadataFile).Return(&NopCloser{imagesbuf}, nil)
	s.registry.On("PullImage", mock.AnythingOfType("<-chan time.Time"), "BASE/repo:tag").Return(nil)
	s.registry.On("ImagePath", "BASE/repo:tag").Return("testserver:5000/BASE/repo:tag", nil)
	vol.On("Export", "LABEL", "", mock.AnythingOfType("*io.PipeWriter")).Return(nil).Run(func(a mock.Arguments) {
		writer := a.Get(2).(io.Writer)
		tarwriter := tar.NewWriter(writer)
		data := []byte("here is some snapshot data")
		hdr := &tar.Header{Name: "afile", Size: int64(len(data))}
		tarwriter.WriteHeader(hdr)
		tarwriter.Write(data)
		tarwriter.Close()
	})
	allImages := []string{"testserver:5000/BASE/repo:tag"}
	s.docker.On("SaveImages", allImages, mock.AnythingOfType("*io.PipeWriter")).Return(nil).Run(func(a mock.Arguments) {
		writer := a.Get(1).(io.Writer)
		tarwriter := tar.NewWriter(writer)
		data := []byte("here is some snapshot data")
		hdr := &tar.Header{Name: "afile", Size: int64(len(data))}
		tarwriter.WriteHeader(hdr)
		tarwriter.Write(data)
		tarwriter.Close()
	})
	err = s.dfs.Backup(backupInfo, buf)
	c.Assert(err, IsNil)
	c.Assert(buf.Len() > 0, Equals, true)
}

func (s *DFSTestSuite) TestBackup(c *C) {
	buf := bytes.NewBufferString("")
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
	s.docker.On("FindImage", "library/repo:tag").Return(&dockerclient.Image{}, dockerclient.ErrNoSuchImage).Once()
	s.docker.On("PullImage", "library/repo:tag").Return(nil)
	vol := s.getVolumeFromSnapshot("BASE_LABEL", "BASE")
	info := &volume.SnapshotInfo{
		Name:     "BASE_LABEL",
		TenantID: "BASE",
		Label:    "LABEL",
		Created:  time.Now().UTC(),
	}
	imagesbuf := bytes.NewBufferString("")
	err := json.NewEncoder(imagesbuf).Encode([]string{"BASE/repo:tag"})
	c.Assert(err, IsNil)
	vol.On("SnapshotInfo", "BASE_LABEL").Return(info, nil)
	vol.On("ReadMetadata", "LABEL", ImagesMetadataFile).Return(&NopCloser{imagesbuf}, nil)
	s.registry.On("PullImage", mock.AnythingOfType("<-chan time.Time"), "BASE/repo:tag").Return(nil)
	s.registry.On("ImagePath", "BASE/repo:tag").Return("testserver:5000/BASE/repo:tag", nil)
	vol.On("Export", "LABEL", "", mock.AnythingOfType("*io.PipeWriter")).Return(nil).Run(func(a mock.Arguments) {
		writer := a.Get(2).(io.Writer)
		tarwriter := tar.NewWriter(writer)
		data := []byte("here is some snapshot data")
		hdr := &tar.Header{Name: "afile", Size: int64(len(data))}
		tarwriter.WriteHeader(hdr)
		tarwriter.Write(data)
		tarwriter.Close()
	})
	allImages := append(backupInfo.BaseImages, "testserver:5000/BASE/repo:tag")
	s.docker.On("SaveImages", allImages, mock.AnythingOfType("*io.PipeWriter")).Return(nil).Run(func(a mock.Arguments) {
		writer := a.Get(1).(io.Writer)
		tarwriter := tar.NewWriter(writer)
		data := []byte("here is some snapshot data")
		hdr := &tar.Header{Name: "afile", Size: int64(len(data))}
		tarwriter.WriteHeader(hdr)
		tarwriter.Write(data)
		tarwriter.Close()
	})
	err = s.dfs.Backup(backupInfo, buf)
	c.Assert(err, IsNil)
	c.Assert(buf.Len() > 0, Equals, true)
}
