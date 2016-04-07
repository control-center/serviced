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

package dfs

import (
	"io"
	"time"

	csync "github.com/control-center/serviced/commons/sync"
	"github.com/control-center/serviced/coordinator/storage"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/volume"
)

// DFS is the api for the distributed filesystem
type DFS interface {
	DFSLocker
	// Timeout returns the dfs timeout setting
	Timeout() time.Duration
	// Create sets up a new application
	Create(tenantID string) error
	// Destroy removes an existing application
	Destroy(tenantID string) error
	// Download adds an image for an application into the registry
	Download(image, tenantID string, upgrade bool) (registry string, err error)
	// Commit uploads a new image into the registry
	Commit(ctrID string) (tenantID string, err error)
	// Snapshot captures application data at a specific point in time
	Snapshot(info SnapshotInfo) (string, error)
	// Rollback reverts application to a specific snapshot
	Rollback(snapshotID string) error
	// Delete deletes an application's snapshot
	Delete(snapshotID string) error
	// List lists snapshots for a particular application
	List(tenantID string) (snapshots []string, err error)
	// Info provides detailed info for a particular snapshot
	Info(snapshotID string) (*SnapshotInfo, error)
	// Backup saves and exports the current state of the system
	Backup(info BackupInfo, w io.Writer) error
	// Restore restores the system to the state of the backup
	Restore(r io.Reader, backupInfo *BackupInfo) error
	// BackupInfo provides detailed info for a particular backup
	BackupInfo(r io.Reader) (*BackupInfo, error)
	// Tag adds a tag to an existing snapshot
	Tag(snapshotID string, tagName string) error
	// Untag removes a tag from an existing snapshot
	Untag(tenantID, tagName string) (string, error)
	// TagInfo provides detailed info for a particular snapshot by given tag
	TagInfo(tenantID, tagName string) (*SnapshotInfo, error)
	// UpgradeRegistry loads images for each service
	// into the docker registry index
	UpgradeRegistry(svcs []service.Service, tenantID, registryHost string, override bool) error
	// Override replaces an image in the registry with a new image
	Override(newImage, oldImage string) error
}

var _ = DFS(&DistributedFilesystem{})

// BackupInfo provides meta info about a backup
type BackupInfo struct {
	Templates     []servicetemplate.ServiceTemplate
	BaseImages    []string
	Pools         []pool.ResourcePool
	Hosts         []host.Host
	Snapshots     []string
	Timestamp     time.Time
	BackupVersion int
}

// SnapshotInfo provides meta info about a snapshot
type SnapshotInfo struct {
	*volume.SnapshotInfo
	Images   []string
	Services []service.Service
}

// DistributedFilesystem manages disk and registry data for all system
// applications.
type DistributedFilesystem struct {
	docker docker.Docker
	index  registry.RegistryIndex
	reg    registry.Registry
	disk   volume.Driver
	// FIXME: replace this with a NFS server, instead of restarting the
	// daemon
	net     storage.StorageDriver
	timeout time.Duration
	locker  *csync.TimedMutex
	tmp     string // tmp directory where backups are temporarily spooled
}

// NewDistributedFilesystem instantiates a new DistributedFilsystem object
func NewDistributedFilesystem(docker docker.Docker, index registry.RegistryIndex, reg registry.Registry, disk volume.Driver, net storage.StorageDriver, timeout time.Duration) *DistributedFilesystem {
	return &DistributedFilesystem{
		docker:  docker,
		index:   index,
		reg:     reg,
		disk:    disk,
		net:     net,
		timeout: timeout,
		locker:  csync.NewTimedMutex(),
	}
}

// Timeout returns the service timeout time for the distributed filesystem
func (dfs *DistributedFilesystem) Timeout() time.Duration {
	return dfs.timeout
}

// SetTmp sets the temp directory for the spooler
func (dfs *DistributedFilesystem) SetTmp(tmp string) {
	dfs.tmp = tmp
}
