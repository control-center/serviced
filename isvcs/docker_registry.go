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
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"

	"fmt"
	"net/http"
	"time"
)

var dockerRegistry *IService

const (
	registryPort     = 5000
	v1RegistryVolume = "registry"
	v2RegistryVolume = "v2"
)

func initDockerRegistry() {
	var err error

	defaultHealthCheck := healthCheckDefinition{
		healthCheck: registryHealthCheck,
		Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
		Timeout:     DEFAULT_HEALTHCHECK_TIMEOUT,
	}

	healthChecks := make([]map[string]healthCheckDefinition, 1)
	healthChecks[0] = make(map[string]healthCheckDefinition)
	healthChecks[0][DEFAULT_HEALTHCHECK_NAME] = defaultHealthCheck

	dockerPortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "", // docker registry should always be open
		HostPort:       registryPort,
	}
	command := `SETTINGS_FLAVOR=serviced exec /opt/registry/registry /opt/registry/registry-config.yml`

	dockerRegistry, err = NewIService(
		IServiceDefinition{
			ID:           DockerRegistryISVC.ID,
			Name:         "docker-registry",
			Repo:         IMAGE_REPO,
			Tag:          IMAGE_TAG,
			Command:      func() string { return command },
			PortBindings: []portBinding{dockerPortBinding},
			Volumes:      map[string]string{v2RegistryVolume: "/tmp/registry-dev"},
			HealthChecks: healthChecks,
			PreStart:     checkDockerRegistryPath,
		},
	)
	if err != nil {
		log.WithError(err).Fatal("Unable to initialize Docker registry internal service container")
	}
}

func checkDockerRegistryPath(svc *IService) error {
	// does the v2 path exist?
	v2Path := svc.getResourcePath(v2RegistryVolume)
	if _, err := os.Stat(v2Path); os.IsNotExist(err) {
		// if the v2 path does not exist, then check for the v1 path
		v1Path := svc.getResourcePath(v1RegistryVolume)
		if _, err := os.Stat(v1Path); os.IsNotExist(err) {
			// no v1 path, so nothing to migrate
			return nil
		} else if err != nil {
			log.WithFields(logrus.Fields{
				"v1path": v1Path,
			}).WithError(err).Error("Unable to look up v1 Docker registry path")
			return err
		}
		// is this a v1 path, or just a misnamed v2 path? (1.1 => 1.1.x
		// upgrade)
		if _, err := os.Stat(filepath.Join(v1Path, "docker/registry/v2")); os.IsNotExist(err) {
			// this is (probably) a v1 registry, so need to migrate at master
			// startup
			return nil
		} else if err != nil {
			log.WithFields(logrus.Fields{
				"v1path": v1Path,
			}).WithError(err).Error("Unable to verify v1 Docker registry path")
			return err
		}
		// this is a v2 registry, now we just need to move it into its new
		// location. (1.1 => 1.1.x upgrade)
		if err := os.Rename(v1Path, v2Path); err != nil {
			log.WithError(err).Error("Unable to migrate v2 registry")
			return err
		}
		log.Info("Migrated v2 Docker registry")
	} else if err != nil {
		log.WithFields(logrus.Fields{
			"v2path": v2Path,
		}).WithError(err).Error("Unable to look up v2 Docker registry path")
		return err
	}
	return nil
}

func registryHealthCheck(halt <-chan struct{}) error {
	url := fmt.Sprintf("http://localhost:%d/", registryPort)
	log := log.WithFields(logrus.Fields{
		"registryurl": url,
	})
	for {
		resp, err := http.Get(url)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err == nil {
			break
		} else {
			log.Debug("Unable to connect to Docker registry")
		}

		select {
		case <-halt:
			log.Debug("Stopped health checks for Docker registry")
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
	log.Debug("Docker registry checked in healthy")
	return nil
}
