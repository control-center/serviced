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

	"path/filepath"
	"sync"
	"time"

	"errors"
	"github.com/Sirupsen/logrus"
	model "github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/volume"
	gzip "github.com/klauspost/pgzip"
)

var (
	log = logging.PackageLogger()
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
func (dao *ControlPlaneDao) Backup(backupRequest model.BackupRequest, filename *string) (err error) {
	ctx := datastore.Get()
	if len(backupRequest.Username) > 0 {
		ctx.SetUser(backupRequest.Username)
	}
	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock("backup")
	defer dfslocker.Unlock()

	if backupRequest.Dirpath == "" {
		backupRequest.Dirpath = dao.backupsPath
	}
	// CC-2421: Check for space before doing backup
	est := model.BackupEstimate{}
	err = dao.facade.EstimateBackup(ctx, backupRequest, &est)
	if err != nil {
		log.WithError(err).Error("Could not estimate backup size")
		return
	} else if !est.AllowBackup {
		estlog := log.WithFields(logrus.Fields{
			"estimatedsize":  est.EstimatedString,
			"destination":    est.BackupPath,
			"availablespace": est.AvailableString,
		})
		if backupRequest.Force {
			estlog.Warning("Backup not recommended, but proceeding because '-force' was specified")
		} else {
			message := fmt.Sprintf("Could not take backup - insufficient space on %s (%s estimated backup size, %s available)", est.BackupPath, est.EstimatedString, est.AvailableString)
			err := errors.New(message)
			estlog.WithError(err).Error("Could not take backup")
			return err
		}
	}

	// set the progress of the backup file
	*filename = time.Now().UTC().Format("backup-2006-01-02-150405.tgz")
	backupfilename := filepath.Join(backupRequest.Dirpath, *filename)

	inprogress.SetProgress(backupfilename, "backup")
	defer func() {
		if err != nil {
			log.WithError(err).Error("Backup failed with error")
			os.Remove(backupfilename)
		}
		inprogress.SetError(err)
	}()
	// create the file and write
	fh, err := os.Create(backupfilename)
	if err != nil {
		log.WithError(err).WithField("backupfilename", backupfilename).Error("Could not create backup file")
		return
	}
	defer fh.Close()
	w := gzip.NewWriter(fh)
	// CC-2292: Limit concurrency of backup gzipping
	// This setting will cause the writer to process up to 2 100KB blocks
	// at a time before the writer blocks. The default was 16 250KB blocks.
	// Smaller blocks will allow other goroutines to get time more frequently.
	w.SetConcurrency(100000, 2)
	defer w.Close()
	err = dao.facade.Backup(ctx, w, backupRequest.Excludes, backupRequest.SnapshotSpacePercent, backupfilename)
	return
}

func (dao *ControlPlaneDao) GetBackupEstimate(backupRequest model.BackupRequest, backupEstimate *model.BackupEstimate) (err error) {
	ctx := datastore.Get()
	start := time.Now()
	if backupRequest.Dirpath == "" {
		log.WithField("backupspath", dao.backupsPath).Info("Dirpath was empty. Using value from dao.BackupsPath")
		backupRequest.Dirpath = dao.backupsPath
	}
	err = dao.facade.EstimateBackup(ctx, backupRequest, backupEstimate)
	if err != nil {
		return err
	}
	log.WithFields(logrus.Fields{
		"elapsed":  time.Since(start),
		"estimate": backupEstimate,
	}).Info("Backup estimate done")

	return nil
}

// AsyncBackup is the same as backup, but asynchronous
func (dao *ControlPlaneDao) AsyncBackup(backupRequest model.BackupRequest, filename *string) (err error) {
	ctx := datastore.Get()
	if len(backupRequest.Username) > 0 {
		ctx.SetUser(backupRequest.Username)
	}
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock("backup")
	inprogress.Reset()
	dfslocker.Unlock()
	go dao.Backup(backupRequest, filename)
	return
}

// Restore restores the full application stack from a backup file.
func (dao *ControlPlaneDao) Restore(restoreRequest model.RestoreRequest, _ *int) (err error) {
	ctx := datastore.Get()
	if len(restoreRequest.Username) > 0 {
		ctx.SetUser(restoreRequest.Username)
	}
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock("restore")
	defer dfslocker.Unlock()
	inprogress.SetProgress(restoreRequest.Filename, "restore")
	defer func() {
		if err != nil {
			log.WithError(err).Error("Restore failed with error. Check DM_BASESIZE, " +
				"thin pool should have enough space for twice the size of each tenant device it supports")
		}
		inprogress.SetError(err)
	}()
	info, err := dfs.ExtractBackupInfo(restoreRequest.Filename)
	if err != nil {
		return err
	}
	fh, err := os.Open(restoreRequest.Filename)
	if err != nil {
		return err
	}
	defer fh.Close()
	gz, err := gzip.NewReader(fh)
	if err != nil {
		return err
	}
	defer gz.Close()
	err = dao.facade.Restore(ctx, gz, info, restoreRequest.Filename)
	return err
}

// AsyncRestore is the same as restore, but asynchronous.
func (dao *ControlPlaneDao) AsyncRestore(restoreRequest model.RestoreRequest, unused *int) (err error) {
	ctx := datastore.Get()
	if len(restoreRequest.Username) > 0 {
		ctx.SetUser(restoreRequest.Username)
	}
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock("restore")
	inprogress.Reset()
	dfslocker.Unlock()
	go dao.Restore(restoreRequest, unused)
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
	dfslocker.Lock("snapshot")
	defer dfslocker.Unlock()

	tagList := []string{}
	if len(req.Tag) > 0 {
		tagList = []string{req.Tag}
	}

	if req.ContainerID != "" {
		*snapshotID, err = dao.facade.Commit(ctx, req.ContainerID, req.Message, tagList, req.SnapshotSpacePercent)
	} else {
		*snapshotID, err = dao.facade.Snapshot(ctx, req.ServiceID, req.Message, tagList, req.SnapshotSpacePercent)
	}
	return
}

// Rollback reverts a single application to a particular state
func (dao *ControlPlaneDao) Rollback(req model.RollbackRequest, _ *int) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock("rollback")
	defer dfslocker.Unlock()
	err = dao.facade.Rollback(ctx, req.SnapshotID, req.ForceRestart)
	return
}

