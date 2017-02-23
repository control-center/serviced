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

package service_test

import (
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	p "github.com/control-center/serviced/domain/pool"
	. "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/service/mocks"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

var _ = Suite(&PoolListenerTestSuite{})

type PoolListenerTestSuite struct {
	pool p.ResourcePool

	shutdown  <-chan interface{}
	poolEvent chan client.Event
	ipEvent   chan client.Event

	getWCall      *mock.Call
	childrenWCall *mock.Call
	syncCall      *mock.Call
	listener      *PoolListener
}

func (s *PoolListenerTestSuite) SetUpTest(c *C) {
	connection := mocks.Connection{}
	synchronizer := mocks.VirtualIPSynchronizer{}

	s.pool = p.ResourcePool{
		ID: "test",
		VirtualIPs: []p.VirtualIP{
			p.VirtualIP{
				PoolID: "test",
				IP:     "1.2.3.4",
			},
		},
	}

	s.poolEvent = make(chan client.Event)
	s.getWCall = connection.On("GetW", "/pools/test", mock.Anything, mock.Anything).
		Return((<-chan client.Event)(s.poolEvent), nil).
		Run(func(a mock.Arguments) {
			node := a.Get(1).(*PoolNode)
			node.ResourcePool = &s.pool
		})

	s.ipEvent = make(chan client.Event)
	s.childrenWCall = connection.On("ChildrenW", "/pools/test/ips", mock.Anything).
		Return([]string{"host-1.2.3.4", "host-7.7.7.7"}, (<-chan client.Event)(s.ipEvent), nil)

	s.syncCall = synchronizer.On("Sync", s.pool, mock.Anything, s.shutdown).
		Return(nil)

	s.listener = NewPoolListener(&synchronizer)
	s.listener.SetConnection(&connection)
}

func (s *PoolListenerTestSuite) TestListenerShouldSyncAndWatchForChanges(c *C) {
	var wg sync.WaitGroup
	wg.Add(3)

	done := make(chan struct{})

	s.getWCall.Run(func(a mock.Arguments) {
		node := a.Get(1).(*PoolNode)
		node.ResourcePool = &s.pool
		wg.Done()
	})

	s.childrenWCall.Run(func(a mock.Arguments) {
		wg.Done()
	})

	s.syncCall.Run(func(a mock.Arguments) {
		wg.Done()
	})

	go func() {
		s.listener.Spawn(s.shutdown, "test")
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

func (s *PoolListenerTestSuite) TestListenerShouldSyncAfterPoolEvent(c *C) {
	done := make(chan struct{})

	s.syncCall.Run(func(a mock.Arguments) {
		done <- struct{}{}
	})

	go func() {
		s.listener.Spawn(s.shutdown, "test")
	}()

	select {
	case <-done:
		s.poolEvent <- client.Event{Type: client.EventNodeDeleted}
		select {
		case <-done:
		case <-time.After(time.Second):
			c.Fatalf("Timed out waiting for listener to exit")
		}
	case <-time.After(time.Second):
		c.Fatalf("Timed out waiting for listener to exit")
	}
}

func (s *PoolListenerTestSuite) TestListenerShouldSyncAfterIPEvent(c *C) {
	done := make(chan struct{})

	s.syncCall.Run(func(a mock.Arguments) {
		done <- struct{}{}
	})

	go func() {
		s.listener.Spawn(s.shutdown, "test")
	}()

	select {
	case <-done:
		s.ipEvent <- client.Event{Type: client.EventNodeDeleted}
		select {
		case <-done:
		case <-time.After(time.Second):
			c.Fatalf("Timed out waiting for listener to exit")
		}
	case <-time.After(time.Second):
		c.Fatalf("Timed out waiting for listener to exit")
	}
}

func (s *PoolListenerTestSuite) TestListenerShouldProperlyParseHostIPChildren(c *C) {
	done := make(chan struct{})

	var assignments map[string]string
	s.syncCall.Run(func(a mock.Arguments) {
		assignments = a.Get(1).(map[string]string)
		done <- struct{}{}
	})

	go func() {
		s.listener.Spawn(s.shutdown, "test")
	}()

	select {
	case <-done:
		c.Assert(assignments["1.2.3.4"], Equals, "host")
		c.Assert(assignments["7.7.7.7"], Equals, "host")
	case <-time.After(time.Second):
		c.Fatalf("Timed out waiting for listener to exit")
	}
}
