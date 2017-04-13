// Copyright 2014 The Serviced Authors.
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
	"time"

	"github.com/control-center/serviced/coordinator/client"
	. "github.com/control-center/serviced/zzk"

	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestLeaderListener(c *C) {
	conn, err := GetLocalConnection("/")
	c.Assert(err, IsNil)

	err = conn.CreateDir("/leader")
	c.Assert(err, IsNil)

	listener := NewLeaderListener("/leader")
	ch := listener.Wait()

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		listener.Run(shutdown, conn)
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ch:
		c.Errorf("Unexpected event from listener")
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
	}

	// set up a leader
	leader, err := conn.NewLeader("/leader")
	c.Assert(err, IsNil)
	_, err = leader.TakeLead(&client.Dir{}, done)
	c.Assert(err, IsNil)

	// assert that the wait channel was enabled
	timer.Reset(time.Second)
	select {
	case <-ch:
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
		c.Fatalf("Timed out waiting for listener")
	}

	if !timer.Stop() {
		<-timer.C
	}

	// set up another leader out of band
	go func() {
		leader, err := conn.NewLeader("/leader")
		c.Assert(err, IsNil)
		leader.TakeLead(&client.Dir{}, done)
	}()

	c.Assert(leader.ReleaseLead(), IsNil)
	timer.Reset(time.Second)
	select {
	case <-listener.Wait():
	case <-done:
		c.Fatalf("Listener exited unexpectedly")
	case <-timer.C:
		c.Fatalf("Timed out waiting for listener")
	}

	if !timer.Stop() {
		<-timer.C
	}

	// shut down
	timer.Reset(time.Second)
	close(shutdown)
	select {
	case <-listener.Wait():
		c.Errorf("Received unexpected event")
	case <-done:
	case <-timer.C:
		c.Fatal("Timed out waiting to shutdown")
	}
}
