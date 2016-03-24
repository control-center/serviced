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

package docker

import (
	"bytes"
	"testing"
	"time"

	dockerclient "github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"
)

func TestDocker(t *testing.T) { TestingT(t) }

type DockerSuite struct {
	dc     *dockerclient.Client
	docker *DockerClient
	regid  string
}

var _ = Suite(&DockerSuite{})

func (s *DockerSuite) SetUpSuite(c *C) {
	var err error
	s.dc, err = dockerclient.NewClient(DefaultSocket)
	if err != nil {
		c.Fatalf("Could not connect to docker client: %s", err)
	}
	s.docker = &DockerClient{dc: s.dc}
	if ctr, err := s.dc.InspectContainer("regtestserver"); err == nil {
		s.dc.KillContainer(dockerclient.KillContainerOptions{ID: ctr.ID})
		opts := dockerclient.RemoveContainerOptions{
			ID:            ctr.ID,
			RemoveVolumes: true,
			Force:         true,
		}
		s.dc.RemoveContainer(opts)
	} else {
		opts := dockerclient.PullImageOptions{
			Repository: "registry",
			Tag:        "2.1.1",
		}
		auth := dockerclient.AuthConfiguration{}
		s.dc.PullImage(opts, auth)
	}
	// Start the docker registry
	opts := dockerclient.CreateContainerOptions{Name: "regtestserver"}
	opts.Config = &dockerclient.Config{Image: "registry:2.1.1"}
	opts.HostConfig = &dockerclient.HostConfig{
		PortBindings: map[dockerclient.Port][]dockerclient.PortBinding{
			"5000/tcp": []dockerclient.PortBinding{
				{HostIP: "localhost", HostPort: "5000"},
			},
		},
	}
	ctr, err := s.dc.CreateContainer(opts)
	if err != nil {
		c.Fatalf("Could not initialize docker registry: %s", err)
	}
	s.regid = ctr.ID
	if err := s.dc.StartContainer(ctr.ID, nil); err != nil {
		c.Fatalf("Could not start docker registry: %s", err)
	}
}

func (s *DockerSuite) TearDownSuite(c *C) {
	// Shut down the docker registry
	s.dc.StopContainer(s.regid, 10)
	opts := dockerclient.RemoveContainerOptions{
		ID:            s.regid,
		RemoveVolumes: true,
		Force:         true,
	}
	s.dc.RemoveContainer(opts)
}

func (s *DockerSuite) SetUpTest(c *C) {
	opts := dockerclient.PullImageOptions{Repository: "busybox", Tag: "latest"}
	auth := dockerclient.AuthConfiguration{}
	if err := s.dc.PullImage(opts, auth); err != nil {
		c.Fatalf("Could not pull test image: %s", err)
	}
}

func (s *DockerSuite) TestFindImage(c *C) {
	_, err := s.docker.FindImage("fakebox")
	c.Assert(err, Equals, dockerclient.ErrNoSuchImage)
	expected, err := s.dc.InspectImage("busybox")
	c.Assert(err, IsNil)
	actual, err := s.docker.FindImage("busybox")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
}

func (s *DockerSuite) TestSaveImages(c *C) {
	buffer := bytes.NewBufferString("")
	err := s.docker.SaveImages([]string{"busybox"}, buffer)
	c.Assert(err, IsNil)
	err = s.dc.RemoveImage("busybox")
	c.Assert(err, IsNil)
	err = s.dc.LoadImage(dockerclient.LoadImageOptions{InputStream: buffer})
	c.Assert(err, IsNil)
	_, err = s.dc.InspectImage("busybox")
	c.Assert(err, IsNil)
}

func (s *DockerSuite) TestLoadImage(c *C) {
	buffer := bytes.NewBufferString("")
	err := s.dc.ExportImages(dockerclient.ExportImagesOptions{Names: []string{"busybox"}, OutputStream: buffer})
	c.Assert(err, IsNil)
	err = s.dc.RemoveImage("busybox")
	c.Assert(err, IsNil)
	err = s.docker.LoadImage(buffer)
	c.Assert(err, IsNil)
	_, err = s.dc.InspectImage("busybox")
	c.Assert(err, IsNil)
}

