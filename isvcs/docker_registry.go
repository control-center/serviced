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

package isvcs

import (
	"github.com/zenoss/glog"

	"fmt"
	"net/http"
	"time"
)

var dockerRegistry *IService

const registryPort = 5000

func init() {
	var err error

	defaultHealthCheck := healthCheckDefinition{
		healthCheck: registryHealthCheck,
		Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
	}
	healthChecks := map[string]healthCheckDefinition{
		DEFAULT_HEALTHCHECK_NAME: defaultHealthCheck,
	}

	command := `DOCKER_REGISTRY_CONFIG=/docker-registry/config/config_sample.yml SETTINGS_FLAVOR=serviced exec docker-registry`
	dockerRegistry, err = NewIService(
		IServiceDefinition{
			Name:         "docker-registry",
			Repo:         IMAGE_REPO,
			Tag:          IMAGE_TAG,
			Command:      func() string { return command },
			Ports:        []uint16{registryPort},
			Volumes:      map[string]string{"registry": "/tmp/registry"},
			HealthChecks: healthChecks,
		},
	)
	if err != nil {
		glog.Fatalf("Error initializing docker-registry container: %s", err)
	}
}

func registryHealthCheck() error {

	start := time.Now()
	timeout := time.Second * 30
	url := fmt.Sprintf("http://localhost:%d/", registryPort)
	for {
		if _, err := http.Get(url); err == nil {
			break
		} else {
			if time.Since(start) > timeout {
				return fmt.Errorf("could not startup docker-registry container")
			}
		}
		time.Sleep(time.Second)
	}
	return nil
}
