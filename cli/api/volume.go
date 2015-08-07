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

package api

import (
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

//func (a *api) GetVolumeStatus(volumeName string) (map[string]map[string]interface{}, error) {
func (a *api) GetVolumeStatus(volumeNames []string ) (*volume.Statuses, error) {
	glog.V(2).Infof("api.GetVolumeStatus(%v)\n", volumeNames)   // TODO: remove or add V level
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}
	response, err := client.GetVolumeStatus(volumeNames)
	if err != nil {
		glog.Errorf("Error from client.GetVolumeStatus(%v): %v", volumeNames, err)
		return nil, err
	}
	glog.V(2).Infof("api.GetVolumeStatus(): response from client.GetVolumeStatus(): %+v", response)   // TODO: remove or add V level
	return response, nil
}

func (a *api) ResizeVolume(tenantid string, size uint64) (*volume.Status, error) {
	glog.V(2).Infof("api.ResizeVolume(%s)", tenantid)
	client, err := a.connectMaster()
	if err != nil {
		glog.Errorf("Error connecting to Master: %v", err)    // TODO: remove or add V level
		return nil, err
	}
	//var vol volume.Volume
	status, err := client.ResizeVolume(tenantid, size)
	if (err != nil) {
		glog.Errorf("Error resizing volume: %v", err)  // TODO: remove or add V level
	}
	return status, err
}
