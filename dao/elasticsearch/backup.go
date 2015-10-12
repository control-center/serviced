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
	"compress/gzip"
	"os"
	"path/filepath"
	"sync"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"

	"time"
)

var backupLock sync.Mutex
var backupFile string
var backupError = make(chan error)

func (this *ControlPlaneDao) BackupStatus(_ int, message *string) error {
	return nil
}

// ListBackups lists the backup files in a given directory
func (this *ControlPlaneDao) ListBackups(dirname string, infos *[]dao.BackupFile) error {
	*infos = []dao.BackupFile{}

	// Read the contents of the directory
	dir, err := os.Open(dirname)
	if err != nil {
		return err
	}
	defer dir.Close()
	fis, err := dir.Readdir(0)
	for _, fi := range fis {
		if !fi.IsDir() {
			if fi.Name() == backupFile {
				bf := dao.BackupFile{
					InProgress: true,
					FullPath:   filepath.Join(dirname, fi.Name()),
					Name:       fi.Name(),
					Size:       fi.Size(),
					Mode:       fi.Mode(),
					ModTime:    fi.ModTime(),
				}
				*infos = append(*infos, bf)
				continue
			}
			// Try to load the backup info
			isbackup := func(filename string) bool {
				fh, err := os.Open(filepath.Join(dirname, filename))
				if err != nil {
					return false
				}
				defer fh.Close()
				gz, err := gzip.NewReader(fh)
				if err != nil {
					return false
				}
				_, err = this.facade.BackupInfo(datastore.Get(), gz)
				if err != nil {
					return false
				}
				return true
			}(fi.Name())
			if isbackup {
				bf := dao.BackupFile{
					InProgress: backupFile == fi.Name(),
					FullPath:   filepath.Join(dirname, fi.Name()),
					Name:       fi.Name(),
					Size:       fi.Size(),
					Mode:       fi.Mode(),
					ModTime:    fi.ModTime(),
				}
				*infos = append(*infos, bf)
			}
		}
	}
	return nil
}

// Backup saves templates, services, and snapshots into a tgz
func (this *ControlPlaneDao) Backup(dirname string, filename *string) error {
	backupLock.Lock()
	defer backupLock.Unlock()
	backupFile = time.Now().UTC().Format("backup-2006-01-02-150405.tgz")
	*filename = backupFile
	defer func() { backupFile = "" }()
	fh, err := os.Create(filepath.Join(dirname, *filename))
	if err != nil {
		return err
	}
	defer fh.Close()
	gz := gzip.NewWriter(fh)
	defer gz.Close()
	err = this.facade.Backup(datastore.Get(), gz)
	return err
}

// AsyncBackup performs the backup asynchronously
func (this *ControlPlaneDao) AsyncBackup(dirpath string, filename *string) error {
	go func() {
		err := this.Backup(dirpath, filename)
		backupError <- err
	}()
	return nil
}

// Restore restores a serviced installation to the state of its backup
func (this *ControlPlaneDao) Restore(filename string, _ *int) error {
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
	err = this.facade.Restore(datastore.Get(), gz)
	return err
}

// AsyncRestore performs the restore aynchronously
func (this *ControlPlaneDao) AsyncRestore(filename string, _ *int) error {
	go func() {
		err := this.Restore(filename, nil)
		backupError <- err
	}()
	return nil
}
