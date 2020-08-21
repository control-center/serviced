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
	"bytes"
	"github.com/Sirupsen/logrus"
	"github.com/zenoss/glog"
	"os/exec"
	"regexp"

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

// Get the IP of the docker0 interface, which can be used to access the serviced API from inside the container.
// inspiration: the following is used for the same purpose during deploy/provision:
// ip addr show docker0 | grep inet | grep -v inet6 | awk '{print $2}' | awk -F / '{print $1}'
func GetDockerIP() string {
	// Execute 'ip -4 -br addr show docker0'
	c1 := exec.Command("ip", "-4", "-br", "addr", "show", "docker0")
	var out bytes.Buffer
	c1.Stdout = &out
	err := c1.Run()
	if err != nil {
		plog.WithField("command", c1).Infof("Error calling command: %s", err)
		return ""
	}
	outstr := out.String()
	// use a regex to extract the ip address from the result.
	// We're expecting something that looks like: ###.###.###.###/##
	// We use a capture group to exclude the trailing /##.
	re := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)/\d+`)
	addr := re.FindStringSubmatch(outstr)
	if addr != nil && len(addr) > 1 {
		return addr[1]
	}
	plog.WithFields(logrus.Fields{"match": addr, "output": outstr}).Info("Output was not as expected")
	return ""
}