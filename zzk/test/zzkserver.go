// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build integration

// Package zzktest provides helper code for integration tests that use Zookeeper
package zzktest

import (
	"fmt"

	"github.com/control-center/serviced/dfs/docker"
	dockerclient "github.com/fsouza/go-dockerclient"
)

// ZZKServer
type ZZKServer struct {
	dc       *dockerclient.Client
	zkCtrID  string
}

// Start will start an instance of Zookeeper using a docker container.
// If it finds that a container is running already, it will kill that container
// before starting a new instance
func (s *ZZKServer) Start() error {
	var err error
	if s.dc, err = dockerclient.NewClient(docker.DefaultSocket); err != nil {
		return fmt.Errorf("Could not connect to docker client: %s", err)
	}

	// Make sure we start with a fresh instance
	if ctr, err := s.dc.InspectContainer("zktestserver"); err == nil {
		s.dc.KillContainer(dockerclient.KillContainerOptions{ID: ctr.ID})
		opts := dockerclient.RemoveContainerOptions{
			ID:            ctr.ID,
			RemoveVolumes: true,
			Force:         true,
		}
		s.dc.RemoveContainer(opts)
	} else {
		opts := dockerclient.PullImageOptions{
			Repository: "jplock/zookeeper",
			Tag:        "3.4.6",
		}
		auth := dockerclient.AuthConfiguration{}
		s.dc.PullImage(opts, auth)
	}

	// Start zookeeper
	opts := dockerclient.CreateContainerOptions{Name: "zktestserver"}
	opts.Config = &dockerclient.Config{Image: "jplock/zookeeper:3.4.6"}
	ctr, err := s.dc.CreateContainer(opts)
	if err != nil {
		return fmt.Errorf("Could not initialize zookeeper: %s", err)
	}

	// Start the container
	s.zkCtrID = ctr.ID
	hconf := &dockerclient.HostConfig{
		PortBindings: map[dockerclient.Port][]dockerclient.PortBinding{
			"2181/tcp": []dockerclient.PortBinding{
				{HostIP: "localhost", HostPort: "2181"},
			},
		},
	}
	if err := s.dc.StartContainer(ctr.ID, hconf); err != nil {
		return fmt.Errorf("Could not start zookeeper: %s", err)
	}

	return nil
}

// Stop will stop the Zookeeper instance and remove the container
func (s *ZZKServer) Stop() {
	if s.zkCtrID == "" {
		return
	}

	s.dc.StopContainer(s.zkCtrID, 10)
	opts := dockerclient.RemoveContainerOptions{
		ID:            s.zkCtrID,
		RemoveVolumes: true,
		Force:         true,
	}
	s.dc.RemoveContainer(opts)
	s.zkCtrID = ""
}
