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
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

// Info returns information about an existing snapshot.
func (dfs *DistributedFilesystem) Info(snapshotID string) (*SnapshotInfo, error) {
	vol, info, err := dfs.getSnapshotVolumeAndInfo(snapshotID)
	if err != nil {
		return nil, err
	}
	r, err := vol.ReadMetadata(snapshotID, ImagesMetadataFile)
	if err != nil {
		glog.Errorf("Could not read images metadata from snapshot %s: %s", snapshotID, err)
		return nil, err
	}
	var images []string
	if err := importJSON(r, &images); err != nil {
		glog.Errorf("Could not interpret images metadata from snapshot %s: %s", snapshotID, err)
		return nil, err
	}
	r, err = vol.ReadMetadata(snapshotID, ServicesMetadataFile)
	if err != nil {
		glog.Errorf("Could not read services metadata from snapshot %s: %s", snapshotID, err)
		return nil, err
	}
	var svcs []service.Service
	if err := importJSON(r, &svcs); err != nil {
		glog.Errorf("Could not interpret services metadata from snapshot %s: %s", snapshotID, err)
		return nil, err
	}
	return &SnapshotInfo{info, images, svcs}, nil
}

// getSnapshotVolumeAndInfo returns the parent volume and info about a snapshot.
func (dfs *DistributedFilesystem) getSnapshotVolumeAndInfo(snapshotID string) (volume.Volume, *volume.SnapshotInfo, error) {
	vol, err := dfs.disk.GetTenant(snapshotID)
	if err != nil {
		glog.Errorf("Could not get tenant of snapshot %s: %s", snapshotID, err)
		return nil, nil, err
	}
	info, err := vol.SnapshotInfo(snapshotID)
	if err != nil {
		glog.Errorf("Could not get info for snapshot %s: %s", snapshotID, err)
		return nil, nil, err
	}
	return vol, info, nil
}
