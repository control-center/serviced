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
	"github.com/control-center/serviced/dfs/docker"
	"github.com/zenoss/glog"
)

// Destroy destroys all application data from the dfs and docker registry
func (dfs *DistributedFilesystem) Destroy(tenantID string) error {
	// TODO: remove nfs exports here
	// Right now the entirety of /var/volumes is shared on NFS, but it would
	// make more sense to create a directory /exports/serviced and then bind
	// mount in appication volumes individually.
	// https://access.redhat.com/documentation/en-US/Red_Hat_Enterprise_Linux/5/html/Deployment_Guide/s1-nfs-server-config-exports.html
	vol, err := dfs.disk.Get(tenantID)
	if err != nil {
		glog.Errorf("Could not get volume for tenant %s: %s", tenantID, err)
		return err
	}
	snapshots, err := vol.Snapshots()
	if err != nil {
		glog.Errorf("Could not get snapshots for tenant %s: %s", tenantID, err)
		return err
	}
	for _, snapshot := range snapshots {
		if err := dfs.Delete(snapshot); err != nil {
			glog.Errorf("Could not remove snapshot %s for tenant %s: %s", snapshot, tenantID, err)
			return err
		}
	}
	if err := dfs.deleteImages(tenantID, docker.Latest); err != nil {
		return err
	}
	if err := dfs.net.Stop(); err != nil {
		glog.Errorf("Could not stop nfs server: %s", err)
		return err
	}
	defer dfs.net.Restart()
	if err := dfs.disk.Remove(tenantID); err != nil {
		glog.Errorf("Could not remove application data for tenant %s: %s", tenantID, err)
		return err
	}
	return nil
}
