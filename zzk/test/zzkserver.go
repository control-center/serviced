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

//go:build integration
// +build integration

// Package zzktest provides helper code for integration tests that use Zookeeper
package zzktest

import (
	"fmt"
	"strconv"

	"github.com/control-center/serviced/dfs/docker"
	dockerclient "github.com/fsouza/go-dockerclient"
)

// ZZKServer
type ZZKServer struct {
	Port    int
	dc      *dockerclient.Client
	zkCtrID string
}

const (
	DEFAULT_PORT    = 2181
	zzkVersion      = "3.9.2"
	DefaultRegistry = "https://index.docker.io/v1/"
)

// Start will start an instance of Zookeeper using a docker container.
// If it finds that a container is running already, it will kill that container
// before starting a new instance
func (s *ZZKServer) Start() error {
	var err error
	if s.dc, err = dockerclient.NewClient(docker.DefaultSocket); err != nil {
		return fmt.Errorf("Could not connect to docker client: %s", err)
	}

	// Make sure we start with a fresh instance
	containerName := "zktestserver"
	if ctr, err := s.dc.InspectContainer(containerName); err == nil {
		fmt.Printf("ZZKServer.Start(): Killing container %s ...\n", ctr.ID)
		err = s.dc.KillContainer(dockerclient.KillContainerOptions{ID: ctr.ID})
		if err != nil {
			return fmt.Errorf("ERROR: unable to kill container %s: %s", ctr.ID, err)
		}

		opts := dockerclient.RemoveContainerOptions{
			ID:            ctr.ID,
			RemoveVolumes: true,
			Force:         true,
		}
		err = s.dc.RemoveContainer(opts)
		if err != nil {
			return fmt.Errorf("ERROR: unable to remove container %s: %s", ctr.ID, err)
		}
	} else if _, ok := err.(*dockerclient.NoSuchContainer); !ok {
		return fmt.Errorf("ERROR: unable to inspect container %s: %s", containerName, err)
	} else {
		opts := dockerclient.PullImageOptions{
			Repository: "zookeeper",
			Tag:        zzkVersion,
		}

		auths, err := dockerclient.NewAuthConfigurationsFromDockerCfg()
		if err != nil {
			return err
		}

		auth, _ := auths.Configs[DefaultRegistry]

		fmt.Printf("ZZKServer.Start(): Pulling %s:%s ...\n", opts.Repository, opts.Tag)
		err = s.dc.PullImage(opts, auth)
		if err != nil {
			return fmt.Errorf("Could not pull image %s:%s - %s", opts.Repository, opts.Tag, err)
		}
		fmt.Printf("ZZKServer.Start(): Pull finished for %s:%s ...\n", opts.Repository, opts.Tag)
	}

	// Start zookeeper
	opts := dockerclient.CreateContainerOptions{Name: containerName}
	opts.Config = &dockerclient.Config{Image: fmt.Sprintf("zookeeper:%s", zzkVersion)}

	if s.Port == 0 {
		s.Port = DEFAULT_PORT
	}
	dockerPort := dockerclient.Port(fmt.Sprintf("%d/tcp", s.Port))
	opts.HostConfig = &dockerclient.HostConfig{
		PortBindings: map[dockerclient.Port][]dockerclient.PortBinding{
			dockerPort: []dockerclient.PortBinding{
				{HostIP: "localhost", HostPort: strconv.Itoa(s.Port)},
			},
		},
	}
	ctr, err := s.dc.CreateContainer(opts)
	if err != nil {
		return fmt.Errorf("Could not initialize zookeeper: %s", err)
	}

	// Start the container
	s.zkCtrID = ctr.ID
	if err := s.dc.StartContainer(ctr.ID, nil); err != nil {
		return fmt.Errorf("Could not start zookeeper: %s", err)
	}

	return nil
}

// Stop will stop the Zookeeper instance and remove the container
func (s *ZZKServer) Stop() {
	if s.zkCtrID == "" {
		return
	}

	err := s.dc.StopContainer(s.zkCtrID, 10)
	if err != nil {
		fmt.Printf("ERROR: unable to stop container %s: %s", s.zkCtrID, err)
	}

	opts := dockerclient.RemoveContainerOptions{
		ID:            s.zkCtrID,
		RemoveVolumes: true,
		Force:         true,
	}
	err = s.dc.RemoveContainer(opts)
	if err != nil {
		fmt.Printf("ERROR: unable to remove container %s: %s", s.zkCtrID, err)
	}
	s.zkCtrID = ""
}
