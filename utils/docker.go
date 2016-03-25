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

package utils

import (
	"github.com/zenoss/glog"

	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/registry"
)

func DockerIsLoggedIn() bool {

	// Load the user's ~/.docker/config.json file if it exists.
	configFile, err := cliconfig.Load("")
	if err != nil {
		glog.Errorf("Error checking Docker Hub login: %s", err)
		return false
	}

	// Make sure there is at least one AuthConfig (credential set).
	if len(configFile.AuthConfigs) < 1 {
		glog.Errorf("Error checking Docker Hub login: config.json is not populated")
		return false
	}

	// Iterate over AuthConfigs and attempt to login.
	svc := registry.NewService(registry.ServiceOptions{})
	for _, authConfig := range configFile.AuthConfigs {
		_, _, err := svc.Auth(&authConfig, "")
		if err == nil {
			return true
		}
	}

	glog.Errorf("Error checking Docker Hub login: no credentials in config.json succeeded")
	return false
}
