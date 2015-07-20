// Copyright 2014 The Serviced Authors.
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

package docker

import (
	"fmt"

	"github.com/control-center/serviced/commons"
	"github.com/zenoss/glog"
)

// ServiceUse will tag a new image (imageName) in a given registry for a given tenant
// to latest, making sure to push changes to the registry
func ServiceUse(serviceID string, imageName string, registry string, noOp bool) (string, error) {
	// If noOp is True, then replace the 'real' functions that talk to Docker with
	// no-op functions (for dry run purposes)
	pullImage := PullImage
	findImage := FindImage
	tagImage := TagImage
	if noOp {
		pullImage = noOpPullImage
		findImage = noOpFindImage
		tagImage = noOpTagImage
	}

	// imageName is the new image to pull, eg. "zenoss/resmgr-unstable:1.2.3.4"
	glog.V(0).Infof("preparing to use image: %s", imageName)
	imageID, err := commons.ParseImageID(imageName)
	if err != nil {
		return "", err
	}
	if imageID.Tag == "" {
		imageID.Tag = "latest"
	}
	glog.Infof("pulling image %s, this may take a while...", imageID)
	if err := pullImage(imageID.String()); err != nil {
		glog.Warningf("unable to pull image %s", imageID)
	}

	//verify image has been pulled
	img, err := findImage(imageID.String(), false)
	if err != nil {
		err = fmt.Errorf("could not look up image %s: %s. Check your docker login and retry service deployment.", imageID, err)
		return "", err
	}

	//Tag images to latest all images
	var newTag *commons.ImageID

	newTag, err = commons.RenameImageID(registry, serviceID, imageID.String(), "latest")
	if err != nil {
		return "", err
	}
	glog.Infof("tagging image %s to %s ", imageName, newTag)
	if _, err = tagImage(img, newTag.String(), true); err != nil {
		glog.Errorf("could not tag image: %s (%v)", imageName, err)
		return "", err
	}
	return newTag.String(), nil
}

func noOpTagImage(img *Image, tag string, push bool) (*Image, error) {
	return nil, nil
}

func noOpFindImage(repotag string, pull bool) (*Image, error) {
	return nil, nil
}

func noOpPullImage(repotag string) error {
	return nil
}
