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
	"errors"

	"github.com/control-center/serviced/dfs/docker"
	"github.com/zenoss/glog"
)

var (
	ErrRunningContainer = errors.New("container is running")
	ErrStaleContainer   = errors.New("container is stale")
)

// Commit commits a container spawned from the latest docker registry image
// and updates the registry.  Returns the affected registry image.
func (dfs *DistributedFilesystem) Commit(ctrID string) (string, error) {
	ctr, err := dfs.docker.FindContainer(ctrID)
	if err != nil {
		glog.Errorf("Could not find container %s: %s", ctrID, err)
		return "", err
	}
	// do not commit if the container is running
	if ctr.State.Running {
		return "", ErrRunningContainer
	}
	// check if the container is stale (ctr.Config.Image is the repo:tag)
	rImage, err := dfs.index.FindImage(ctr.Config.Image)
	if err != nil {
		glog.Errorf("Could not find image %s in registry for container %s: %s", ctr.Config.Image, ctr.ID, err)
		return "", err
	}
	// verify that we are committing to latest (ctr.Image is the UUID)
	if rImage.Tag != docker.Latest || rImage.UUID != ctr.Image {
		return "", ErrStaleContainer
	}
	// commit the container
	oldID := rImage.UUID
	img, err := dfs.docker.CommitContainer(ctr.ID, ctr.Config.Image)
	if err != nil {
		glog.Errorf("Could not commit container %s: %s", ctr.ID, err)
		return "", err
	}
	// push the image into the registry
	if (img.ID != oldID) { //If the commit produced a new ID, call PushImageAfterCommit, which will actually push the image twice to address an issue with imageIDs changing in docker 1.9.1
		if err := dfs.index.PushImageAfterCommit(rImage.String(), img.ID); err != nil {
			glog.Errorf("Could not push image %s (%s), re-pushing previous image ID: %s", rImage, img.ID, err)
			//If the push failed, we need to re-tag and re-push the old ID to avoid a mismatch between the master and agents
			if err2 := dfs.index.PushImage(rImage.String(), oldID); err2 != nil {
				glog.Errorf("Could not re-push old image %s (%s): %s", rImage.String(), oldID, err2)
				return "", err2
			}

			//try to delete the committed image
			if err2 := dfs.docker.RemoveImage(img.ID); err2 != nil {
				glog.Warningf("Could not clean up committed image %s: %s", img.ID, err2)
			}

			return "", err
		}
	} else {
		if err := dfs.index.PushImage(rImage.String(), img.ID); err != nil {
			glog.Errorf("Could not push image %s (%s): %s", rImage, img.ID, err)
			return "", err
		}
	}
	return rImage.Library, nil
}
