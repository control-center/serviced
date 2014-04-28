package cmd

import (
	"errors"
	"fmt"
	"path"

	"github.com/zenoss/serviced/serviced/api"
)

const (
	NOT_FOUND = "PATH_NOT_FOUND"
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
	case NOT_FOUND:
		return "", ErrBackupFailed
	default:
		return fmt.Sprintf("%s.tgz", path.Base(dirpath)), nil
	}
}

func (t BackupAPITest) Restore(path string) error {
	switch path {
	case NOT_FOUND:
		return ErrRestoreFailed
	default:
		return nil
	}
}

func ExampleServicedCli_cmdBackup() {
	InitBackupAPITest("serviced", "backup", NOT_FOUND)
	InitBackupAPITest("serviced", "backup", "path/to/dir")

	// Output:
	// dir.tgz
}

func ExampleServicedCli_cmdRestore() {
	InitBackupAPITest("serviced", "restore", NOT_FOUND)
	InitBackupAPITest("serviced", "restore", "path/to/file")

	// Output:
}