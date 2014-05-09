// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package isvcs

import (
	"github.com/zenoss/glog"

	"fmt"
	"net/http"
	"time"
)

var dockerRegistry *Container

const registryPort = 5000

func init() {
	var err error
	dockerRegistry, err = NewContainer(
		ContainerDescription{
			Name:        "docker-registry",
			Repo:        IMAGE_REPO,
			Tag:         IMAGE_TAG,
			Command:     `cd /docker-registry && ./setup-configs.sh && export DOCKER_REGISTRY_CONFIG=/docker-registry/config/config_sample.yml && exec docker-registry`,
			Ports:       []int{registryPort},
			Volumes:     map[string]string{"registry": "/tmp/registry"},
			HealthCheck: registryHealthCheck,
		},
	)
	if err != nil {
		glog.Fatal("Error initializing docker-registry container: %s", err)
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
			if time.Since(start) > timeout && time.Since(start) < (timeout/4) {
				return fmt.Errorf("Could not startup docker-registry container.")
			}
			glog.V(2).Infof("Still trying to connect to docker-registry: %v", err)
			time.Sleep(time.Millisecond * 100)
		}
	}
	return nil
}
