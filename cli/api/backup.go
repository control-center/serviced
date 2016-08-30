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
