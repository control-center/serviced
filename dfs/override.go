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
	"github.com/zenoss/glog"
)

// Override replaces an image in the docker registry with a new image
// and updates the registry.
func (dfs *DistributedFilesystem) Override(newimg, oldimg string) error {

	// make sure the old image exists
	oldImage, err := dfs.index.FindImage(oldimg)
	if err != nil {
		glog.Errorf("Could not find image %s in registry: %s", oldimg, err)
		return err
	}
	// // verify that we are committing to latest
	// if oldImage.Tag != docker.Latest {
	// 	return "", ErrStaleContainer
	// }

	// make sure the new image exists
	newImage, err := dfs.docker.FindImage(newimg)
	if err != nil {
		glog.Errorf("Could not find replacement image %s: %s", newimg, err)
		return err
	}

	// push the image into the registry
	hash, err := dfs.docker.GetImageHash(newImage.ID)
	if err != nil {
		glog.Errorf("Could not get has for image %s: %s", newimg, err)
		return err
	}

	if err := dfs.index.PushImage(oldImage.String(), newImage.ID, hash); err != nil {
		glog.Errorf("Could not push image %s (%s): %s", oldImage, newImage.ID, err)
		return err
	}
	return nil
}
