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
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
	"time"
)

type ServiceUseRequest struct {
	ServiceID   string
	ImageID     string
	ReplaceImgs []string
	Registry    string
	NoOp        bool
}

type WaitServiceRequest struct {
	ServiceIDs []string
	State      service.DesiredState
	Timeout    time.Duration
	Recursive  bool
}

// ServiceUse will use a new image for a given service - this will pull the image and tag it
func (c *Client) ServiceUse(serviceID string, imageID string, registry string, replaceImgs []string, noOp bool) (string, error) {
	svcUseRequest := &ServiceUseRequest{ServiceID: serviceID, ImageID: imageID, ReplaceImgs: replaceImgs, Registry: registry, NoOp: noOp}
	result := ""
	glog.Infof("Pulling %s, tagging to latest, and pushing to registry %s - this may take a while", imageID, registry)
	err := c.call("ServiceUse", svcUseRequest, &result)
	if err != nil {
		return "", err
	}
	return result, nil
}

// WaitService will wait for the specified services to reach the specified
// state, within the given timeout
func (c *Client) WaitService(serviceIDs []string, state service.DesiredState, timeout time.Duration, recursive bool) error {
	waitSvcRequest := &WaitServiceRequest{
		ServiceIDs: serviceIDs,
		State:      state,
		Timeout:    timeout,
		Recursive:  recursive,
	}
	throwaway := ""
	err := c.call("WaitService", waitSvcRequest, &throwaway)
	return err
}
