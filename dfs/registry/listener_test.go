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

package registry

import (
	"errors"
	"fmt"
	"path"
	"testing"
	"time"

	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/dfs/docker/mocks"
	"github.com/control-center/serviced/domain/registry"
	zzktest "github.com/control-center/serviced/zzk/test"
	dockerclient "github.com/fsouza/go-dockerclient"

	. "gopkg.in/check.v1"
)

var (
	ErrTestImageNotFound = errors.New("image not found")
)

type testImage struct {
	Image *registry.Image
}

func (i *testImage) ID() string {
	return i.Image.ID()
}

func (i *testImage) LeaderPath() string {
	return path.Join(zkregistryrepos, i.Image.Library, i.Image.Repo)
}

func (i *testImage) Path() string {
	return path.Join(zkregistrytags, i.ID())
}

func (i *testImage) Address(host string) string {
	return path.Join(host, i.Image.String())
}

func (i *testImage) Create(c *C, conn coordclient.Connection) *RegistryImageNode {
	err := conn.CreateDir(i.LeaderPath())
	c.Assert(err, IsNil)
	node := &RegistryImageNode{Image: *i.Image, PushedAt: time.Unix(0, 0)}
	c.Logf("Creating node at %s: %v", i.Path(), *i.Image)
	err = conn.Create(i.Path(), node)
	c.Assert(err, IsNil)
	exists, err := conn.Exists(i.Path())
	c.Assert(exists, Equals, true)
	return node
}

func (i *testImage) Update(c *C, conn coordclient.Connection, node *RegistryImageNode) {
	err := conn.Set(i.Path(), node)
	c.Assert(err, IsNil)
}

func (i *testImage) GetW(c *C, conn coordclient.Connection, done <-chan struct{}) (<-chan coordclient.Event, *RegistryImageNode) {
	node := &RegistryImageNode{}
	evt, err := conn.GetW(i.Path(), node, done)
	c.Assert(err, IsNil)
	return evt, node
}

func TestRegistryListener(t *testing.T) { TestingT(t) }

type RegistryListenerSuite struct {
	dc        *dockerclient.Client
	conn      coordclient.Connection
	docker    *mocks.Docker
	listener  *RegistryListener
	zkCtrID   string
	zzkServer *zzktest.ZZKServer
}

var _ = Suite(&RegistryListenerSuite{})

func (s *RegistryListenerSuite) SetUpSuite(c *C) {
	s.zzkServer = &zzktest.ZZKServer{}
	err := s.zzkServer.Start()
	if err != nil {
		c.Fatalf("Could not start zookeeper: %s", err)
	}

	// Connect to the zookeeper client
	dsn := zookeeper.NewDSN([]string{fmt.Sprintf("localhost:%d", s.zzkServer.Port)}, 15*time.Second).String()
	zkclient, err := coordclient.New("zookeeper", dsn, "/", nil)
	if err != nil {
		c.Fatalf("Could not establish the zookeeper client: %s", err)
	}
	s.conn, err = zkclient.GetCustomConnection("/")
	if err != nil {
		c.Fatalf("Could not create a connection to the zookeeper client: %s", err)
	}
}

func (s *RegistryListenerSuite) TearDownSuite(c *C) {
	if s.conn != nil {
		s.conn.Close()
	}
	s.zzkServer.Stop()
}

func (s *RegistryListenerSuite) SetUpTest(c *C) {
	// Initialize the mock docker object
	s.docker = &mocks.Docker{}
	// Initialize the listener
	s.listener = NewRegistryListener(s.docker, "test-server:5000", "test-host")
	s.listener.conn = s.conn
	// Create the base path
	s.conn.CreateDir("/docker/registry")
}

func (s *RegistryListenerSuite) TearDownTest(c *C) {
	s.conn.Delete("/docker/registry")
}

func (s *RegistryListenerSuite) TestRegistryListener_NoNode(c *C) {
	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, "keynotexists")
	}()
	select {
	case <-time.After(5 * time.Second):
		close(shutdown)
		select {
		case <-time.After(5 * time.Second):
			close(shutdown)
			select {
			case <-time.After(5 * time.Second):
				c.Fatalf("listener did not shutdown within timeout!")
			case <-done:
				c.Errorf("listener timed out waiting to shutdown")
			}
		case <-done:
		}
	}
}

func (s *RegistryListenerSuite) TestRegistryListener_ImagePushed(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	node := rImage.Create(c, s.conn)
	node.PushedAt = time.Now().UTC()
	rImage.Update(c, s.conn, node)
	imageDone := make(chan struct{})
	defer close(imageDone)
	evt, _ := rImage.GetW(c, s.conn, imageDone)

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Errorf("listener exited prematurely")
	case <-evt:
		c.Errorf("listener updated node")
	}
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within timeout!")
	case <-done:
	}
}

