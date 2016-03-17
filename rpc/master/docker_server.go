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

// ResetRegistry pulls from the configured docker registry and updates the
// index.
func (s *Server) ResetRegistry(req struct{}, reply *int) error {
	return s.f.RepairRegistry(s.context())
}

// SyncRegistry prompts the master to repush all images in the index into the
// docker registry.
func (s *Server) SyncRegistry(req struct{}, reply *int) error {
	return s.f.SyncRegistryImages(s.context(), true)
}

// UpgradeRegistry migrates docker registry images from an older or remote
// docker registry.
func (s *Server) UpgradeRegistry(req UpgradeDockerRequest, reply *int) error {
	return s.f.UpgradeRegistry(s.context(), req.Endpoint, req.Override)
}

// DockerOverride replaces an image in the registry with a new image
func (s *Server) DockerOverride(overrideReq DockerOverrideRequest, _ *int) error {
	return s.f.DockerOverride(s.context(), overrideReq.NewImage, overrideReq.OldImage)
}
