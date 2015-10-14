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
	"time"

	"github.com/zenoss/glog"
)

const (
	ServicesMetadataFile = "./.snapshot/services.json"
	ImagesMetadataFile   = "./.snapshot/images.json"
)

// Snapshot saves the current state of a particular application
func (dfs *DistributedFilesystem) Snapshot(data SnapshotInfo) (string, error) {
	label := generateSnapshotLabel()
	vol, err := dfs.disk.Get(data.Info.TenantID)
	if err != nil {
		glog.Errorf("Could not get volume for tenant %s: %s", data.Info.TenantID, err)
		return "", err
	}
	// relabel all registry tags for this snapshot
	images := make([]string, len(data.Images))
	for i, image := range data.Images {
		rImage, err := dfs.index.FindImage(image)
		if err != nil {
			glog.Errorf("Could not find image %s for snapshot: %s", image, err)
			return "", err
		}
		rImage.Tag = label
		if err := dfs.index.PushImage(rImage.String(), rImage.UUID); err != nil {
			glog.Errorf("Could not retag image %s for snapshot: %s", image, err)
			return "", err
		}
		images[i] = rImage.String()
	}
	// write snapshot metadata
	w, err := vol.WriteMetadata(label, ImagesMetadataFile)
	if err != nil {
		glog.Errorf("Could not create image metadata file for tenant %s: %s", data.Info.TenantID, err)
		return "", err
	}
	if err := exportJSON(w, images); err != nil {
		glog.Errorf("Could not write service metadata file for tenant %s: %s", data.Info.TenantID, err)
		return "", err
	}
	w, err = vol.WriteMetadata(label, ServicesMetadataFile)
	if err != nil {
		glog.Errorf("Could not create service metadata file for tenant %s: %s", data.Info.TenantID, err)
		return "", err
	}
	if err := exportJSON(w, data.Services); err != nil {
		glog.Errorf("Could not write service metadata file for tenant %s: %s", data.Info.TenantID, err)
		return "", err
	}
	// snapshot the volume
	if err := vol.Snapshot(label, data.Info.Message, data.Info.Tags); err != nil {
		glog.Errorf("Could not snapshot volume for tenant %s: %s", data.Info.TenantID, err)
		return "", err
	}
	info, err := vol.SnapshotInfo(label)
	if err != nil {
		glog.Errorf("Could not get info for snapshot %s of tenant %s: %s", label, data.Info.TenantID, err)
		return "", err
	}
	return info.Name, nil
}

// generateSnapshotLabel creates a label for a snapshot
func generateSnapshotLabel() string {
	return time.Now().UTC().Format("20060102-150405.000")
}
