// Copyright 2014 The Serviced Authors.
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
	"sort"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	h "github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/service/mocks"

	. "gopkg.in/check.v1"
)

var _ = Suite(&ZKHostUnassignHandlerTestSuite{})

type ZKHostUnassignHandlerTestSuite struct {
	zzk.ZZKTestSuite

	// Dependencies
	registeredHostHandler mocks.RegisteredHostHandler
	assignmentHandler     *ZKAssignmentHandler
	connection            client.Connection

	// Data
	cancel   <-chan interface{}
	testHost h.Host
}

func (s *ZKHostUnassignHandlerTestSuite) SetUpTest(c *C) {
	s.ZZKTestSuite.SetUpTest(c)

	s.testHost = h.Host{ID: "testHost", PoolID: "poolid"}
	s.cancel = make(<-chan interface{})

	var err error
	s.connection, err = zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	err = s.connection.Create(
		Base().Pools().ID("poolid").Hosts().ID("testHost").Path(),
		&HostNode{Host: &s.testHost},
	)
	c.Assert(err, IsNil)

	s.registeredHostHandler = mocks.RegisteredHostHandler{}
	s.registeredHostHandler.On("GetRegisteredHosts", "poolid").
		Return([]h.Host{s.testHost}, nil)

	s.assignmentHandler = NewZKAssignmentHandler(
		&RandomHostSelectionStrategy{},
		&s.registeredHostHandler,
		s.connection)
	s.assignmentHandler.Timeout = time.Second
}

func (s *ZKHostUnassignHandlerTestSuite) TestUnassignsVirtualIPsFromHostCorrectly(c *C) {
	err := s.assignmentHandler.Assign("poolid", "7.7.7.7", "netmask", "http", s.cancel)
	c.Assert(err, IsNil)

	err = s.assignmentHandler.Assign("poolid", "9.9.9.9", "netmask", "http", s.cancel)
	c.Assert(err, IsNil)

	unassignmentHandler := NewZKHostUnassignmentHandler(s.connection)
	err = unassignmentHandler.UnassignAll("poolid", "testHost")
	c.Assert(err, IsNil)

	s.assertNodeHasChildren(c, "pools/poolid/ips", []string{})
	s.assertNodeHasChildren(c, "pools/poolid/hosts/testHost/ips", []string{})
}

func (s *ZKHostUnassignHandlerTestSuite) TestUnassignWorksIfNoAssignments(c *C) {
	unassignmentHandler := NewZKHostUnassignmentHandler(s.connection)
	err := unassignmentHandler.UnassignAll("poolid", "testHost")
	c.Assert(err, IsNil)
}

func (s *ZKHostUnassignHandlerTestSuite) assertNodeHasChildren(c *C, path string, children []string) {
	obtained, err := s.connection.Children(path)
	c.Assert(err, IsNil)
	sort.Strings(obtained)
	c.Assert(obtained, DeepEquals, children)
}
