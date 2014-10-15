// Copyright 2014 The Serviced Authors.
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
	"github.com/zenoss/glog"

	"fmt"

	"time"
)

var backupError = make(chan error)

// Backup saves templates, services, and snapshots into a tgz file
func (this *ControlPlaneDao) Backup(dirpath string, filename *string) error {
	this.dfs.Lock()
	defer this.dfs.Unlock()
	var err error
	*filename, err = this.dfs.Backup(dirpath)
	return err
}

// AsyncBackup performs the backup asynchronously
func (this *ControlPlaneDao) AsyncBackup(dirpath string, filename *string) error {
	// TODO: There is a risk of contention here if two backup operations are
	// called simultaneously. We may want to move backups into a leader queue
	// on the coordinator.
	if locked, err := this.dfs.IsLocked(); err != nil {
		glog.Errorf("Could not check lock on dfs: %s", err)
		return err
	} else if locked {
		err := fmt.Errorf("another operation is running")
		glog.Errorf("Another DFS operation is running, please wait and try again.")
		return err
	}

	go func() {
		err := this.Backup(dirpath, filename)
		backupError <- err
	}()

	return nil
}

// Restore restores a serviced installation to the state of its backup
func (this *ControlPlaneDao) Restore(filename string, unused *int) error {
	this.dfs.Lock()
	defer this.dfs.Unlock()
	return this.dfs.Restore(filename)
}

// AsyncRestore performs the restore aynchronously
func (this *ControlPlaneDao) AsyncRestore(filename string, unused *int) error {
	// TODO: There is a risk of contention here if two backup operations are
	// called simultaneously. We may want to move backups into a leader queue
	// on the coordinator.
	if locked, err := this.dfs.IsLocked(); err != nil {
		glog.Errorf("Could not check lock on dfs: %s", err)
		return err
	} else if locked {
		err := fmt.Errorf("another operation is running")
		glog.Errorf("Another DFS operation is running, please wait and try again.")
		return err
	}

	go func() {
		err := this.Restore(filename, unused)
		backupError <- err
	}()
	return nil
}

// BackupStatus monitors the status of a backup or restore
func (this *ControlPlaneDao) BackupStatus(unused int, status *string) error {
	message := make(chan string)
	go func() { message <- this.dfs.GetStatus(10 * time.Second) }()
	select {
	case *status = <-message:
	case err := <-backupError:
		return err
	}
	return nil
}
