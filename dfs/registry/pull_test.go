// Copyright 2015 The Serviced Authors.
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

// +build integration

package registry

import (
	"errors"
	"time"

	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

var ErrTestGetImageHashFailed = errors.New("error getting image hash")

func (s *RegistryListenerSuite) TestPull_NoNode(c *C) {
	errC := make(chan error, 1)
	go func() {
		errC <- s.listener.PullImage(time.After(15*time.Second), "someimage")
	}()
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within timeout!")
	case err := <-errC:
		c.Assert(err, NotNil)
	}
}

func (s *RegistryListenerSuite) TestPull_LocalImageFound(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	_ = rImage.Create(c, s.conn)
	rAddress := rImage.Address(s.listener.address)
	s.docker.On("TagImage", rImage.Image.UUID, rAddress).Return(nil).Once()
	errC := make(chan error, 1)
	go func() {
		errC <- s.listener.PullImage(time.After(15*time.Second), rAddress)
	}()
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within timeout!")
	case err := <-errC:
		c.Assert(err, IsNil)
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestPull_RemoteImageFound(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	_ = rImage.Create(c, s.conn)
	rAddress := rImage.Address(s.listener.address)
	s.docker.On("TagImage", rImage.Image.UUID, rAddress).Return(dockerclient.ErrNoSuchImage).Once()
	s.docker.On("PullImage", rAddress).Return(nil).Once()
	s.docker.On("TagImage", rImage.Image.UUID, rAddress).Return(nil).Once()
	errC := make(chan error, 1)
	go func() {
		errC <- s.listener.PullImage(time.After(15*time.Second), rAddress)
	}()
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within timeout!")
	case err := <-errC:
		c.Assert(err, IsNil)
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestPull_RemoteImageFoundByHash(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "matchinghash",
		},
	}
	_ = rImage.Create(c, s.conn)
	rAddress := rImage.Address(s.listener.address)
	s.docker.On("TagImage", rImage.Image.UUID, rAddress).Return(dockerclient.ErrNoSuchImage).Twice()
	s.docker.On("PullImage", rAddress).Return(nil).Once()
	s.docker.On("GetImageHash", rAddress).Return("matchinghash", nil).Once()
	errC := make(chan error, 1)
	go func() {
		errC <- s.listener.PullImage(time.After(15*time.Second), rAddress)
	}()
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within timeout!")
	case err := <-errC:
		c.Assert(err, IsNil)
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestPull_ImagePushingTimeout(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "matchinghash",
		},
	}
	_ = rImage.Create(c, s.conn)
	rAddress := rImage.Address(s.listener.address)
	s.docker.On("TagImage", rImage.Image.UUID, rAddress).Return(dockerclient.ErrNoSuchImage).Twice()
	s.docker.On("PullImage", rAddress).Return(dockerclient.ErrNoSuchImage).Once()
	s.docker.On("GetImageHash", rAddress).Return("unmatchinghash", nil).Once()
	timeout := time.After(20 * time.Second)
	errC := make(chan error, 1)
	go func() {
		errC <- s.listener.PullImage(time.After(15*time.Second), rAddress)
	}()
	select {
	case <-timeout:
		c.Fatalf("listener did not shutdown within timeout!")
	case err := <-errC:
		c.Assert(err, Equals, ErrOpTimeout)
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestPull_ImagePushing(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "matchinghash",
		},
	}
	node := rImage.Create(c, s.conn)
	rAddress := rImage.Address(s.listener.address)
	s.docker.On("TagImage", rImage.Image.UUID, rAddress).Return(dockerclient.ErrNoSuchImage).Times(3)
	s.docker.On("PullImage", rAddress).Return(dockerclient.ErrNoSuchImage).Run(func(_ mock.Arguments) {
		node.PushedAt = time.Now().UTC()
		rImage.Update(c, s.conn, node)
	}).Once()
	s.docker.On("TagImage", rImage.Image.UUID, rAddress).Return(nil).Once()
	s.docker.On("PullImage", rAddress).Return(nil).Once()
	s.docker.On("GetImageHash", rAddress).Return("unmatchinghash", nil).Once()

	errC := make(chan error, 1)
	go func() {
		errC <- s.listener.PullImage(time.After(15*time.Second), rAddress)
	}()
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within timeout!")
	case err := <-errC:
		c.Assert(err, Equals, nil)
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestPull_ImageNotPushingTimeout(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "matchinghash",
		},
	}
	node := rImage.Create(c, s.conn)
	node.PushedAt = time.Now().UTC()
	rImage.Update(c, s.conn, node)
	rAddress := rImage.Address(s.listener.address)
	s.docker.On("TagImage", rImage.Image.UUID, rAddress).Return(dockerclient.ErrNoSuchImage).Times(4)
	s.docker.On("PullImage", rAddress).Return(dockerclient.ErrNoSuchImage).Twice()
	s.docker.On("GetImageHash", rAddress).Return("nonmatchinghash", nil).Once()
	s.docker.On("GetImageHash", rAddress).Return("", ErrTestGetImageHashFailed).Once()
	imageDone := make(chan struct{})
	defer close(imageDone)
	evt, _ := rImage.GetW(c, s.conn, imageDone)
	timeout := time.After(20 * time.Second)
	errC := make(chan error, 1)
	go func() {
		errC <- s.listener.PullImage(time.After(15*time.Second), rAddress)
	}()
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("Push not triggered by the slave!")
	case <-errC:
		c.Fatalf("listener exited unexpectedly!")
	case <-evt:
		_, node = rImage.GetW(c, s.conn, imageDone)
		c.Assert(node.Image, NotNil)
		c.Assert(node.PushedAt.Unix(), Equals, int64(0))
	}
	select {
	case <-timeout:
		c.Fatalf("listener did not shutdown within the timeout!")
	case err := <-errC:
		c.Assert(err, Equals, ErrOpTimeout)
	}
	s.docker.AssertExpectations(c)
}
