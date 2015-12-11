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

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/dfs/docker"
	index "github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"
)

var (
	ErrImageCollision = errors.New("registry image collision")
)

// Download will download the image from upstream and save the image to the
// registry.
func (dfs *DistributedFilesystem) Download(image, tenantID string, upgrade bool) (string, error) {
	if !upgrade {
		// Is this an image that has already been deployed?
		if rImage, err := dfs.findImage(image, tenantID); err != nil {
			return "", err
		} else if rImage != "" {
			return rImage, nil
		}
	}
	// Figure out what the image will be in the docker registry
	rImage, err := dfs.parseRegistryImage(image, tenantID)
	if err != nil {
		return "", err
	}
	// Pull the image (if it doesn't exist locally)
	img, err := dfs.pullImage(image)
	if err != nil {
		return "", err
	}

	hash, err := dfs.docker.GetImageHash(img.ID)
	if err != nil {
		glog.Errorf("Could not get hash for image %s: %s", img.ID, err)
		return "", err
	}

	rimg, err := dfs.index.FindImage(rImage)
	if err == index.ErrImageNotFound {
		// Image does not exist in the registry, so push
		if err := dfs.index.PushImage(rImage, img.ID, hash); err != nil {
			glog.Errorf("Could not push image %s into registry: %s", rImage, err)
			return "", err
		}
		return rImage, nil
	} else if err != nil {
		glog.Errorf("Could not look up image %s from the registry: %s", rImage, err)
		return "", err
	}
	// Compare the uuids of the topmost layer
	if rimg.UUID != img.ID {
		if upgrade {
			// We are upgrading the image, so overwrite the existing tag with
			// the new UUID.
			if err := dfs.index.PushImage(rImage, img.ID, hash); err != nil {
				glog.Errorf("Could not upgrade image %s into registry: %s", rImage, err)
				return "", err
			}
		} else {
			return "", ErrImageCollision
		}
	}
	return rImage, nil
}

// findImage will verify whether the image has already been deployed with the
// application.
func (dfs *DistributedFilesystem) findImage(image, tenantID string) (string, error) {
	// Is the image in the index?
	rImage, err := dfs.index.FindImage(image)
	if err != nil {
		if err == index.ErrImageNotFound {
			return "", nil
		}
		glog.Errorf("Could not look up image %s from the registry: %s", image, err)
		return "", err
	}
	// Is this image latest under the same tenant?
	if rImage.Library == tenantID && rImage.Tag == docker.Latest {
		return rImage.String(), nil
	}
	return "", nil
}

// parseRegistryImage formats the image as it will be written into the docker
// registry.
func (dfs *DistributedFilesystem) parseRegistryImage(image, tenantID string) (string, error) {
	// Is this a proper docker image?
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		glog.Errorf("Could not parse image %s: %s", image, err)
		return "", err
	}
	// Get the registry path
	rImage := (&registry.Image{
		Library: tenantID,
		Repo:    imageID.Repo,
		Tag:     docker.Latest,
	}).String()
	return rImage, nil
}

// pullImage pulls the image from the upstream if it doesn't already exist
// locally
func (dfs *DistributedFilesystem) pullImage(image string) (*dockerclient.Image, error) {
	// Find (or download) the image
	img, err := dfs.docker.FindImage(image)
	if docker.IsImageNotFound(err) {
		glog.Infof("Image %s not found locally, pulling", image)
		if err := dfs.docker.PullImage(image); err != nil {
			glog.Errorf("Could not pull image %s: %s", image, err)
			return nil, err
		} else if img, err = dfs.docker.FindImage(image); err != nil {
			glog.Errorf("Could not find image %s: %s", image, err)
			return nil, err
		}
	} else if err != nil {
		glog.Errorf("Could not find image %s: %s", image, err)
		return nil, err
	}
	return img, nil
}
