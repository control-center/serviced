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
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfsnew/docker"
	"github.com/control-center/serviced/domain/registry"
	"github.com/zenoss/glog"
)

var (
	ErrImageCollision = errors.New("registry image collision")
)

// Download will download the image from upstream and save the image to the
// registry.
func (dfs *DistributedFilesystem) Download(image, tenantID string, upgrade bool) (string, error) {
	// Is this a proper docker image?
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return "", err
	}
	// Get the registry path
	rImage := (&registry.Image{
		Library: tenantID,
		Repo:    imageID.Repo,
		Tag:     docker.Latest,
	}).String()
	// Find (or download) the image
	img, err := dfs.docker.FindImage(image)
	if docker.IsImageNotFound(err) {
		glog.Infof("Image %s not found locally, pulling", image)
		if err := dfs.docker.PullImage(image); err != nil {
			glog.Errorf("Could not pull image %s: %s", image, err)
			return "", err
		} else if img, err = dfs.docker.FindImage(image); err != nil {
			glog.Errorf("Could not find image %s: %s", image, err)
			return "", err
		}
	} else if err != nil {
		glog.Errorf("Could not find image %s: %s", image, err)
		return "", err
	}
	// Compare the uuids of the topmost layer
	if rimg, err := dfs.index.FindImage(rImage); datastore.IsErrNoSuchEntity(err) {
		if err := dfs.index.PushImage(rImage, img.ID); err != nil {
			glog.Errorf("Could not push image %s into registry: %s", rImage, err)
			return "", err
		}
	} else if err != nil {
		glog.Errorf("Could not look up image %s: %s", rImage, err)
		return "", err
	} else if rimg.UUID != img.ID {
		if upgrade {
			if err := dfs.index.PushImage(rImage, img.ID); err != nil {
				glog.Errorf("Could not push image %s into registry: %s", rImage, err)
				return "", err
			}
		} else {
			return "", ErrImageCollision
		}
	}
	return rImage, nil
}
