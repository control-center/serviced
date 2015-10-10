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

// List returns the list of snapshots for a given tenant.
func (dfs *DistributedFilesystem) List(tenantID string) ([]string, error) {
	vol, err := dfs.disk.Get(tenantID)
	if err != nil {
		glog.Errorf("Could not get volume for tenant %s: %s", tenantID, err)
		return nil, err
	}
	snapshots, err := vol.Snapshots()
	if err != nil {
		glog.Errorf("Could not get snapshots for tenant %s: %s", tenantID, err)
		return nil, err
	}
	return snapshots, nil
}
