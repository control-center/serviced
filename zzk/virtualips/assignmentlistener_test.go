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

// +build unit

package virtualips_test

import (
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
	. "github.com/control-center/serviced/utils/checkers"
	"github.com/control-center/serviced/zzk/virtualips"
	"github.com/control-center/serviced/zzk/virtualips/mocks"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

var _ = Suite(&AssignmentListenerTestSuite{})

type AssignmentListenerTestSuite struct {
	connection mocks.Connection
	handler    mocks.AssignmentHandler

	virtualIP pool.VirtualIP
}

func (s *AssignmentListenerTestSuite) SetUpTest(c *C) {
	s.connection = mocks.Connection{}
	s.handler = mocks.AssignmentHandler{}

	s.virtualIP = pool.VirtualIP{
		PoolID:        "test",
		IP:            "1.2.3.4",
		Netmask:       "255.255.255.0",
		BindInterface: "http",
	}
}

func (s *AssignmentListenerTestSuite) TestListenerShouldAssignVirtualIPAndWatchForChanges(c *C) {
	var wg sync.WaitGroup
	wg.Add(3)

	done := make(chan struct{})
	shutdown := make(chan interface{})
	defer func() { close(shutdown) }()

	virtualIPEvent := make(chan client.Event)
	s.connection.On("GetW", "/pools/test/virtualIPs/1.2.3.4", mock.Anything, mock.Anything).
		Return((<-chan client.Event)(virtualIPEvent), nil).
		Run(func(a mock.Arguments) {
			node := a.Get(1).(*virtualips.VirtualIPNode)
			node.VirtualIP = &s.virtualIP
			wg.Done()
		})

	s.handler.On("Assign", "test", "1.2.3.4", "255.255.255.0", "http", mock.Anything).
		Return(nil).
		Run(func(a mock.Arguments) {
			wg.Done()
		})

	assignmentEvent := make(chan client.Event)
	s.handler.On("Watch", "test", "1.2.3.4", mock.Anything).
		Return((<-chan client.Event)(assignmentEvent), nil).
		Run(func(a mock.Arguments) {
			wg.Done()
		})

	listener := virtualips.NewAssignmentListener("test", &s.handler)
	listener.SetConnection(&s.connection)

	go func() {
		listener.Spawn(shutdown, "1.2.3.4")
	}()

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		c.Fatalf("Timed out waiting for listener to exit")
	}
}

func (s *AssignmentListenerTestSuite) TestListenerShouldReassignVirtualIPIfHostUnassigns(c *C) {
	listening := make(chan struct{})
	reassigned := make(chan struct{})
	shutdown := make(chan interface{})
	defer func() { close(shutdown) }()

	virtualIPEvent := make(chan client.Event)
	s.connection.On("GetW", "/pools/test/virtualIPs/1.2.3.4", mock.Anything, mock.Anything).
		Return((<-chan client.Event)(virtualIPEvent), nil).
		Run(func(a mock.Arguments) {
			node := a.Get(1).(*virtualips.VirtualIPNode)
			node.VirtualIP = &s.virtualIP
		})

	s.handler.On("Assign", "test", "1.2.3.4", "255.255.255.0", "http", mock.Anything).Return(nil)

	assignmentEvent := make(chan client.Event)
	watch := s.handler.On("Watch", "test", "1.2.3.4", mock.Anything).
		Return((<-chan client.Event)(assignmentEvent), nil)

	watch.Run(func(a mock.Arguments) {
		listening <- struct{}{}
	})

	listener := virtualips.NewAssignmentListener("test", &s.handler)
	listener.SetConnection(&s.connection)

	go func() {
		listener.Spawn(shutdown, "1.2.3.4")
	}()

	select {
	case <-listening:
		watch.Run(func(a mock.Arguments) {
			reassigned <- struct{}{}
		})
		assignmentEvent <- client.Event{Type: client.EventNodeDeleted}
		select {
		case <-reassigned:
		case <-time.After(time.Second):
			c.Fatalf("Timed out waiting for listener to reassign virtual IP")
		}

	case <-time.After(time.Second):
		c.Fatalf("Timed out waiting for listener to start listening for events")
	}
}

func (s *AssignmentListenerTestSuite) TestListenerShouldUnassignVirtualIPGoesAway(c *C) {
	listening := make(chan struct{})
	shutdown := make(chan interface{})
	defer func() { close(shutdown) }()

	virtualIPEvent := make(chan client.Event)
	s.connection.On("GetW", "/pools/test/virtualIPs/1.2.3.4", mock.Anything, mock.Anything).
		Return((<-chan client.Event)(virtualIPEvent), nil).
		Run(func(a mock.Arguments) {
			node := a.Get(1).(*virtualips.VirtualIPNode)
			node.VirtualIP = &s.virtualIP
			listening <- struct{}{}
		})

	s.handler.On("Assign", "test", "1.2.3.4", "255.255.255.0", "http", mock.Anything).Return(nil)

	assignmentEvent := make(chan client.Event)
	s.handler.On("Watch", "test", "1.2.3.4", mock.Anything).
		Return((<-chan client.Event)(assignmentEvent), nil)

	unassigned := make(chan struct{})
	unassignedCalled := false
	s.handler.On("Unassign", "test", "1.2.3.4").Return(nil).
		Run(func(a mock.Arguments) {
			unassignedCalled = true
			unassigned <- struct{}{}
		})

	listener := virtualips.NewAssignmentListener("test", &s.handler)
	listener.SetConnection(&s.connection)

	go func() {
		listener.Spawn(shutdown, "1.2.3.4")
	}()

	select {
	case <-listening:
		c.Assert(unassignedCalled, IsFalse)
		virtualIPEvent <- client.Event{Type: client.EventNodeDeleted}
		select {
		case <-unassigned:
		case <-time.After(time.Second):
			c.Fatalf("Timed out waiting for listener to unassign virtual IP")
		}
	case <-time.After(time.Second):
		c.Fatalf("Timed out waiting for listener to start listening for events")
	}
}
