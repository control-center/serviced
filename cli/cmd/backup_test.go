// Copyright 2014 The Serviced Authors.
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

// +build unit

package cmd

import (
	"errors"
	"fmt"
	"path"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/utils"
)

const (
	PathNotFound = "PathNotFound"
	NilPath      = "NilPath"
	TooSmallPath = "TooSmallPath"
)

var DefaultBackupAPITest = BackupAPITest{}

var (
	ErrBackupFailed  = errors.New("backup failed")
	ErrRestoreFailed = errors.New("restore failed")
	ErrBackupPathTooSmall = errors.New("not enough space in backup path")
)

type BackupAPITest struct {
	api.API
}

func InitBackupAPITest(args ...string) {
	New(DefaultBackupAPITest, utils.TestConfigReader{}, MockLogControl{}).Run(args)
}

func InitBackupAPITestNoExit(args ...string) {
	c := New(DefaultBackupAPITest, utils.TestConfigReader{}, MockLogControl{})
	c.exitDisabled = true
	c.Run(args)
}

func (t BackupAPITest) Backup(dirpath string, excludes []string, force bool) (string, error) {
	switch dirpath {
	case PathNotFound:
		return "", ErrBackupFailed
	case NilPath:
		return "", nil
	case TooSmallPath:
		if force {
			return fmt.Sprintf("%s.tgz", path.Base(dirpath)), nil
		} else {
			return "", ErrBackupPathTooSmall
		}
	default:
		return fmt.Sprintf("%s.tgz", path.Base(dirpath)), nil
	}
}

func (t BackupAPITest) Restore(path string) error {
	switch path {
	case PathNotFound:
		return ErrRestoreFailed
	default:
		return nil
	}
}

func (t BackupAPITest) GetBackupEstimate(path string, _ []string) (*dao.BackupEstimate, error) {
	switch path{
	case TooSmallPath:
	case PathNotFound:
		return &dao.BackupEstimate{
			AvailableBytes:   1000,
			EstimatedBytes:   10000,
			AvailableString:  "1K",
			EstimatedString:  "10K",
			BackupPath:       path,
			AllowBackup:      false,
		}, nil
	default:
		return &dao.BackupEstimate{
			AvailableBytes:   1000000000,
			EstimatedBytes:   1000000,
			AvailableString:  "1G",
			EstimatedString:  "1M",
			BackupPath:       path,
			AllowBackup:      true,
		}, nil
	}
	return nil, nil
}

func ExampleServicedCli_cmdBackup_InvalidPath() {
	// Invalid path
	pipeStderr(func() { InitBackupAPITestNoExit("serviced", "backup", PathNotFound) })

	// Output:
	// backup failed
}

func ExampleServicedCli_cmdBackup_ReturnEmptyPath() {
	// Backup returns an empty file path
	pipeStderr(func() { InitBackupAPITestNoExit("serviced", "backup", NilPath) })

	// Output:
	// received nil path to backup file
}

func ExampleServicedCli_cmdBackup() {
	// Success
	InitBackupAPITest("serviced", "backup", "path/to/dir")

	// Output:
	// dir.tgz
}

func ExampleServicedCLI_CmdBackup_usage() {
	InitBackupAPITestNoExit("serviced", "backup")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    backup - Dump all templates and services to a tgz file
	//
	// USAGE:
	//    command backup [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced backup DIRPATH
	//
	// OPTIONS:
	//    --exclude '--exclude option --exclude option'	Subdirectory of the tenant volume to exclude from backup
	//    --check						check space, but do not do backup
	//    --force						attempt backup even if space check fails
}

func ExampleServicedCLI_CmdBackup_noforce() {
	// Backup called with not enough space
	InitBackupAPITestNoExit("serviced", "backup", TooSmallPath)

	// Output:
	// not enough space in backup path
}


func ExampleServicedCLI_CmdBackup_force() {
	// Backup called with not enough space, --force argument
	InitBackupAPITest("serviced", "backup", TooSmallPath, "--force")

	// Output:
	// TooSmallPath.tgz
}

func ExampleServicedCLI_CmdBackup_check() {
	// Backup called with check-only flag
	InitBackupAPITestNoExit("serviced", "backup", "path/to/dir", "--check")

	// Output:
	// Checking for space...
	// Okay to backup. Estimated space required: 1M, Available: 1G
	// Check only - not taking backup
}

func ExampleServicedCli_cmdRestore() {
	InitBackupAPITestNoExit("serviced", "restore", PathNotFound)
	InitBackupAPITest("serviced", "restore", "path/to/file")

	// Output:
}

func ExampleServicedCLI_CmdRestore_usage() {
	InitBackupAPITest("serviced", "restore")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    restore - Restore templates and services from a tgz file
	//
	// USAGE:
	//    command restore [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced restore FILEPATH
	//
	// OPTIONS:
}

