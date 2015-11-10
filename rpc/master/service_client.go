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

package master

import (
	"github.com/zenoss/glog"
)

type ServiceUseRequest struct {
	ServiceID string
	ImageID   string
	Registry  string
	NoOp      bool
}

// ServiceUse will use a new image for a given service - this will pull the image and tag it
func (c *Client) ServiceUse(serviceID string, imageID string, registry string, noOp bool) (string, error) {
	svcUseRequest := &ServiceUseRequest{ServiceID: serviceID, ImageID: imageID, Registry: registry, NoOp: noOp}
	imageResult := ""
	glog.Infof("Pulling %s, tagging to latest, and pushing to registry %s - this may take a while", imageID, registry)
	err := c.call("ServiceUse", svcUseRequest, &imageResult)
	if err != nil {
		return "", err
	}
	return imageResult, nil
}