func (s *DockerSuite) TestPushImage(c *C) {
	defer s.dc.RemoveImage("localhost:5000/busybox:push")
	err := s.docker.PushImage("localhost:5000/busybox:push")
	c.Assert(err, NotNil)
	err = s.dc.TagImage("busybox", dockerclient.TagImageOptions{Repo: "localhost:5000/busybox", Tag: "push"})
	c.Assert(err, IsNil)
	err = s.docker.PushImage("localhost:5000/busybox:push")
	c.Assert(err, IsNil)
	err = s.dc.RemoveImage("localhost:5000/busybox:push")
	c.Assert(err, IsNil)
	opts := dockerclient.PullImageOptions{
		Repository: "localhost:5000/busybox",
		Registry:   "localhost:5000",
		Tag:        "push",
	}
	auth := dockerclient.AuthConfiguration{}
	err = s.dc.PullImage(opts, auth)
	c.Assert(err, IsNil)
	_, err = s.dc.InspectImage("localhost:5000/busybox:push")
	c.Assert(err, IsNil)
}

func (s *DockerSuite) TestPullImage(c *C) {
	defer s.dc.RemoveImage("localhost:5000/busybox:pull")
	err := s.docker.PullImage("localhost:5000/busybox:pull")
	c.Assert(err, NotNil)
	err = s.dc.TagImage("busybox", dockerclient.TagImageOptions{Repo: "localhost:5000/busybox", Tag: "pull"})
	c.Assert(err, IsNil)
	opts := dockerclient.PushImageOptions{
		Name:     "localhost:5000/busybox",
		Tag:      "pull",
		Registry: "localhost:5000",
	}
	auth := dockerclient.AuthConfiguration{}
	err = s.dc.PushImage(opts, auth)
	c.Assert(err, IsNil)
	err = s.dc.RemoveImage("localhost:5000/busybox:pull")
	c.Assert(err, IsNil)
	err = s.docker.PullImage("localhost:5000/busybox:pull")
	c.Assert(err, IsNil)
	_, err = s.dc.InspectImage("localhost:5000/busybox:pull")
	c.Assert(err, IsNil)
}

func (s *DockerSuite) TestTagImage(c *C) {
	defer s.dc.RemoveImage("localhost:5000/busybox:tag")
	err := s.docker.TagImage("fakebox", "localhost:5000/busybox:tag")
	c.Assert(err, NotNil)
	err = s.docker.TagImage("busybox", "localhost:5000/busybox:tag")
	c.Assert(err, IsNil)
	_, err = s.dc.InspectImage("localhost:5000/busybox:tag")
	c.Assert(err, IsNil)
}

func (s *DockerSuite) TestRemoveImage(c *C) {
	defer s.dc.RemoveImage("localhost:5000/busybox:remove")
	err := s.docker.RemoveImage("localhost:5000/busybox:remove")
	c.Assert(err, NotNil)
	err = s.dc.TagImage("busybox", dockerclient.TagImageOptions{Repo: "localhost:5000/busybox", Tag: "remove"})
	c.Assert(err, IsNil)
	err = s.docker.RemoveImage("localhost:5000/busybox:remove")
	c.Assert(err, IsNil)
	_, err = s.dc.InspectImage("localhost:5000/busybox:remove")
	c.Assert(err, NotNil)
}

func (s *DockerSuite) TestFindContainer(c *C) {
	_, err := s.docker.FindContainer("fakecontainer")
	c.Assert(err, NotNil)
	opts := dockerclient.CreateContainerOptions{}
	opts.Config = &dockerclient.Config{Image: "busybox"}
	expected, err := s.dc.CreateContainer(opts)
	c.Assert(err, IsNil)
	defer s.dc.RemoveContainer(dockerclient.RemoveContainerOptions{ID: expected.ID, RemoveVolumes: true, Force: true})
	actual, err := s.docker.FindContainer(expected.ID)
	c.Assert(err, IsNil)
	c.Assert(actual.ID, DeepEquals, expected.ID)
}

