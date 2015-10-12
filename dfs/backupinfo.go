// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dfs

import (
	"archive/tar"
	"encoding/json"
	"io"

	"github.com/zenoss/glog"
)

// BackupInfo provides metadata info about the contents of a backup
func (dfs *DistributedFilesystem) BackupInfo(r io.Reader) (*BackupInfo, error) {
	tarfile := tar.NewReader(r)
	for {
		header, err := tarfile.Next()
		if err != nil {
			glog.Errorf("Could not read backup: %s", err)
			return nil, err
		}
		if header.Name == BackupMetadataFile {
			var data BackupInfo
			if err := json.NewDecoder(tarfile).Decode(&data); err != nil {
				glog.Errorf("Could not load backup metadata: %s", err)
				return nil, err
			}
			return &data, nil
		}
	}
}
