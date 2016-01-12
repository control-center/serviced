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
	"encoding/json"
	"io"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

func importJSON(r io.ReadCloser, v interface{}) error {
	defer r.Close()
	return json.NewDecoder(r).Decode(v)
}

func exportJSON(w io.WriteCloser, v interface{}) error {
	defer w.Close()
	return json.NewEncoder(w).Encode(v)
}

func readSnapshotInfo(vol volume.Volume, info *volume.SnapshotInfo) (*SnapshotInfo, error) {
	// Retrieve the images metadata
	r, err := vol.ReadMetadata(info.Label, ImagesMetadataFile)
	if err != nil {
		glog.Errorf("Could not read images metadata from snapshot %s: %s", info.Label, err)
		return nil, err
	}
	var images []string
	if err := importJSON(r, &images); err != nil {
		glog.Errorf("Could not interpret images metadata from snapshot %s: %s", info.Label, err)
		return nil, err
	}
	// Retrieve services metadata
	r, err = vol.ReadMetadata(info.Label, ServicesMetadataFile)
	if err != nil {
		glog.Errorf("Could not read services metadata from snapshot %s: %s", info.Label, err)
		return nil, err
	}
	var svcs []service.Service
	if err := importJSON(r, &svcs); err != nil {
		glog.Errorf("Could not interpret services metadata from snapshot %s: %s", info.Label, err)
		return nil, err
	}
	return &SnapshotInfo{info, images, svcs}, nil
}
