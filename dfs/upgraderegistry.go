// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dfs

import (
	"fmt"

	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

// UpgradeRegistry loads images for each service into the docker registry
// index.  Also migrates images from a previous (or V1) registry at
// registryHost (host:port).
func (dfs *DistributedFilesystem) UpgradeRegistry(svcs []service.Service, tenantID, registryHost string, override bool) error {
	imageIDs := make(map[string]struct{})
	for _, svc := range svcs {
		if svc.ImageID == "" {
			// no image, no migration needed
			continue
		}
		image := svc.ImageID
		if _, ok := imageIDs[image]; ok {
			// image has already been added
			continue
		}
		imageIDs[image] = struct{}{}
		if !override {
			// is image in registry?
			rImage, err := dfs.findImage(image, tenantID)
			if err != nil {
				return err
			} else if rImage != "" {
				// image is already in the registry
				glog.V(2).Infof("Image %s for service %s (%s) already present in docker registry", rImage, svc.Name, svc.ID)
				continue
			}
		}
		// get registry image tag from image name
		rImage, err := dfs.parseRegistryImage(image, tenantID)
		if err != nil {
			glog.Warningf("Cannot parse image name %s under service %s (%s)", image, svc.Name, svc.ID)
			continue
		}
		// download image from old registry at registryHost defined at HOST:PORT
		// and retag it at the original registry path as defined by the service.
		if registryHost != "" {
			glog.Infof("Downloading image %s from %s registry", image, registryHost)
			oldImage := fmt.Sprintf("%s/%s", registryHost, rImage)
			if err := dfs.docker.PullImage(oldImage); err != nil {
				glog.Warningf("Could not pull image %s from registry %s, falling back to local library: %s", image, registryHost, err)
			} else if err := dfs.docker.TagImage(oldImage, image); err != nil {
				glog.Errorf("Could not retag image %s as %s: %s", oldImage, image, err)
				return err
			}
		}
		// find image in docker library
		img, err := dfs.docker.FindImage(image)
		if docker.IsImageNotFound(err) {
			glog.Warningf("Could not find image %s for service %s (%s)", image, svc.Name, svc.ID)
			continue
		} else if err != nil {
			glog.Errorf("Error looking up image %s for service %s (%s): %s", image, svc.Name, svc.ID, err)
			return err
		}

		hash, err := dfs.docker.GetImageHash(img.ID)
		if err != nil {
			glog.Errorf("Could not get hash for image %s: %s", img.ID, err)
			return err
		}

		// write to registry index
		if err := dfs.index.PushImage(rImage, img.ID, hash); err != nil {
			glog.Errorf("Could not write %s (%s) to registry index: %s", rImage, img.ID, err)
			return err
		}
		glog.Infof("Added image %s for service %s (%s) to the docker registry", rImage, svc.Name, svc.ID)
	}
	return nil
}
