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
	"github.com/control-center/serviced/zzk/service2"
	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestRegisterExport(c *C) {

	// pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// watch the application path
	done := make(chan struct{})
	ok, ev, err := conn.ExistsW("/net/export/tenantid/app", done)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, false)

	// start
	shutdown := make(chan struct{})
	go func() {
		RegisterExport(shutdown, conn, "tenantid", ExportDetails{
			ExportBinding: service.ExportBinding{Application: "app"},
			InstanceID:    1,
		})
		close(done)
	}()

	var ch []string

	timer := time.NewTimer(time.Second)
	select {
	case <-ev:
		ch, ev, err = conn.ChildrenW("/net/export/tenantid/app", done)
		c.Assert(err, IsNil)
		if len(ch) == 0 {
			timer.Reset(time.Second)
			select {
			case <-ev:
				ch, ev, err = conn.ChildrenW("/net/export/tenantid/app", done)
				c.Assert(err, IsNil)
			case <-done:
				c.Fatalf("Listener exited unexpectedly")
			case <-timer.C:
				close(shutdown)
				c.Fatalf("Listener timed out")
			}
		}
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Listener timed out")
	}
	c.Assert(ch, HasLen, 1)
	node := ch[0]

	// delete
	err = conn.Delete("/net/export/tenantid/app/" + node)
	c.Assert(err, IsNil)

	ch, ev, err = conn.ChildrenW("/net/export/tenantid/app", done)
	c.Assert(err, IsNil)
	if len(ch) == 0 {
		timer.Reset(time.Second)
		select {
		case <-ev:
			ch, err = conn.Children("/net/export/tenantid/app")
			c.Assert(err, IsNil)
		case <-done:
			c.Fatalf("Listener exited unexpectedly")
		case <-timer.C:
			close(shutdown)
			c.Fatalf("Listener timed out")
		}
	}
	c.Assert(ch, HasLen, 1)
	c.Assert(ch[0], Not(Equals), node)

	// shutdown
	close(shutdown)
	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("Listener timed out")
	}
	ch, err = conn.Children("/net/export/tenantid/app")
	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 0)
}

func (t *ZZKTest) TestTrackExports(c *C) {
	// pre-requisites
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// start the listener
	shutdown := make(chan struct{})
	ev := TrackExports(shutdown, conn, "tenantid", "app")

	// get empty list
	timer := time.NewTimer(time.Second)
	select {
	case exports := <-ev:
		c.Check(exports, HasLen, 0)
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for exports")
	}

	// add an export
	export := &ExportDetails{
		ExportBinding: service.ExportBinding{Application: "app"},
		InstanceID:    0,
	}
	err = conn.Create("/net/export/tenantid/app/0", export)
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case exports := <-ev:
		c.Check(exports, HasLen, 1)
		c.Check(exports[0].InstanceID, Equals, export.InstanceID)
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for exports")
	}

	// add an export and delete the other export
	export = &ExportDetails{
		ExportBinding: service.ExportBinding{Application: "app"},
		InstanceID:    1,
	}
	err = conn.Create("/net/export/tenantid/app/1", export)
	c.Assert(err, IsNil)
	timer.Stop() // timer won't reset once it has triggered
	time.Sleep(time.Second)
	err = conn.Delete("/net/export/tenantid/app/0")
	c.Assert(err, IsNil)
	time.Sleep(time.Second)

	timer.Reset(time.Second)
	select {
	case exports := <-ev:
		c.Check(exports, HasLen, 1)
		c.Check(exports[0].InstanceID, Equals, export.InstanceID)
	case <-timer.C:
		close(shutdown)
		c.Fatalf("Timed out waiting for exports")
	}

	// shutdown
	close(shutdown)

	timer.Reset(time.Second)
	select {
	case _, ok := <-ev:
		c.Check(ok, Equals, false)
	case <-timer.C:
		c.Fatalf("Timed out waiting for exports")
	}
}