// DeleteSnapshot deletes a single snapshot
func (dao *ControlPlaneDao) DeleteSnapshot(snapshotID string, _ *int) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock("delete snapshot")
	defer dfslocker.Unlock()
	err = dao.facade.DeleteSnapshot(ctx, snapshotID)
	return
}

// DeleteSnapshots deletes all snapshots for a service
func (dao *ControlPlaneDao) DeleteSnapshots(serviceID string, _ *int) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock("delete snapshots")
	defer dfslocker.Unlock()
	err = dao.facade.DeleteSnapshots(ctx, serviceID)
	return
}

// ListSnapshots returns a list of all snapshots for a service
func (dao *ControlPlaneDao) ListSnapshots(serviceID string, snapshots *[]model.SnapshotInfo) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock("list snapshots")
	defer dfslocker.Unlock()

	*snapshots = make([]model.SnapshotInfo, 0)
	snapshotIDs, err := dao.facade.ListSnapshots(ctx, serviceID)
	if err != nil {
		return err
	}
	for _, snapshotID := range snapshotIDs {
		var newInfo model.SnapshotInfo

		info, err := dao.facade.GetSnapshotInfo(ctx, snapshotID)
		if err == volume.ErrInvalidSnapshot {
			newInfo = model.SnapshotInfo{
				SnapshotID: snapshotID,
				Invalid:    true,
			}

		} else if err != nil {
			return err
		} else {
			newInfo = model.SnapshotInfo{
				SnapshotID:  info.Name,
				TenantID:    info.TenantID,
				Description: info.Message,
				Tags:        info.Tags,
				Created:     info.Created,
				Invalid:     false,
			}
		}
		*snapshots = append(*snapshots, newInfo)
	}
	return
}

// ResetRegistry prompts all images to be pushed back into the docker registry
func (dao *ControlPlaneDao) ResetRegistry(_ model.EntityRequest, _ *int) (err error) {
	// Do not DFSLock here, Facade does that
	err = dao.facade.SyncRegistryImages(datastore.Get(), true)
	return
}

// RepairRegistry will try to recover the latest image of all service images
// from the docker registry and save it to the index.
func (dao *ControlPlaneDao) RepairRegistry(_ model.EntityRequest, _ *int) (err error) {
	// Do not DFSLock here, Facade does that
	err = dao.facade.RepairRegistry(datastore.Get())
	return
}

// ReadyDFS locks until it receives notice that the dfs is idle
func (dao *ControlPlaneDao) ReadyDFS(serviceID string, _ *int) (err error) {
	ctx := datastore.Get()

	// synchronize the dfs
	dfslocker := dao.facade.DFSLock(ctx)
	dfslocker.Lock("reset tenant lock")
	defer dfslocker.Unlock()

	err = dao.facade.ResetLock(ctx, serviceID)
	return
}

// TagSnapshot adds a tag to an existing snapshot
func (dao *ControlPlaneDao) TagSnapshot(request model.TagSnapshotRequest, _ *int) error {
	return dao.facade.TagSnapshot(request.SnapshotID, request.TagName)
}

// RemoveSnapshotTag removes a tag from an existing snapshot
func (dao *ControlPlaneDao) RemoveSnapshotTag(request model.SnapshotByTagRequest, snapshotID *string) (err error) {
	ctx := datastore.Get()
	*snapshotID, err = dao.facade.RemoveSnapshotTag(ctx, request.ServiceID, request.TagName)
	return err
}

// GetSnapshotByServiceIDAndTag Gets the snapshot from a specific service with a specific tag
func (dao *ControlPlaneDao) GetSnapshotByServiceIDAndTag(request model.SnapshotByTagRequest, snapshot *model.SnapshotInfo) (err error) {
	ctx := datastore.Get()
	info, err := dao.facade.GetSnapshotByServiceIDAndTag(ctx, request.ServiceID, request.TagName)
	if err != nil {
		return
	}
	*snapshot = model.SnapshotInfo{
		SnapshotID:  info.Name,
		TenantID:    info.TenantID,
		Description: info.Message,
		Tags:        info.Tags,
		Created:     info.Created,
	}
	return nil
}
