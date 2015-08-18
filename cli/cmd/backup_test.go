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

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/utils"
)

const (
	PathNotFound = "PathNotFound"
	NilPath      = "NilPath"
)

var DefaultBackupAPITest = BackupAPITest{}

var (
	ErrBackupFailed  = errors.New("backup failed")
	ErrRestoreFailed = errors.New("restore failed")
)

type BackupAPITest struct {
	api.API
}

func InitBackupAPITest(args ...string) {
	New(DefaultBackupAPITest, utils.TestConfigReader{}).Run(args)
}

func (t BackupAPITest) Backup(dirpath string) (string, error) {
	switch dirpath {
	case PathNotFound:
		return "", ErrBackupFailed
	case NilPath:
		return "", nil
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

func ExampleServicedCli_cmdBackup() {
	// Invalid path
	InitBackupAPITest("serviced", "backup", PathNotFound)
	// Backup returns an empty file path
	InitBackupAPITest("serviced", "backup", NilPath)
	// Success
	InitBackupAPITest("serviced", "backup", "path/to/dir")

	// Output:
	// dir.tgz
}

func ExampleServicedCLI_CmdBackup_usage() {
	InitBackupAPITest("serviced", "backup")

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
}

func ExampleServicedCli_cmdRestore() {
	InitBackupAPITest("serviced", "restore", PathNotFound)
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
