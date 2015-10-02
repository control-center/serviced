// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build integration

package registry

import (
	"testing"
	"time"

	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/dfs/docker/mocks"
	dockerclient "github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"
)

func TestRegistry(t *testing.T) { TestingT(t) }

type RegistrySuite struct {
	dc       *dockerclient.Client
	zkid     string
	conn     coordclient.Connection
	docker   *mocks.Docker
	listener *RegistryListener
}

var _ = Suite(&RegistrySuite{})

func (s *RegistrySuite) SetUpSuite(c *C) {
	var err error
	if s.dc, err = dockerclient.NewClient(docker.DefaultSocket); err != nil {
		c.Fatalf("Could not connect to docker client: %s", err)
	}
	// Start zookeeper
	opts := dockerclient.CreateContainerOptions{}
	opts.Config = &dockerclient.Config{Image: "jplock/zookeeper:3.4.6"}
	ctr, err := s.dc.CreateContainer(opts)
	if err != nil {
		c.Fatalf("Could not initialize zookeeper: %s", err)
	}
	s.zkid = ctr.ID
	hconf := &dockerclient.HostConfig{
		PortBindings: map[dockerclient.Port][]dockerclient.PortBinding{
			"2181/tcp": []dockerclient.PortBinding{
				{HostIP: "localhost", HostPort: "2181"},
			},
		},
	}
	if err := s.dc.StartContainer(ctr.ID, hconf); err != nil {
		c.Fatalf("Could not start zookeeper: %s", err)
	}
	// Connect to the zookeeper client
	dsn := zookeeper.NewDSN([]string{"localhost:2181"}, 15*time.Second).String()
	zkclient, err := coordclient.New("zookeeper", dsn, "/", nil)
	if err != nil {
		c.Fatalf("Could not establish the zookeeper client: %s", err)
	}
	s.conn, err = zkclient.GetCustomConnection("/")
	if err != nil {
		c.Fatalf("Could not create a connection to the zookeeper client: %s", err)
	}
}

func (s *RegistrySuite) TearDownSuite(c *C) {
	if s.conn != nil {
		s.conn.Close()
	}
	s.dc.StopContainer(s.zkid, 10)
	opts := dockerclient.RemoveContainerOptions{
		ID:            s.zkid,
		RemoveVolumes: true,
		Force:         true,
	}
	s.dc.RemoveContainer(opts)
}

func (s *RegistrySuite) SetUpTest(c *C) {
	// Initialize the mock docker object
	s.docker = &mocks.Docker{}
	// Initialize the listener
	s.listener = NewRegistryListener(s.docker, "test-server:5000", "test-host")
	s.listener.conn = s.conn
	// Create the base path
	s.conn.CreateDir(zkregistrypath)
}

func (s *RegistrySuite) TearDownTest(c *C) {
	s.conn.Delete(zkregistrypath)
}