func (s *DockerSuite) TestCommitContainer(c *C) {
	defer s.dc.RemoveImage("localhost:5000/busybox:commit")
	opts := dockerclient.CreateContainerOptions{}
	opts.Config = &dockerclient.Config{Image: "busybox"}
	ctr, err := s.dc.CreateContainer(opts)
	c.Assert(err, IsNil)
	defer s.dc.RemoveContainer(dockerclient.RemoveContainerOptions{ID: ctr.ID, RemoveVolumes: true, Force: true})
	expected, err := s.docker.CommitContainer(ctr.ID, "localhost:5000/busybox:commit")
	c.Assert(err, IsNil)
	actual, err := s.dc.InspectImage("localhost:5000/busybox:commit")
	c.Assert(err, IsNil)
	c.Assert(actual.ID, DeepEquals, expected.ID)
}

func (s *DockerSuite) TestGetImageHash(c *C) {
	_, err := s.docker.GetImageHash("fakebox")
	c.Assert(err, Equals, dockerclient.ErrNoSuchImage)
	hash1, err1 := s.docker.GetImageHash("busybox")
	hash2, err2 := s.docker.GetImageHash("registry:2.1.1")
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)
	c.Assert(hash1, Not(Equals), "")
	c.Assert(hash2, Not(Equals), "")
	c.Assert(hash1, Not(Equals), hash2)
}

func (s *DockerSuite) TestGetContainerStats(c *C) {
	stats, err := s.docker.GetContainerStats("fakecontainer", 30*time.Second)
	c.Assert(err, NotNil)
	c.Assert(stats, IsNil)
	opts := dockerclient.CreateContainerOptions{}
	opts.Config = &dockerclient.Config{Image: "busybox"}
	container, err := s.dc.CreateContainer(opts)
	c.Assert(err, IsNil)
	defer s.dc.RemoveContainer(dockerclient.RemoveContainerOptions{ID: container.ID, RemoveVolumes: true, Force: true})
	stats, err = s.docker.GetContainerStats(container.ID, 30*time.Second)
	c.Assert(err, IsNil)
	c.Assert(stats, NotNil)
}

func (s *DockerSuite) TestFindImageByHash(c *C) {
	_, err := s.docker.FindImageByHash("non_existant_hash", false)
	c.Assert(err, Equals, dockerclient.ErrNoSuchImage)
	expectedhash, err := s.docker.GetImageHash("busybox")
	c.Assert(err, IsNil)
	actual, err := s.docker.FindImageByHash(expectedhash, false)
	c.Assert(err, IsNil)
	// have to compare hashes, IDs may not match
	actualhash, err := s.docker.GetImageHash(actual.ID)
	c.Assert(err, IsNil)
	c.Assert(actualhash, Equals, expectedhash)

	// Test "allLayers"
	// find a lower layer of busybox
	historyList, err := s.dc.ImageHistory("busybox")
	c.Assert(err, IsNil)
	c.Assert(len(historyList) > 1, Equals, true)
	lowerLayer := historyList[1]

	lowerLayerHash, err := s.docker.GetImageHash(lowerLayer.ID)
	c.Assert(err, IsNil)

	// If not checking all layers, this should fail
	actual, err = s.docker.FindImageByHash(lowerLayerHash, false)
	c.Assert(err, Equals, dockerclient.ErrNoSuchImage)

	// With all layers = true, this should succeed
	actual, err = s.docker.FindImageByHash(lowerLayerHash, true)
	c.Assert(err, IsNil)
	actualhash, err = s.docker.GetImageHash(actual.ID)
	c.Assert(err, IsNil)
	c.Assert(actualhash, Equals, lowerLayerHash)
}
