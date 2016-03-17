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

// ResetRegistry moves all relevant images into the new docker registry
func (a *api) ResetRegistry() error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}
	return client.ResetRegistry()
}

// SyncRegistry walks the service tree and syncs all images from docker to local
// registry.
func (a *api) RegistrySync() error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}
	return client.SyncRegistry()
}

// UpgradeRegistry migrates images from an older or remote docker registry.
func (a *api) UpgradeRegistry(endpoint string, override bool) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}
	return client.UpgradeRegistry(endpoint, override)
}

// DockerOverride replaces an image in the docker registry with the specified image
func (a *api) DockerOverride(newImage string, oldImage string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}
	return client.DockerOverride(newImage, oldImage)
}
