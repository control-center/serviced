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

	"github.com/zenoss/glog"

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
	healthChecks := map[string]healthCheckDefinition{
		DEFAULT_HEALTHCHECK_NAME: defaultHealthCheck,
	}

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
		glog.Fatalf("Error initializing docker-registry container: %s", err)
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
			glog.Errorf("Error trying to look up v1 docker registry path at %s: %s", v1Path, err)
			return err
		}
		// is this a v1 path, or just a misnamed v2 path? (1.1 => 1.1.x
		// upgrade)
		if _, err := os.Stat(filepath.Join(v1Path, "docker/registry/v2")); os.IsNotExist(err) {
			// this is (probably) a v1 registry, so need to migrate at master
			// startup
			return nil
		} else if err != nil {
			glog.Errorf("Error trying to verify v1 docker registry path at %s: %s", v1Path, err)
			return err
		}
		// this is a v2 registry, now we just need to move it into its new
		// location. (1.1 => 1.1.x upgrade)
		if err := os.Rename(v1Path, v2Path); err != nil {
			glog.Errorf("Could not migrate v2 registry: %s", err)
			return err
		}
		glog.Infof("Successfully migrated v2 registry")
	} else if err != nil {
		glog.Errorf("Error trying to look up v2 docker registry path at %s: %s", v2Path, err)
		return err
	}
	return nil
}

func registryHealthCheck(halt <-chan struct{}) error {
	url := fmt.Sprintf("http://localhost:%d/", registryPort)
	for {
		resp, err := http.Get(url)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err == nil {
			break
		} else {
			glog.V(1).Infof("Still trying to connect to docker registry at %s: %v", url, err)
		}

		select {
		case <-halt:
			glog.V(1).Infof("Quit healthcheck for docker registry at %s", url)
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
	glog.V(1).Infof("docker registry running, browser at %s", url)
	return nil
}