func (s *RegistryListenerSuite) TestRegistryListener_ImageNotFound(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "correcthash",
		},
	}
	regaddr := rImage.Address(s.listener.address)
	_ = rImage.Create(c, s.conn)
	imageDone := make(chan struct{})
	defer close(imageDone)
	evt, _ := rImage.GetW(c, s.conn, imageDone)
	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, false).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, true).Return(nil, ErrTestImageNotFound).Once()

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Fatalf("listener exited prematurely")
	case <-evt:
		c.Errorf("listener updated node")
	}
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_AnotherNodePush(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	node := rImage.Create(c, s.conn)
	s.docker.On("FindImage", rImage.Image.UUID).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil).Once()

	// take lead of the node
	leader, err := s.conn.NewLeader(rImage.LeaderPath())
	c.Assert(err, IsNil)
	leaderDone := make(chan struct{})
	defer close(leaderDone)
	_, err = leader.TakeLead(&RegistryImageLeader{HostID: "master"}, leaderDone)
	c.Assert(err, IsNil)

	childWDone := make(chan struct{})
	defer close(childWDone)
	leaders, cvt, err := s.conn.ChildrenW(rImage.LeaderPath(), childWDone)
	c.Assert(err, IsNil)
	c.Assert(leaders, HasLen, 1)

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()

	// verify lead was attempted
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not try to lead")
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-cvt:
		// assert the listener is NOT the leader
		c.Logf("listener is acquiring lead")
		lnode := &RegistryImageLeader{}
		err = leader.Current(lnode)
		c.Assert(err, IsNil)
		c.Assert(lnode.HostID, Equals, "master")
	}

	// "push" the image
	c.Logf("updating push")
	node.PushedAt = time.Now().UTC()
	rImage.Update(c, s.conn, node)
	imageDone := make(chan struct{})
	defer close(imageDone)
	evt, _ := rImage.GetW(c, s.conn, imageDone)
	err = leader.ReleaseLead()
	c.Assert(err, IsNil)

	// verify the node was NOT updated
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-evt:
		c.Errorf("listener updated node")
	}

	// verify shutdown
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within the timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_PushFails(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	_ = rImage.Create(c, s.conn)
	imageDone := make(chan struct{})
	defer close(imageDone)
	evt, _ := rImage.GetW(c, s.conn, imageDone)
	leader, err := s.conn.NewLeader(rImage.LeaderPath())
	c.Assert(err, IsNil)
	childWDone := make(chan struct{})
	defer close(childWDone)
	_, cvt, err := s.conn.ChildrenW(rImage.LeaderPath(), childWDone)
	c.Assert(err, IsNil)
	timeoutC := make(chan time.Time)
	s.docker.On("FindImage", rImage.Image.UUID).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil)
	s.docker.On("TagImage", rImage.Image.UUID, rImage.Address(s.listener.address)).Return(nil)
	s.docker.On("PushImage", rImage.Address(s.listener.address)).Return(errors.New("could not push image")).WaitUntil(timeoutC).Once()
	s.docker.On("PushImage", rImage.Address(s.listener.address)).Return(nil)

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()

	// verify lead was attempted
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not try to lead")
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-cvt:
		// assert the listener IS the leader
		c.Logf("listener is acquiring lead")
		lnode := &RegistryImageLeader{}
		err = leader.Current(lnode)
		c.Assert(err, IsNil)
		c.Assert(lnode.HostID, Equals, s.listener.hostid)
	case <-evt:
		c.Errorf("listener updated node")
	}

	// verify the node did update
	close(timeoutC)
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not update node!")
	case <-done:
		c.Fatalf("listener exited prematurely")
	case <-evt:
	}

	// verify shutdown
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within the timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_LeadDisconnect(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	_ = rImage.Create(c, s.conn)
	imageDone := make(chan struct{})
	defer close(imageDone)
	evt, _ := rImage.GetW(c, s.conn, imageDone)
	leader, err := s.conn.NewLeader(rImage.LeaderPath())
	c.Assert(err, IsNil)
	childWDone := make(chan struct{})
	defer close(childWDone)
	_, cvt, err := s.conn.ChildrenW(rImage.LeaderPath(), childWDone)
	c.Assert(err, IsNil)
	timeoutC := make(chan time.Time)
	s.docker.On("FindImage", rImage.Image.UUID).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil).Once()
	s.docker.On("TagImage", rImage.Image.UUID, rImage.Address(s.listener.address)).Return(nil).Once()
	s.docker.On("PushImage", rImage.Address(s.listener.address)).Return(nil).WaitUntil(timeoutC).Once()

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()

	// verify lead was attempted
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not try to lead")
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-cvt:
		// assert the listener IS the leader
		c.Logf("listener is acquiring lead")
		lnode := &RegistryImageLeader{}
		err = leader.Current(lnode)
		c.Assert(err, IsNil)
		c.Assert(lnode.HostID, Equals, s.listener.hostid)
	case <-evt:
		c.Errorf("listener updated node")
	}

	// delete the leader
	children, err := s.conn.Children(rImage.LeaderPath())
	c.Assert(err, IsNil)
	c.Assert(children, HasLen, 1)
	err = s.conn.Delete(path.Join(rImage.LeaderPath(), children[0]))
	c.Assert(err, IsNil)

	// verify the node did update
	close(timeoutC)
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not update the node")
	case <-done:
		c.Fatalf("listener exited prematurely")
	case <-evt:
	}

	// verify shutdown
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within the timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_Success(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	_ = rImage.Create(c, s.conn)
	imageDone := make(chan struct{})
	defer close(imageDone)
	evt, _ := rImage.GetW(c, s.conn, imageDone)
	leader, err := s.conn.NewLeader(rImage.LeaderPath())
	c.Assert(err, IsNil)
	childWDone := make(chan struct{})
	defer close(childWDone)
	_, cvt, err := s.conn.ChildrenW(rImage.LeaderPath(), childWDone)
	c.Assert(err, IsNil)
	timeoutC := make(chan time.Time)
	s.docker.On("FindImage", rImage.Image.UUID).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil).Once()
	s.docker.On("TagImage", rImage.Image.UUID, rImage.Address(s.listener.address)).Return(nil).Once()
	s.docker.On("PushImage", rImage.Address(s.listener.address)).Return(nil).WaitUntil(timeoutC).Once()

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()

	// verify lead was attempted
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not try to lead")
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-cvt:
		// assert the listener IS the leader
		c.Logf("listener is acquiring lead")
		lnode := &RegistryImageLeader{}
		err = leader.Current(lnode)
		c.Assert(err, IsNil)
		c.Assert(lnode.HostID, Equals, s.listener.hostid)
	case <-evt:
		c.Errorf("listener updated node")
	}

	// verify the node DID update
	close(timeoutC)
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Fatalf("listener exited prematurely")
	case <-evt:
	}

	// verify shutdown
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within the timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_SuccessByID(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}

	s.docker.On("FindImage", rImage.Image.UUID).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, IsNil)
	c.Assert(img.ID, Equals, rImage.Image.UUID)
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_SuccessByRegAddr(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(&dockerclient.Image{ID: "differentid"}, nil).Once()
	s.docker.On("GetImageHash", "differentid").Return(rImage.Image.Hash, nil).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, IsNil)
	c.Assert(img.ID, Equals, "differentid")
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_SuccessByPullAndID(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(nil).Once()
	s.docker.On("FindImage", regaddr).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, IsNil)
	c.Assert(img.ID, Equals, rImage.Image.UUID)
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_SuccessByPullAndHash(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(nil).Once()
	s.docker.On("FindImage", regaddr).Return(&dockerclient.Image{ID: "differentid"}, nil).Once()
	s.docker.On("GetImageHash", "differentid").Return(rImage.Image.Hash, nil).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, IsNil)
	c.Assert(img.ID, Equals, "differentid")
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_SuccessByHash(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, false).Return(&dockerclient.Image{ID: "differentid"}, nil).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, IsNil)
	c.Assert(img.ID, Equals, "differentid")
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_SuccessByHashAllLayers(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, false).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, true).Return(&dockerclient.Image{ID: "differentid"}, nil).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, IsNil)
	c.Assert(img.ID, Equals, "differentid")
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_Fail(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, false).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, true).Return(nil, ErrTestImageNotFound).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, Equals, ErrTestImageNotFound)
	c.Assert(img, IsNil)
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_FailGettingHash(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(&dockerclient.Image{ID: "differentid"}, nil).Once()
	s.docker.On("GetImageHash", "differentid").Return("", ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, false).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, true).Return(nil, ErrTestImageNotFound).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, Equals, ErrTestImageNotFound)
	c.Assert(img, IsNil)
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_FailHashesDontMatch(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(&dockerclient.Image{ID: "differentid"}, nil).Once()
	s.docker.On("GetImageHash", "differentid").Return("differenthash", nil).Once()
	s.docker.On("PullImage", regaddr).Return(ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, false).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, true).Return(nil, ErrTestImageNotFound).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, Equals, ErrTestImageNotFound)
	c.Assert(img, IsNil)
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_FailAfterPull(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(nil).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, false).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, true).Return(nil, ErrTestImageNotFound).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, Equals, ErrTestImageNotFound)
	c.Assert(img, IsNil)
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_FailAfterPullGettingHash(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(nil).Once()
	s.docker.On("FindImage", regaddr).Return(&dockerclient.Image{ID: "differentid"}, nil).Once()
	s.docker.On("GetImageHash", "differentid").Return("", ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, false).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, true).Return(nil, ErrTestImageNotFound).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, Equals, ErrTestImageNotFound)
	c.Assert(img, IsNil)
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_TestFindImage_FailAfterPullHashesDontMatch(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
			Hash:    "hashvalue",
		},
	}
	regaddr := rImage.Address(s.listener.address)

	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImage", regaddr).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("PullImage", regaddr).Return(nil).Once()
	s.docker.On("FindImage", regaddr).Return(&dockerclient.Image{ID: "differentid"}, nil).Once()
	s.docker.On("GetImageHash", "differentid").Return("differenthash", nil).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, false).Return(nil, ErrTestImageNotFound).Once()
	s.docker.On("FindImageByHash", rImage.Image.Hash, true).Return(nil, ErrTestImageNotFound).Once()

	img, err := s.listener.FindImage(rImage.Image)

	c.Assert(err, Equals, ErrTestImageNotFound)
	c.Assert(img, IsNil)
	s.docker.AssertExpectations(c)
}
