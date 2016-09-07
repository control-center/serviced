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

package service_test

import (
	"time"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/service"
	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestWaitService(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add 1 service
	err = conn.CreateDir("/pools/poolid/services/serviceid")
	c.Assert(err, IsNil)

	// add 1 host
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	// no states
	shutdown := make(chan struct{})
	done := make(chan struct{})
	go func() {
		checkCount := func(count int) bool {
			return count == 1
		}

		checkState := func(s *State, ok bool) bool {
			if !ok {
				return false
			}
			return s.DesiredState == service.SVCRun && s.Started.After(s.Terminated)
		}

		err := WaitService(shutdown, conn, "poolid", "serviceid", checkCount, checkState)
		c.Assert(err, IsNil)
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
	}

	// create the state
	req := StateRequest{
		PoolID:     "poolid",
		HostID:     "hostid",
		ServiceID:  "serviceid",
		InstanceID: 0,
	}

	err = CreateState(conn, req)
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
	}

	// the state is "running"
	err = UpdateState(conn, req, func(s *State) bool {
		s.Started = time.Now()
		return true
	})
	c.Assert(err, IsNil)

	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("Timed out waiting for listener")
	}
}
