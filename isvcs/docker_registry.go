// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package isvcs

import (
	"github.com/zenoss/glog"
)

var dockerRegistry *Container

func init() {
	var err error
	dockerRegistry, err = NewContainer(
		ContainerDescription{
			Name:        "docker-registry",
			Repo:        IMAGE_REPO,
			Tag:         IMAGE_TAG,
			Command:     `cd /docker-registry && ./setup-configs.sh && export DOCKER_REGISTRY_CONFIG=/docker-registry/config/config_sample.yml && exec docker-registry`,
			Ports:       []int{5000},
			Volumes:     map[string]string{"docker-registry": "/tmp/docker-registry"},
		},
	)
	if err != nil {
		glog.Fatal("Error initializing docker-registry container: %s", err)
	}
}

