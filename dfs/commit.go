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
	"github.com/control-center/serviced/domain/registry"
	"github.com/zenoss/glog"

	dockerclient "github.com/fsouza/go-dockerclient"
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
	// verify that we are committing to latest
	if rImage.Tag != docker.Latest || !dfs.imagesIdentical(rImage, ctr) {
		return "", ErrStaleContainer
	}
	// commit the container
	img, err := dfs.docker.CommitContainer(ctr.ID, ctr.Config.Image)
	if err != nil {
		glog.Errorf("Could not commit container %s: %s", ctr.ID, err)
		return "", err
	}
	// push the image into the registry
	hash, err := dfs.docker.GetImageHash(img.ID)
	if err != nil {
		glog.Errorf("Could not get has for image %s: %s", img.ID, err)
		return "", err
	}

	if err := dfs.index.PushImage(rImage.String(), img.ID, hash); err != nil {
		glog.Errorf("Could not push image %s (%s): %s", rImage, img.ID, err)
		return "", err
	}
	return rImage.Library, nil
}

func (dfs *DistributedFilesystem) imagesIdentical(rImage *registry.Image, ctr *dockerclient.Container) bool {
	// If image IDs are the same, we're done (ctr.Image is the UUID)
	if rImage.UUID == ctr.Image {
		return true
	}

	// If IDs do not match, the we have to compare image hashes
	if ctrHash, err := dfs.docker.GetImageHash(ctr.Image); err == nil {
		glog.V(2).Infof("For image %s, comparing hash (%s) to master's hash (%s)", ctr.ID, ctrHash, rImage.Hash)
		if ctrHash == rImage.Hash {
			return true
		}
	} else {
		glog.Warningf("Error building hash of container %s (image %s): %s", ctr.ID, ctr.Image, err)
	}
	return false
}

