// Copyright 2015 The Serviced Authors.
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

package elasticsearch

import (
	"sync"

	"github.com/control-center/serviced/dao"
)

// InProgress prompts which backup is currently backing up or restoring
type InProgress struct {
	locker   *sync.RWMutex
	filename string
	op       string
}

// SetProgress sets the current operation
func (p *InProgress) SetProgress(filename, op string) {
	p.locker.Lock()
	defer p.locker.Unlock()
	p.filename = filename
	p.op = op
}

// UnsetProgress unsets the current operation
func (p *InProgress) UnsetProgress() {
	p.locker.Lock()
	defer p.locker.Unlock()
	p.filename = ""
	p.op = ""
}

// GetProgress returns the progress of the current running backup or restore.
func (p *InProgress) GetProgress() (string, string) {
	p.locker.RLock()
	defer p.locker.RUnlock()
	return p.filename, p.op
}

var inprogress = &InProgress{locker: &sync.RWMutex{}, filename: "", op: ""}

// Backup takes a backup of the full application stack and returns the filename
// that it is written to.
func (dao *ControlPlaneDao) Backup(dirpath string, filename *string) (err error) {
	return
}

// AsyncBackup is the same as backup, but asynchronous
func (dao *ControlPlaneDao) AsyncBackup(dirpath string, filename *string) (err error) {
	return
}

// Restore restores the full application stack from a backup file.
func (dao *ControlPlaneDao) Restore(filename string, _ *int) (err error) {
	return
}

// AsyncRestore is the same as restore, but asynchronous.
func (dao *ControlPlaneDao) AsyncRestore(filename string, _ *int) (err error) {
	return
}

// ListBackups returns the list of backups
func (dao *ControlPlaneDao) ListBackups(dirpath string, files *[]dao.BackupFile) (err error) {
	return
}

// BackupStatus returns the current status of the backup or restore that is
// running.
func (dao *ControlPlaneDao) BackupStatus(_ dao.EntityRequest, status *string) (err error) {
	return
}

// Snapshot captures the current state of a single application
func (dao *ControlPlaneDao) Snapshot(req dao.SnapshotRequest, snapshotID *string) (err error) {
	// TODO: Add DockerID to SnapshotRequest and combine snapshot and commit calls
	return
}

// Rollback reverts a single application to a particular state
func (dao *ControlPlaneDao) Rollback(req dao.RollbackRequest, _ *int) (err error) {
	return
}

// DeleteSnapshot deletes a single snapshot
func (dao *ControlPlaneDao) DeleteSnapshot(snapshotID string, _ *int) (err error) {
	return
}

// DeleteSnapshots deletes all snapshots for a service
func (dao *ControlPlaneDao) DeleteSnapshots(serviceID string, _ *int) (err error) {
	return
}

// ListSnapshots returns a list of all snapshots for a service
func (dao *ControlPlaneDao) ListSnapshots(serviceID string, snapshots *[]dao.SnapshotInfo) (err error) {
	return
}

// ResetRegistry prompts all images to be pushed back into the docker registry
func (dao *ControlPlaneDao) ResetRegistry(_ dao.EntityRequest, _ *int) (err error) {
	return
}

// RepairRegistry will try to recover the latest image of all service images
// from the docker registry and save it to the index.
func (dao *ControlPlaneDao) RepairRegistry(_ dao.EntityRequest, _ *int) (err error) {
	return
}

// ReadyDFS locks until it receives notice that the dfs is idle
func (dao *ControlPlaneDao) ReadyDFS(_ dao.EntityRequest, _ *int) (err error) {
	return
}
