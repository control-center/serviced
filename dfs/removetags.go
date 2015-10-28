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

// RemoveTags removes 1 or more tags from an existing snapshot
func (dfs *DistributedFilesystem) RemoveTags(snapshotID string, tagNames []string) ([]string, error) {
	vol, err := dfs.disk.GetTenant(snapshotID)
	if err != nil {
		glog.Errorf("Could not get tenant of snapshot %s: %s", snapshotID, err)
		return nil, err
	}

	newTags, err := vol.RemoveSnapshotTags(snapshotID, tagNames)
	if err != nil {
		glog.Errorf("Could not remove tags %v from snapshot %s: %s", tagNames, snapshotID, err)
		return nil, err
	}

	return newTags, nil
}