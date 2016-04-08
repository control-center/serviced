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

// +build integration

package dfs_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
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

/*
* Benchmarking backups with some fake data. We'll have Docker and the DFS each
* return a random 100M tar.
 */

const (
	TenantID      = "TENANT"
	SnapshotLabel = "SNAP"
	SnapshotName  = "TENANT_SNAP"
)

var i int

func (s *DFSTestSuite) BenchmarkBackup(c *C) {
	//1012318498
	// Create a file to play with
	dir := c.MkDir()
	i += 1
	c.Logf("Creating 100M file of random data: %d", i)
	random := fmt.Sprintf("%s/rand.file", dir)
	tar := fmt.Sprintf("%s/rand.tar", dir)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /dev/urandom | head -c100M > %s", random)).Run()
	exec.Command("tar", "cf", tar, random).Run()

	tardata, _ := ioutil.ReadFile(tar)

	now := time.Now().UTC()

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
		Snapshots: []string{SnapshotName},
		Timestamp: now,
	}
	snapshotInfo := volume.SnapshotInfo{
		Name:     SnapshotName,
		TenantID: TenantID,
		Label:    SnapshotLabel,
		Created:  now,
	}
	allImages := append(backupInfo.BaseImages, "testserver:5000/TENANT/repo:tag")
	imagesbuf := bytes.NewBufferString("")

	c.ResetTimer()

	for i := 0; i < c.N; i++ {
		c.StopTimer()
		imagesbuf.Reset()
		json.NewEncoder(imagesbuf).Encode([]string{"TENANT/repo:tag"})
		vol := s.getVolumeFromSnapshot(SnapshotName, TenantID)
		vol.On("SnapshotInfo", SnapshotName).Return(&snapshotInfo, nil)
		vol.On("ReadMetadata", SnapshotLabel, ImagesMetadataFile).Return(&NopCloser{imagesbuf}, nil)
		c.Logf("ReadMetadata should return %#v", imagesbuf)
		s.registry.On("PullImage", mock.AnythingOfType("<-chan time.Time"), "TENANT/repo:tag").Return(nil)
		s.docker.On("PullImage", "library/repo:tag").Return(nil)
		s.docker.On("FindImage", "library/repo:tag").Return(&dockerclient.Image{}, dockerclient.ErrNoSuchImage)
		s.registry.On("ImagePath", "TENANT/repo:tag").Return("testserver:5000/TENANT/repo:tag", nil)
		vol.On("Export", SnapshotLabel, "", mock.AnythingOfType("*io.PipeWriter")).Return(nil).Run(func(a mock.Arguments) {
			writer := a.Get(2).(io.Writer)
			_, err := writer.Write(tardata)
			c.Assert(err, IsNil)
		})
		s.docker.On("SaveImages", allImages, mock.AnythingOfType("*io.PipeWriter")).Return(nil).Run(func(a mock.Arguments) {
			writer := a.Get(1).(io.Writer)
			_, err := writer.Write(tardata)
			c.Assert(err, IsNil)
		})
		buf := bytes.NewBufferString("")
		c.StartTimer()
		err := s.dfs.Backup(backupInfo, buf)
		c.Assert(err, IsNil)
	}
}
