// Copyright 2015 The Serviced Authors.
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

// UpgradeDockerRequest are options for upgrading/migrating the docker registry.
type UpgradeDockerRequest struct {
	Endpoint string
	Override bool
}

// DockerOverrideRequest are options for replacing an image in the docker registry
type DockerOverrideRequest struct {
	OldImage string
	NewImage string
}

// ResetRegistry pulls latest from the running docker registry and updates the
// index.
func (c *Client) ResetRegistry() error {
	return c.call("ResetRegistry", struct{}{}, new(int))
}

// SyncRegistry sends a signal to the master to repush all images into the
// docker registry.
func (c *Client) SyncRegistry() error {
	return c.call("SyncRegistry", struct{}{}, new(int))
}

// UpgradeRegistry migrates images from an older or remote docker registry and
// updates the index.
func (c *Client) UpgradeRegistry(endpoint string, override bool) error {
	req := UpgradeDockerRequest{endpoint, override}
	return c.call("UpgradeRegistry", req, new(int))
}

// DockerOverride replaces an image in the registry with a new image
func (c *Client) DockerOverride(newImage, oldImage string) error {
	req := DockerOverrideRequest{
		OldImage: oldImage,
		NewImage: newImage,
	}
	return c.call("DockerOverride", req, new(int))
}
