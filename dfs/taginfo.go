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

// TagInfo returns information about an existing snapshot referenced by tag.
func (dfs *DistributedFilesystem) TagInfo(tenantID, tagName string) (*SnapshotInfo, error) {
	vol, err := dfs.disk.Get(tenantID)
	if err != nil {
		glog.Errorf("Could not get tenant volume %s: %s", tenantID, err)
		return nil, err
	}
	info, err := vol.GetSnapshotWithTag(tagName)
	if err != nil {
		glog.Errorf("Could not get info for snapshot with tag %s: %s", tagName, err)
		return nil, err
	}
	return readSnapshotInfo(vol, info)
}
