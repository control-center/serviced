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
	"fmt"
	"os"

	"compress/gzip"
	"path/filepath"
	"sync"
	"time"

	model "github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/glog"
)

// InProgress prompts which backup is currently backing up or restoring
type InProgress struct {
	locker   *sync.RWMutex
	running  bool
	filename string
	op       string
	err      error
}

// SetProgress sets the current operation
func (p *InProgress) SetProgress(filename, op string) {
	p.locker.Lock()
	defer p.locker.Unlock()
	p.running = true
	p.filename = filename
	p.op = op
}

// SetError sets the error returned from the current operation
func (p *InProgress) SetError(err error) {
	p.locker.Lock()
	defer p.locker.Unlock()
	p.running = false
	p.err = err
}

// Reset resets the progress indicator
func (p *InProgress) Reset() {
	p.locker.Lock()
	defer p.locker.Unlock()
	p.running = false
	p.filename = ""
	p.op = ""
	p.err = nil
}

// UnsetProgress unsets the current operation
func (p *InProgress) UnsetProgress(err error) {
	p.locker.Lock()
	defer p.locker.Unlock()
	p.filename = ""
	p.op = ""
	p.err = err
}

// GetProgress returns the progress of the current running backup or restore.
func (p *InProgress) GetProgress() (bool, string, string, error) {
	p.locker.RLock()
	defer p.locker.RUnlock()
	return p.running, p.filename, p.op, p.err
}

var inprogress = &InProgress{locker: &sync.RWMutex{}}

// Backup takes a backup of the full application stack and returns the filename
// that it is written to.
func (dao *ControlPlaneDao) Backup(dirpath string, filename *string) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()

	// set the progress of the backup file
	*filename = time.Now().UTC().Format("backup-2006-01-02-150405.tgz")
	backupfilename := filepath.Join(dirpath, *filename)
	if dirpath == "" {
		backupfilename = filepath.Join(dao.backupsPath, *filename)
	}
	inprogress.SetProgress(backupfilename, "backup")
	defer func() {
		if err != nil {
			glog.Errorf("Backup failed with error: %s", err)
			os.Remove(backupfilename)
		}
		inprogress.SetError(err)
	}()
	// create the file and write
	fh, err := os.Create(backupfilename)
	if err != nil {
		glog.Errorf("Could not create backup file at %s: %s", backupfilename, err)
		return
	}
	defer fh.Close()
	gz := gzip.NewWriter(fh)
	defer gz.Close()
	err = dao.facade.Backup(ctx, gz)
	return
}

// AsyncBackup is the same as backup, but asynchronous
func (dao *ControlPlaneDao) AsyncBackup(dirpath string, filename *string) (err error) {
	go dao.Backup(dirpath, filename)
	return
}

// Restore restores the full application stack from a backup file.
func (dao *ControlPlaneDao) Restore(filename string, _ *int) error {
	ctx := datastore.Get()

	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()
	inprogress.SetProgress(filename, "restore")

	fh, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fh.Close()
	gz, err := gzip.NewReader(fh)
	if err != nil {
		return err
	}
	defer gz.Close()
	err = dao.facade.Restore(ctx, gz)
	inprogress.SetError(err)
	return err
}

// AsyncRestore is the same as restore, but asynchronous.
func (dao *ControlPlaneDao) AsyncRestore(filename string, unused *int) (err error) {
	go dao.Restore(filename, unused)
	return
}

// ListBackups returns the list of backups
func (dao *ControlPlaneDao) ListBackups(dirpath string, files *[]model.BackupFile) (err error) {
	*files = []model.BackupFile{}

	// Read the contents of the directory
	if dirpath == "" {
		dirpath = dao.backupsPath
	}
	dir, err := os.Open(dirpath)
	if err != nil {
		return err
	}
	defer dir.Close()

	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	// What is currently running?
	running, fp, _, _ := inprogress.GetProgress()

	for _, fi := range fis {
		if !fi.IsDir() {
			// Set up the backup file
			fullpath := filepath.Join(dirpath, fi.Name())
			bf := model.BackupFile{
				InProgress: running && fullpath == fp,
				FullPath:   fullpath,
				Name:       fi.Name(),
				Size:       fi.Size(),
				Mode:       fi.Mode(),
				ModTime:    fi.ModTime(),
			}
			// If it is not running, make sure the backup is legit
			if !bf.InProgress {
				isbackup := func(filename string) bool {
					fh, err := os.Open(fullpath)
					if err != nil {
						return false
					}
					defer fh.Close()
					gz, err := gzip.NewReader(fh)
					if err != nil {
						return false
					}
					defer gz.Close()
					_, err = dao.facade.BackupInfo(datastore.Get(), gz)
					if err != nil {
						return false
					}
					return true
				}(fullpath)
				if !isbackup {
					continue
				}
			}
			// Add the file to the list
			*files = append(*files, bf)
		}
	}
	return
}

