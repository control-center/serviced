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
	"github.com/control-center/serviced/dfsnew/docker"
	"github.com/zenoss/glog"
)

// Rollback reverts an application to a previous snapshot.
func (dfs *DistributedFilesystem) Rollback(snapshotID string) error {
	vol, info, err := dfs.getSnapshotVolumeAndInfo(snapshotID)
	if err != nil {
		return err
	}
	// do all the images exist in the registry?
	r, err := vol.ReadMetadata(info.Label, ImagesMetadataFile)
	if err != nil {
		glog.Errorf("Could not receive images metadata from snapshot %s: %s", snapshotID, err)
		return err
	}
	var images []string
	if err := importJSON(r, &images); err != nil {
		glog.Errorf("Could not interpret images metadata file from snapshot %s: %s", snapshotID, err)
		return err
	}
	for _, image := range images {
		rImage, err := dfs.index.FindImage(image)
		if err != nil {
			glog.Errorf("Could not find image %s from snapshot %s: %s", image, snapshotID, err)
			return err
		}
		rImage.Tag = docker.Latest
		if err := dfs.index.PushImage(rImage.String(), rImage.UUID); err != nil {
			glog.Errorf("Could not update image %s from snapshot %s in the registry: %s", image, snapshotID, err)
			return err
		}
	}
	// TODO: remove nfs exports here
	// Right now the entirety of /var/volumes is shared on NFS, but it would
	// make more sense to create a directory /exports/serviced and then bind
	// mount in appication volumes individually.
	// https://access.redhat.com/documentation/en-US/Red_Hat_Enterprise_Linux/5/html/Deployment_Guide/s1-nfs-server-config-exports.html
	if err := dfs.net.Stop(); err != nil {
		glog.Errorf("Could not stop nfs server: %s", err)
		return err
	}
	defer dfs.net.Restart()
	if err := vol.Rollback(info.Label); err != nil {
		glog.Errorf("Could not rollback snapshot %s for tenant %s: %s", snapshotID, info.TenantID, err)
		return err
	}
	return nil
}
