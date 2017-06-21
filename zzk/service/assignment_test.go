// Copyright 2017 The Serviced Authors.
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

// +build integration,!quick

package service_test

import (
	"time"

	"github.com/control-center/serviced/coordinator/client"
	h "github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/service/mocks"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

var _ = Suite(&ZKAssignmentHandlerTestSuite{})

type ZKAssignmentHandlerTestSuite struct {
	zzk.ZZKTestSuite

	// Dependencies
	registeredHostHandler mocks.RegisteredHostHandler
	assignmentHandler     *ZKAssignmentHandler
	connection            client.Connection
	strategy              mocks.HostSelectionStrategy

	selectHostCall *mock.Call

	testHost h.Host
}

func (s *ZKAssignmentHandlerTestSuite) SetUpTest(c *C) {
	s.ZZKTestSuite.SetUpTest(c)

	s.testHost = h.Host{ID: "testHost", PoolID: "poolid"}

	var err error
	s.connection, err = zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	strategy := &mocks.HostSelectionStrategy{}
	s.selectHostCall = strategy.On("Select", mock.AnythingOfType("[]host.Host")).
		Return(s.testHost, nil)

	s.registeredHostHandler = mocks.RegisteredHostHandler{}
	s.assignmentHandler = NewZKAssignmentHandler(
		strategy,
		&s.registeredHostHandler,
		s.connection)
	s.assignmentHandler.Timeout = time.Second

	s.registeredHostHandler.On("GetRegisteredHosts", "poolid").
		Return([]h.Host{s.testHost}, nil)

	err = s.connection.Create(
		Base().Pools().ID("poolid").Hosts().ID("testHost").Path(),
		&HostNode{Host: &s.testHost},
	)
	c.Assert(err, IsNil)
}

func (s *ZKAssignmentHandlerTestSuite) TestAssignsCorrectly(c *C) {
	s.assignmentHandler.Assign("poolid", "7.7.7.7", "netmask", "http")
	s.assertNodeHasChildren(c, "pools/poolid/ips", []string{"testHost-7.7.7.7"})
	s.assertNodeHasChildren(c, "pools/poolid/hosts/testHost/ips", []string{"testHost-7.7.7.7"})
}

func (s *ZKAssignmentHandlerTestSuite) TestMultipleAssignsReturnsError(c *C) {
	s.assignmentHandler.Assign("poolid", "7.7.7.7", "netmask", "http")
	s.assertNodeHasChildren(c, "pools/poolid/ips", []string{"testHost-7.7.7.7"})
	s.assertNodeHasChildren(c, "pools/poolid/hosts/testHost/ips", []string{"testHost-7.7.7.7"})

	err := s.assignmentHandler.Assign("poolid", "7.7.7.7", "netmask", "http")
	c.Assert(err, Equals, ErrAlreadyAssigned)
}

func (s *ZKAssignmentHandlerTestSuite) TestUnassignsCorrectly(c *C) {
	s.assignmentHandler.Assign("poolid", "7.7.7.7", "netmask", "http")
	s.assertNodeHasChildren(c, "pools/poolid/ips", []string{"testHost-7.7.7.7"})
	s.assertNodeHasChildren(c, "pools/poolid/hosts/testHost/ips", []string{"testHost-7.7.7.7"})

	s.assignmentHandler.Unassign("poolid", "7.7.7.7")
	s.assertNodeHasChildren(c, "pools/poolid/ips", []string{})
	s.assertNodeHasChildren(c, "pools/poolid/hosts/testHost/ips", []string{})
}

func (s *ZKAssignmentHandlerTestSuite) TestUnassignsWithNoAssignmentReturnsError(c *C) {
	err := s.assignmentHandler.Unassign("poolid", "7.7.7.7")
	c.Assert(err, Equals, ErrNoAssignedHost)
}

func (s *ZKAssignmentHandlerTestSuite) TestExcludesHostAfterAssigning(c *C) {
	s.assignmentHandler.Assign("poolid", "7.7.7.7", "netmask", "http")
	request := IPRequest{PoolID: "poolid", HostID: "testHost", IPAddress: "7.7.7.7"}
	err := DeleteIP(s.connection, request)
	c.Assert(err, IsNil)

	s.selectHostCall.Return(h.Host{}, ErrNoHosts).Run(func(a mock.Arguments) {
		hosts := a.Get(0).([]h.Host)
		c.Assert(hosts, HasLen, 0)
	}).Once()

	err = s.assignmentHandler.Assign("poolid", "7.7.7.7", "netmask", "http")
	c.Assert(err, Equals, ErrNoHosts)

	time.Sleep(time.Second)

	s.selectHostCall.Return(s.testHost, nil).Run(func(a mock.Arguments) {
		hosts := a.Get(0).([]h.Host)
		c.Assert(hosts, HasLen, 1)
	}).Once()

	err = s.assignmentHandler.Assign("poolid", "7.7.7.7", "netmask", "http")
	c.Assert(err, IsNil)
}

func (s *ZKAssignmentHandlerTestSuite) assertNodeHasChildren(c *C, path string, children []string) {
	obtained, err := s.connection.Children(path)
	c.Assert(err, IsNil)
	c.Assert(obtained, DeepEquals, children)
}
