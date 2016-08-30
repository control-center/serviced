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

package registry_test

import (
	"time"

	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/registry2"
	"github.com/control-center/serviced/zzk/registry2/mocks"
	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestImportListener(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/net/export/tenantid/app1")
	c.Assert(err, IsNil)

	listener := NewImportListener("tenantid")
	term := &mocks.StringMatcher{}
	term.On("MatchString", "app1").Return(true)
	ev := listener.AddTerm(term)

	shutdown := make(chan struct{})
	done := make(chan struct{})
	go func() {
		listener.Run(shutdown, conn)
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case app := <-ev:
		c.Check(app, Equals, "app1")
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for listener")
	}

	// add another match
	term.On("MatchString", "app2").Return(true)
	err = conn.CreateDir("/net/export/tenantid/app2")
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case app := <-ev:
		c.Check(app, Equals, "app2")
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for listener")
	}

	// add a mismatch
	term.On("MatchString", "app3").Return(false)
	err = conn.CreateDir("/net/export/tenantid/app3")
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case app := <-ev:
		c.Errorf("Unexpected event receieved from listener: %s", app)
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
	}

	// shutdown
	close(shutdown)

	timer.Reset(time.Second)
	select {
	case app := <-ev:
		c.Errorf("Unexpected event receieved from listener: %s", app)
	case <-done:
	case <-timer.C:
		c.Fatalf("Timed out waiting for listener")
	}

	// add another node and restart
	term.On("MatchString", "app4").Return(true)
	err = conn.CreateDir("/net/export/tenantid/app4")
	c.Assert(err, IsNil)

	shutdown = make(chan struct{})
	done = make(chan struct{})
	go func() {
		listener.Run(shutdown, conn)
		close(done)
	}()

	timer = time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case app := <-ev:
		c.Check(app, Equals, "app4")
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for listener")
	}

	// shutdown again
	close(shutdown)

	timer.Reset(time.Second)
	select {
	case app := <-ev:
		c.Errorf("Unexpected event receieved from listener: %s", app)
	case <-done:
	case <-timer.C:
		c.Fatalf("Timed out waiting for listener")
	}
}
