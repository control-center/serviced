// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package master

import (
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

//GetVolumeStatus gets status information for the given volume or nil
func (c *Client) GetVolumeStatus() (*volume.Statuses, error) {
	response := &volume.Statuses{}
	if err := c.call("GetVolumeStatus", empty, response); err != nil {
		glog.V(2).Infof("\tcall to GetVolumeStatus returned error: %v", err)
		return nil, err
	}
	return response, nil
}
