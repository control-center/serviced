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

// +build unit integration

package dfs_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	storagemocks "github.com/control-center/serviced/coordinator/storage/mocks"
	. "github.com/control-center/serviced/dfs"
	dockermocks "github.com/control-center/serviced/dfs/docker/mocks"
	registrymocks "github.com/control-center/serviced/dfs/registry/mocks"
	volumemocks "github.com/control-center/serviced/volume/mocks"
)

// Errors returned by mocks for testing
var (
	ErrTestContainerNotFound       = errors.New("container not found")
	ErrTestImageNotFound           = errors.New("image not found")
	ErrTestNoCommit                = errors.New("container not committed")
	ErrTestNoPush                  = errors.New("image not pushed")
	ErrTestNoPull                  = errors.New("could not pull image")
	ErrTestNoTag                   = errors.New("could not tag image")
	ErrTestImageNotInRegistry      = errors.New("image not in registry")
	ErrTestImageNotRemoved         = errors.New("could not remove image")
	ErrTestVolumeNotCreated        = errors.New("could not create volume")
	ErrTestVolumeNotRemoved        = errors.New("could not remove volume")
	ErrTestVolumeNotFound          = errors.New("volume not found")
	ErrTestNoSnapshots             = errors.New("could not get snapshots")
	ErrTestServerRunning           = errors.New("could not stop server")
	ErrTestSnapshotNotFound        = errors.New("snapshot not found")
	ErrTestInfoNotFound            = errors.New("info not found")
	ErrTestNoImagesMetadata        = errors.New("no images metadata")
	ErrTestNoServicesMetadata      = errors.New("no services metadata")
	ErrTestSnapshotNotCreated      = errors.New("snapshot not created")
	ErrTestTagSnapshotFailed       = errors.New("unable to tag snapshot")
	ErrTestRemoveSnapshotTagFailed = errors.New("unable to remove tag from snapshot")
	ErrTestGetSnapshotByTagFailed  = errors.New("unable to retrieve snapshot by tag name")
	ErrTestGeneric                 = errors.New("something went wrong")
	ErrTestNoHash                  = errors.New("unable to get hash of image")
	ErrTestShareNotAdded           = errors.New("could not create share")
	ErrTestShareNotSynced          = errors.New("could not sync shares")
)

type NopCloser struct {
	*bytes.Buffer
}

func (h *NopCloser) Close() error {
	return nil
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&DFSTestSuite{})

type DFSTestSuite struct {
	docker   *dockermocks.Docker
	index    *registrymocks.RegistryIndex
	registry *registrymocks.Registry
	disk     *volumemocks.Driver
	net      *storagemocks.StorageDriver
	dfs      *DistributedFilesystem
}

func (s *DFSTestSuite) SetUpTest(c *C) {
	s.docker = &dockermocks.Docker{}
	s.index = &registrymocks.RegistryIndex{}
	s.registry = &registrymocks.Registry{}
	s.disk = &volumemocks.Driver{}
	s.net = &storagemocks.StorageDriver{}
	s.dfs = NewDistributedFilesystem(s.docker, s.index, s.registry, s.disk, s.net, time.Minute)
}

func (s *DFSTestSuite) getVolumeFromSnapshot(snapshotID, tenantID string) *volumemocks.Volume {
	vol := &volumemocks.Volume{}
	s.disk.On("GetTenant", snapshotID).Return(vol, nil)
	s.disk.On("Get", tenantID).Return(vol, nil)
	return vol
}
