package cmd

import (
	"errors"
	"fmt"
	"path"

	"github.com/zenoss/serviced/cli/api"
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
	New(DefaultBackupAPITest).Run(args)
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
