package cmd

import (
	"errors"
	"fmt"
	"path"

	"github.com/zenoss/serviced/cli/api"
)

const (
	PathNotFound = "PATH_PathNotFound"
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
	InitBackupAPITest("serviced", "backup", PathNotFound)
	InitBackupAPITest("serviced", "backup", "path/to/dir")

	// Output:
	// dir.tgz
}

func ExampleServicedCli_cmdRestore() {
	InitBackupAPITest("serviced", "restore", PathNotFound)
	InitBackupAPITest("serviced", "restore", "path/to/file")

	// Output:
}