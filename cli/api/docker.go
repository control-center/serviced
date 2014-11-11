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

package api

import (
	"fmt"
	"os"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/commons/layer"
	"github.com/control-center/serviced/dao"
	"github.com/zenoss/glog"
)

// ResetRegistry moves all relevant images into the new docker registry
func (a *api) ResetRegistry() error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	return client.ResetRegistry(dao.NullRequest{}, new(int))
}

// Squash flattens the image (or at least down the to optional downToLayer).
// The resulting image is retagged with newName.
func (a *api) Squash(imageName, downToLayer, newName, tempDir string) (resultImageID string, err error) {

	client, err := a.connectDocker()
	if err != nil {
		return "", err
	}

	return layer.Squash(client, imageName, downToLayer, newName, tempDir)
}

// RegistrySync walks the service tree and syncs all images from docker to local registry
func (a *api) RegistrySync() (err error) {
	client, err := a.connectDocker()
	if err != nil {
		return err
	}

	services, err := a.GetServices()
	if err != nil {
		return err
	} else if services == nil || len(services) == 0 {
		return fmt.Errorf("no services found")
	}

	glog.V(2).Infof("RegistrySync from local docker repo %+v", client)
	synced := map[string]bool{}
	for _, svc := range services {
		if len(svc.ImageID) == 0 {
			continue
		}

		if _, ok := synced[svc.ImageID]; ok {
			continue
		}

		fmt.Fprintf(os.Stderr, "Syncing image to local docker registry: %s ...", svc.ImageID)
		if err := docker.PushImage(svc.ImageID); err != nil {
			return err
		}
		synced[svc.ImageID] = true
	}

	fmt.Printf("images synced to local docker registry")
	return nil
}
