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
	"errors"
	"io"
	"strings"

	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

var ErrRestoreNoInfo = errors.New("backup is missing metadata")

// Restore restores application data from a backup
func (dfs *DistributedFilesystem) Restore(r io.Reader) (*BackupInfo, error) {
	var data BackupInfo
	var foundBackupInfo bool
	var registryImages []string
	tarfile := tar.NewReader(r)
	glog.Infof("Loading backup data")
	for {
		header, err := tarfile.Next()
		if err == io.EOF {
			glog.Infof("Finished reading backup")
			if !foundBackupInfo {
				return nil, ErrRestoreNoInfo
			}
			if err := dfs.loadRegistry(registryImages); err != nil {
				return nil, err
			}
			return &data, nil
		} else if err != nil {
			glog.Errorf("Could not read backup: %s", err)
			return nil, err
		}
		switch {
		case header.Name == BackupMetadataFile:
			if err := json.NewDecoder(tarfile).Decode(&data); err != nil {
				glog.Errorf("Could not load backup metadata: %s", err)
				return nil, err
			}
			foundBackupInfo = true
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
			// Lets expedite this if this restore had already imported the snapshot
			// But delete the snapshot if it doesn't have the right information
			defer func() {
				if err != nil {
					vol.RemoveSnapshot(label)
				}
			}()
			if err = vol.Import(label, tarfile); err != nil && err != volume.ErrSnapshotExists {
				glog.Errorf("Could not import volume for tenant %s: %s", tenant, err)
				return nil, err
			}
			// Get all the images for this snapshot for the docker registry
			r, err := vol.ReadMetadata(label, ImagesMetadataFile)
			if err != nil {
				glog.Errorf("Could not receive images metadata from snapshot %s: %s", label, err)
				return nil, err
			}
			var images []string
			if err = importJSON(r, &images); err != nil {
				glog.Errorf("Could not interpret images metadata file from snapshot %s: %s", label, err)
				return nil, err
			}
			registryImages = append(registryImages, images...)
			glog.Infof("Loaded volume for tenant %s", tenant)
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

// loadRegistry reads snapshot images and pushes them into the registry with
// the correct registry labeling
func (dfs *DistributedFilesystem) loadRegistry(images []string) error {
	for _, image := range images {
		img, err := dfs.docker.FindImage(image)
		if err != nil {
			glog.Errorf("Could not load image %s into the registry: %s", image, err)
			return err
		}
		if err := dfs.index.PushImage(image, img.ID); err != nil {
			glog.Errorf("Could not push image %s into the registry: %s", image, err)
			return err
		}
	}
	return nil
}
