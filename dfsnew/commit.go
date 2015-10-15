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

	"github.com/control-center/serviced/dfsnew/docker"
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
	img, err := dfs.docker.CommitContainer(ctr.ID, ctr.Config.Image)
	if err != nil {
		glog.Errorf("Could not commit container %s: %s", ctr.ID, err)
		return "", err
	}
	// push the image into the registry
	if err := dfs.index.PushImage(rImage.String(), img.ID); err != nil {
		glog.Errorf("Could not push image %s (%s): %s", rImage, img.ID, err)
		return "", err
	}
	return rImage.Library, nil
}
