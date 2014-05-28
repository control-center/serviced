// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
// "fmt"
// dutils "github.com/dotcloud/docker/utils"
// "github.com/zenoss/glog"
// docker "github.com/zenoss/go-dockerclient"
)

const (
	DOCKER_ENDPOINT string = "unix:///var/run/docker.sock"
)

func (c *ControlPlaneDao) ListRemoteImages() (map[string][]string, error) {
	return make(map[string][]string), nil
}

func (c *ControlPlaneDao) PullRemoteImage(id string) error {
	return nil
}

func (c *ControlPlaneDao) RemoveRemoteImage(id string) error {
	return nil
}

func (c *ControlPlaneDao) PushImageToRemote(id string) error {
	return nil
}

func (c *ControlPlaneDao) PushTagToRemote(id string, tag string) error {
	return nil
}