// BackupStatus returns the current status of the backup or restore that is
// running.
func (dao *ControlPlaneDao) BackupStatus(_ model.EntityRequest, status *string) (err error) {
	running, filename, op, err := inprogress.GetProgress()
	if running {
		*status = fmt.Sprintf("Performing a %s on %s", op, filename)
	} else {
		if err != nil {
			*status = fmt.Sprintf("Completed a %s on %s with error: %s", op, filename, err)
		} else {
			inprogress.Reset()
			*status = ""
		}
	}
	return
}

// Snapshot captures the current state of a single application
func (dao *ControlPlaneDao) Snapshot(req model.SnapshotRequest, snapshotID *string) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()
	if req.ContainerID != "" {
		*snapshotID, err = dao.facade.Commit(ctx, req.ContainerID, req.Message, req.Tags)
	} else {
		*snapshotID, err = dao.facade.Snapshot(ctx, req.ServiceID, req.Message, req.Tags)
	}
	return
}

// Rollback reverts a single application to a particular state
func (dao *ControlPlaneDao) Rollback(req model.RollbackRequest, _ *int) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()
	err = dao.facade.Rollback(ctx, req.SnapshotID, req.ForceRestart)
	return
}

// DeleteSnapshot deletes a single snapshot
func (dao *ControlPlaneDao) DeleteSnapshot(snapshotID string, _ *int) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()
	err = dao.facade.DeleteSnapshot(ctx, snapshotID)
	return
}

// DeleteSnapshots deletes all snapshots for a service
func (dao *ControlPlaneDao) DeleteSnapshots(serviceID string, _ *int) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()
	err = dao.facade.DeleteSnapshots(ctx, serviceID)
	return
}

// ListSnapshots returns a list of all snapshots for a service
func (dao *ControlPlaneDao) ListSnapshots(serviceID string, snapshots *[]model.SnapshotInfo) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()

	*snapshots = make([]model.SnapshotInfo, 0)
	snapshotIDs, err := dao.facade.ListSnapshots(ctx, serviceID)
	if err != nil {
		return err
	}
	for _, snapshotID := range snapshotIDs {
		info, err := dao.facade.GetSnapshotInfo(ctx, snapshotID)
		if err != nil {
			return err
		}

		newInfo := model.SnapshotInfo{
			SnapshotID:		info.Name,
			Description:	info.Message,
			Tags:			info.Tags,

		}

		*snapshots = append(*snapshots, newInfo)
	}
	return
}

// ResetRegistry prompts all images to be pushed back into the docker registry
func (dao *ControlPlaneDao) ResetRegistry(_ model.EntityRequest, _ *int) (err error) {
	err = dao.facade.SyncRegistryImages(datastore.Get(), true)
	return
}

// RepairRegistry will try to recover the latest image of all service images
// from the docker registry and save it to the index.
func (dao *ControlPlaneDao) RepairRegistry(_ model.EntityRequest, _ *int) (err error) {
	err = dao.facade.RepairRegistry(datastore.Get())
	return
}

// ReadyDFS locks until it receives notice that the dfs is idle
func (dao *ControlPlaneDao) ReadyDFS(serviceID string, _ *int) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()

	err = dao.facade.ResetLock(ctx, serviceID)
	return
}

// TagSnapshot tags an existing snapshot with 1 or more strings
func (dao *ControlPlaneDao) TagSnapshot(request model.TagSnapshotRequest, newTagList *[]string) error {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()

	var err error
	*newTagList, err = dao.facade.TagSnapshot(request.SnapshotID, request.TagNames)
	return err
}

// RemoveSnapshotTags removes specific tags from an existing snapshot
func (dao *ControlPlaneDao) RemoveSnapshotTags(request model.TagSnapshotRequest, newTagList *[]string) error {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()

	var err error
	*newTagList, err = dao.facade.RemoveSnapshotTags(request.SnapshotID, request.TagNames)
	return err
}

// RemoveAllSnapshotTags removes all tags from an existing snapshot
func (dao *ControlPlaneDao) RemoveAllSnapshotTags(snapshotID string, _ *int) error {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock()
	defer dfslocker.Unlock()

	var err error
	err = dao.facade.RemoveAllSnapshotTags(snapshotID)
	return err
}
