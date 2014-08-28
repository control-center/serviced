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
	"fmt"
	"os"

	"github.com/docker/docker/registry"
)

func DockerLogin(username, password, email string) (string, error) {

	if username == "" && password == "" && email == "" {
		// Attempt login with .dockercfg file.
		configFile, err := registry.LoadConfig(os.Getenv("HOME"))
		if err != nil {
			return "", err
		}
		authconfig, ok := configFile.Configs[registry.IndexServerAddress()]
		if !ok {
			return "", fmt.Errorf("Error: Unable to login, no data for index server.")
		}
		status, err := registry.Login(&authconfig, registry.HTTPRequestFactory(nil))
		if err != nil {
			return "", err
		}
		return status, nil
	} else {
		// Attempt login with this function's auth params.
		authconfig := registry.AuthConfig{
			Username:      username,
			Email:         email,
			Password:      password,
			ServerAddress: registry.IndexServerAddress(),
		}
		status, err := registry.Login(&authconfig, registry.HTTPRequestFactory(nil))
		if err != nil {
			return "", err
		}
		return status, nil
	}

	return "", fmt.Errorf("Auth params don't make sense.")

}

func DockerIsLoggedIn() bool {
	_, err := DockerLogin("", "", "")
	if err != nil {
		return false
	}
	return true
}
