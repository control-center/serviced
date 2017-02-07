// Copyright 2016 The Serviced Authors.
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

// +build integration,!quick

package zzk_test

import (
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	. "github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/mocks"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestManage(c *C) {
	shutdown := make(chan interface{})
	pathname := "/managetest"

	exit := make(chan time.Time)

	s := &mocks.Listener2{}
	s.On("Listen", mock.Anything, mock.Anything).Return().Run(
		func(args mock.Arguments) {
			_, ok := args.Get(0).(<-chan interface{})
			c.Assert(ok, Equals, true)
			_, ok = args.Get(1).(client.Connection)
			c.Assert(ok, Equals, true)
		},
	).WaitUntil(exit).Twice()

	done := make(chan struct{})
	go func() {
		Manage(shutdown, pathname, s)
		close(done)
	}()

	select {
	case <-done:
		c.Errorf("Listener exited prematurely")
	case <-time.After(time.Second):
	}

	// restart
	exit <- time.Now()
	select {
	case <-done:
		c.Errorf("Listener exited prematurely")
	case <-time.After(time.Second):
	}

	// shutdown
	s.On("Exited").Return().Once()
	close(shutdown)
	exit <- time.Now()
	select {
	case <-done:
	case <-time.After(time.Second):
		c.Errorf("Listener did not shut down")
	}
	s.AssertExpectations(c)
}

func (t *ZZKTest) TestListen2(c *C) {
	conn, err := GetLocalConnection("/")
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	pathname := "/listentest"

	s := &mocks.Spawner{}
	s.On("SetConn", conn).Return()
	s.On("Path").Return(pathname)

	s.On("Post", mock.Anything).Return().Run(
		func(args mock.Arguments) {
			active, ok := args.Get(0).(map[string]struct{})
			c.Assert(ok, Equals, true)
			c.Assert(active, HasLen, 0)
		},
	).Twice()

	done := make(chan struct{})
	go func() {
		Listen2(shutdown, conn, s)
		close(done)
	}()

	// path does not exist
	select {
	case <-done:
		c.Errorf("Listener exited prematurely")
	case <-time.After(time.Second):
	}

	// path exists
	err = conn.CreateDir(pathname)
	c.Assert(err, IsNil)

	select {
	case <-done:
		c.Errorf("Listener exited prematurely")
	case <-time.After(time.Second):
		s.AssertExpectations(c)
	}

	// add child
	nodeA := "a"
	nodeAExit := make(chan time.Time)

	s.On("Pre").Return()

	s.On("Spawn", mock.Anything, nodeA).Return().Run(
		func(args mock.Arguments) {
			cancel, ok := args.Get(0).(<-chan struct{})
			c.Assert(ok, Equals, true)
			select {
			case <-nodeAExit:
			case <-cancel:
			}
		},
	).Once()

	s.On("Post", mock.Anything).Return().Run(
		func(args mock.Arguments) {
			active, ok := args.Get(0).(map[string]struct{})
			c.Assert(ok, Equals, true)
			c.Assert(active, HasLen, 1)
			_, ok = active[nodeA]
			c.Assert(ok, Equals, true)
		},
	).Twice()

	err = conn.CreateDir(path.Join(pathname, nodeA))
	c.Assert(err, IsNil)

	select {
	case <-done:
		c.Errorf("Listener exited prematurely")
	case <-time.After(time.Second):
	}

	// delete child
	err = conn.Delete(path.Join(pathname, nodeA))
	c.Assert(err, IsNil)

	select {
	case <-done:
		c.Errorf("Listener exited prematurely")
	case <-time.After(time.Second):
		s.AssertExpectations(c)
	}

	// exit spawn
	s.On("Post", mock.Anything).Return().Run(
		func(args mock.Arguments) {
			active, ok := args.Get(0).(map[string]struct{})
			c.Assert(ok, Equals, true)
			c.Assert(active, HasLen, 0)
		},
	).Once()

	nodeAExit <- time.Now()

	select {
	case <-done:
		c.Errorf("Listener exited prematurely")
	case <-time.After(time.Second):
		s.AssertExpectations(c)
	}

	// add child
	nodeB := "b"
	nodeBExit := make(chan time.Time)

	s.On("Spawn", mock.Anything, nodeB).Return().Run(
		func(args mock.Arguments) {
			cancel, ok := args.Get(0).(<-chan struct{})
			c.Assert(ok, Equals, true)
			select {
			case <-nodeBExit:
			case <-cancel:
			}
		},
	).Twice()

	s.On("Post", mock.Anything).Return().Run(
		func(args mock.Arguments) {
			active, ok := args.Get(0).(map[string]struct{})
			c.Assert(ok, Equals, true)
			c.Assert(active, HasLen, 1)
			_, ok = active[nodeB]
			c.Assert(ok, Equals, true)
		},
	).Twice()

	err = conn.CreateDir(path.Join(pathname, nodeB))
	c.Assert(err, IsNil)

	// exit spawn
	nodeBExit <- time.Now()

	select {
	case <-done:
		c.Errorf("Listener exited prematurely")
	case <-time.After(time.Second):
		s.AssertExpectations(c)
	}

	// shutdown
	close(shutdown)
	select {
	case <-done:
	case <-time.After(time.Second):
		c.Fatalf("Listener did not shutdown")
	}
}
