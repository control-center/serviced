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
	s.dc, err = dockerclient.NewClient(defaultSocket)
	if err != nil {
		c.Fatalf("Could not connect to docker client: %s", err)
	}
	s.docker = &DockerClient{dc: s.dc}
	// Start the docker registry
	opts := dockerclient.CreateContainerOptions{}
	opts.Config = &dockerclient.Config{Image: "registry"}
	ctr, err := s.dc.CreateContainer(opts)
	if err != nil {
		c.Fatalf("Could not initialize docker registry: %s", err)
	}
	s.regid = ctr.ID
	hconf := &dockerclient.HostConfig{
		PortBindings: map[dockerclient.Port][]dockerclient.PortBinding{
			"5000/tcp": []dockerclient.PortBinding{
				{HostIP: "localhost", HostPort: "5000"},
			},
		},
	}
	if err := s.dc.StartContainer(ctr.ID, hconf); err != nil {
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

func (s *DockerSuite) TestImageHistory(c *C) {
	_, err := s.dc.ImageHistory("fakebox")
	c.Assert(err, NotNil)
	expected, err := s.dc.ImageHistory("busybox")
	c.Assert(err, IsNil)
	actual, err := s.docker.ImageHistory("busybox")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
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
