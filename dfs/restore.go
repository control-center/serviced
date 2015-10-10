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
	"strings"

	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

// Restore restores application data from a backup
func (dfs *DistributedFilesystem) Restore(r io.Reader) (*BackupInfo, error) {
	var data BackupInfo
	tarfile := tar.NewReader(r)
	glog.Infof("Loading backup data")
	for {
		header, err := tarfile.Next()
		if err == io.EOF {
			glog.Infof("Finished reading backup")
			return &data, nil
		}
		switch {
		case header.Name == BackupMetadataFile:
			if err := json.NewDecoder(tarfile).Decode(&data); err != nil {
				glog.Errorf("Could not load backup metadata: %s", err)
				return nil, err
			}
			glog.Infof("Loaded backup metadata")
		case strings.HasPrefix(header.Name, SnapshotsMetadataDir):
			parts := strings.Split(header.Name, "/")
			if len(parts) != 3 {
				continue
			}
			tenant, label := parts[1], parts[2]
			vol, err := dfs.disk.Create(tenant)
			if err == volume.ErrVolumeExists {
				if vol, err = dfs.disk.Get(tenant); err != nil {
					glog.Errorf("Could not get volume for tenant %s: %s", tenant, err)
					return nil, err
				}
			} else if err != nil {
				glog.Errorf("Could not create volume for tenant %s: %s", tenant, err)
				return nil, err
			}
			if err := vol.Import(label, tarfile); err != nil {
				glog.Errorf("Could not import volume for tenant %s: %s", tenant, err)
				return nil, err
			}
			glog.Infof("Loaded volume for tenant %s: %s", tenant, err)
		case header.Name == DockerImagesFile:
			if err := dfs.docker.LoadImage(tarfile); err != nil {
				glog.Errorf("Could not load images: %s", err)
				return nil, err
			}
			glog.Infof("Loaded images")
		default:
			glog.Warningf("Unrecognized file %s", header.Name)
		}
	}
}
