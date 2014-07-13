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
			Command:     `DOCKER_REGISTRY_CONFIG=/docker-registry/config/config_sample.yml SETTINGS_FLAVOR=serviced docker-registry`,
			Ports:       []int{registryPort},
			Volumes:     map[string]string{"registry": "/tmp/registry"},
			StatusCheck: registryStatusCheck,
		},
	)
	if err != nil {
		glog.Fatalf("Error initializing docker-registry container: %s", err)
	}
}

func registryStatusCheck() error {

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
