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
	"errors"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
	"time"
)

const (
	ServicesMetadataFile = "./.snapshot/services.json"
	ImagesMetadataFile   = "./.snapshot/images.json"
)

var (
	ErrDFSStatusUnavailable = errors.New("Unable to get storage status for dfs root")
)

// Snapshot saves the current state of a particular application
func (dfs *DistributedFilesystem) Snapshot(data SnapshotInfo, spaceFactor int) (string, error) {
	label := generateSnapshotLabel()
	vol, err := dfs.disk.Get(data.TenantID)

	if volume.DriverTypeDeviceMapper == dfs.disk.DriverType() {
		freeSpace, err := ensureFreeSpace(vol, dfs, spaceFactor)
		if err != nil {
			glog.Errorf("Could not determine freespace on devicemapper device %s", err)
			return "", err
		}
		if !freeSpace {
			return "", errors.New("There is not enough diskspace to complete your request. You should enlarge your thin pool using LVM tools and/or delete some snapshots")
		}
	}
	if err != nil {
		glog.Errorf("Could not get volume for tenant %s: %s", data.TenantID, err)
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

		// make sure we actually have the image locally before changing the tag and triggering a push
		if _, err := dfs.reg.FindImage(rImage); err != nil {
			glog.Errorf("Could not find image %s locally for snapshot:  %s", rImage.String(), err)
			return "", err
		}

		rImage.Tag = label
		if err := dfs.index.PushImage(rImage.String(), rImage.UUID, rImage.Hash); err != nil {
			glog.Errorf("Could not retag image %s for snapshot: %s", image, err)
			return "", err
		}
		fullImagePath, err := dfs.reg.ImagePath(rImage.String())
		if err != nil {
			glog.Errorf("Could not get the full image path for image %s: %s", image, err)
			return "", err
		}
		images[i] = fullImagePath
	}
	// write snapshot metadata
	w, err := vol.WriteMetadata(label, ImagesMetadataFile)
	if err != nil {
		glog.Errorf("Could not create image metadata file for tenant %s: %s", data.TenantID, err)
		return "", err
	}
	if err := exportJSON(w, images); err != nil {
		glog.Errorf("Could not write service metadata file for tenant %s: %s", data.TenantID, err)
		return "", err
	}
	w, err = vol.WriteMetadata(label, ServicesMetadataFile)
	if err != nil {
		glog.Errorf("Could not create service metadata file for tenant %s: %s", data.TenantID, err)
		return "", err
	}
	if err := exportJSON(w, data.Services); err != nil {
		glog.Errorf("Could not write service metadata file for tenant %s: %s", data.TenantID, err)
		return "", err
	}
	// snapshot the volume
	if err := vol.Snapshot(label, data.Message, data.Tags); err != nil {
		glog.Errorf("Could not snapshot volume for tenant %s: %s", data.TenantID, err)
		return "", err
	}
	info, err := vol.SnapshotInfo(label)
	if err != nil {
		glog.Errorf("Could not get info for snapshot %s of tenant %s: %s", label, data.TenantID, err)
		return "", err
	}
	return info.Name, nil
}

// generateSnapshotLabel creates a label for a snapshot
func generateSnapshotLabel() string {
	return time.Now().UTC().Format("20060102_150405.000")
}

// checks to see if there is enough free space on volume to perform a snapshot
func ensureFreeSpace(vol volume.Volume, dfs *DistributedFilesystem, snapshotSpacePercent int) (bool, error) {
	status := volume.GetStatus()
	statusMap, found := status.DeviceMapperStatusMap[dfs.disk.Root()]
	if !found {
		return false, ErrDFSStatusUnavailable
	}
	var amountNeeded float64
	foundTenant := false
	for i := 0; i < len(statusMap.Tenants); i++ {
		currentTenant := statusMap.Tenants[i]
		if currentTenant.TenantID == vol.Tenant() {
			amountNeeded = float64(currentTenant.FilesystemUsed) * float64(snapshotSpacePercent/100)
			foundTenant = true
		}
	}
	if !foundTenant {
		return false, errors.New("Unable to find storage information for volume")
	}
	if amountNeeded > float64(statusMap.PoolDataAvailable) {
		return false, nil
	}
	return true, nil
}
