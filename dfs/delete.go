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

import "github.com/control-center/serviced/volume"
import "github.com/zenoss/glog"

// Delete removes application data of a particular snapshot from the dfs and
// registry.
func (dfs *DistributedFilesystem) Delete(snapshotID string) error {
	vol, err := dfs.disk.GetTenant(snapshotID)
	if err != nil {
		return err
	}

	if info, err := vol.SnapshotInfo(snapshotID); err != volume.ErrInvalidSnapshot {
		if err != nil {
			return err
		} else if err := dfs.deleteImages(info.TenantID, info.Label); err != nil {
			return err
		}
	}

	if err := vol.RemoveSnapshot(snapshotID); err != nil {
		glog.Errorf("Could not delete snapshot %s: %s", snapshotID, err)
		return err
	}

	return nil
}

func (dfs *DistributedFilesystem) deleteImages(tenantID, label string) error {
	rImages, err := dfs.index.SearchLibraryByTag(tenantID, label)
	if err != nil {
		glog.Errorf("Could not search registry images for %s under label %s: %s", tenantID, label, err)
		return err
	}
	for _, image := range rImages {
		if err := dfs.index.RemoveImage(image.String()); err != nil {
			glog.Errorf("Could not remove image %s for %s under label %s: %s", image.String(), tenantID, label, err)
			return err
		}
	}
	return nil
}
