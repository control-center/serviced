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
	"github.com/control-center/serviced/zzk/service"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestVHostListener(c *C) {
	// pre-reqs
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	handler := &mocks.VHostHandler{}
	listener := NewVHostListener("master", handler)
	listener.SetConnection(conn)

	// vhost is disabled
	vhost := &VHost{
		TenantID:    "tenantid",
		Application: "app",
		Enabled:     false,
	}
	err = conn.Create("/net/vhost/master/myhost", vhost)
	c.Assert(err, IsNil)

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		listener.Spawn(shutdown, "myhost")
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	select {
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
	}

	// vhost is enabled
	handler.On("Enable", "myhost").Return().Once()
	err = conn.Get("/net/vhost/master/myhost", vhost)
	c.Assert(err, IsNil)
	vhost.Enabled = true
	err = conn.Set("/net/vhost/master/myhost", vhost)
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
	}

	// exports changed
	export := &ExportDetails{
		ExportBinding: service.ExportBinding{
			Application: "app",
			Protocol:    "tcp",
			PortNumber:  6651,
		},
		HostIP:     "10.112.15.87",
		PrivateIP:  "17.147.12.128",
		MuxPort:    44181,
		InstanceID: 0,
	}
	handler.On("Set", "myhost", mock.AnythingOfType("[]registry.ExportDetails")).Return().Run(func(a mock.Arguments) {
		actual := a.Get(1).([]ExportDetails)
		c.Check(actual, HasLen, 1)
		c.Check(actual[0].ExportBinding, DeepEquals, export.ExportBinding)
	}).Once()

	err = conn.Create("/net/export/tenantid/app/0", export)
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
	}

	// shutdown
	handler.On("Disable", "myhost").Return().Once()

	close(shutdown)

	timer.Reset(time.Second)
	select {
	case <-done:
		handler.AssertExpectations(c)
	case <-timer.C:
		c.Fatalf("Listener timed out waiting to shutdown")
	}
}
