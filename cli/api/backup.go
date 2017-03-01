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

package api

import (
	"fmt"
	"path/filepath"

	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/dao"
	"github.com/dustin/go-humanize"
	"errors"
)


type BackupDetails struct {
	Available uint64
	EstimatedBytes uint64
	Estimated dao.BackupEstimate
	Path string
	Excludes []string
	Warn bool
	Message string
}

// Dump all templates and services to a tgz file.
// This includes a snapshot of all shared file systems
// and exports all docker images the services depend on.
func (a *api) Backup(dirpath string, excludes []string) (string, error) {
	client, err := a.connectDAO()
	if err != nil {
		return "", err
	}
	// TODO: (?) add check for space here (or just handle error from client.Backup call?)
	var path string
	req := dao.BackupRequest{
		Dirpath:              dirpath,
		SnapshotSpacePercent: config.GetOptions().SnapshotSpacePercent,
		Excludes:             excludes,
	}
	if err := client.Backup(req, &path); err != nil {
		return "", err
	}

	return path, nil
}

// Restores templates, services, snapshots, and docker images from a tgz file.
// This is the inverse of CmdBackup.
func (a *api) Restore(path string) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	fp, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("could not convert '%s' to an absolute file path: %v", path, err)
	}

	return client.Restore(filepath.Clean(fp), &unusedInt)
}


func (a *api) GetBackupEstimate(dirpath string, excludes []string) (*BackupDetails, error) {
	fmt.Printf("Hello, from GetBackupSpace()\n")
	client, err := a.connectDAO()
	fmt.Printf("Back from connectDAO(). err = %v\n", err)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error in connectDAO(): %v", err))
	}
	req := dao.BackupRequest{
		Dirpath:              dirpath,
		SnapshotSpacePercent: config.GetOptions().SnapshotSpacePercent,
		Excludes:             excludes,
	}
	est := dao.BackupEstimate{}
	if err := client.GetBackupEstimate(req, &est); err != nil {
		return nil, errors.New(fmt.Sprintf("error calling GetBackupestimate(): %v", err))
	}

	warn := est.EstimatedBytes > est.AvailableBytes
	message := ""
	if warn {
		message = fmt.Sprintf("Backup not recommended. Available space on %s is %s, and backup is estimated to take %s.", dirpath, humanize.Bytes(est.AvailableBytes), humanize.Bytes(est.EstimatedBytes))
	} else {
		message = fmt.Sprintf("There should be sufficient room for a backup. Free space on %s is %s, and the backup is estimated to take %s, which will leave %s", dirpath, humanize.Bytes(est.AvailableBytes), humanize.Bytes(est.EstimatedBytes), humanize.Bytes(est.AvailableBytes - est.EstimatedBytes))
	}
	deets := BackupDetails{
		Available: est.AvailableBytes,
		EstimatedBytes: est.EstimatedBytes,
		Estimated: est,
		Path: dirpath,
		Excludes: excludes,
		Warn: warn,
		Message: message,
	}

	return &deets, nil
}