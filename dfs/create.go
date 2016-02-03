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

import "github.com/zenoss/glog"

// Create initializes an application volume on the dfs
func (dfs *DistributedFilesystem) Create(tenantID string) error {
	glog.V(1).Infof("Creating volume for %s", tenantID)
	vol, err := dfs.disk.Create(tenantID)
	if err != nil {
		glog.Errorf("Could not create volume for tenant %s: %s", tenantID, err)
		return err
	}
	glog.V(1).Infof("Volume created for %s at %s", tenantID, vol.Path())
	if err := dfs.export(vol.Path()); err != nil {
		glog.Errorf("Could not export volume at %s: %s", vol.Path(), err)
		return err
	}
	return nil
}

// export exports a volume over a network file share
func (dfs *DistributedFilesystem) export(path string) error {
	if err := dfs.net.AddVolume(path); err != nil {
		glog.Errorf("Error notifying storage of new volume %s: %s", path, err)
		return err
	} else if err := dfs.net.Sync(); err != nil {
		glog.Errorf("Error syncing storage for volume %s: %s", path, err)
		return err
	}
	return nil
}
