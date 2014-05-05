package api

import (
	"fmt"
	"path/filepath"
)

// Dump all templates and services to a tgz file.
// This includes a snapshot of all shared file systems
// and exports all docker images the services depend on.
func (a *api) Backup(dirpath string) (string, error) {
	client, err := a.connectDAO()
	if err != nil {
		return "", err
	}

	var path string
	if err := client.Backup(dirpath, &path); err != nil {
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
