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

// +build integration

package service

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
	"runtime"
	"time"

	. "gopkg.in/check.v1"
)

func getTestConn(c *C, path string) client.Connection {
	root, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	err = root.CreateDir(path)
	c.Assert(err, IsNil)
	conn, err := zzk.GetLocalConnection(path)
	c.Assert(err, IsNil)
	return conn
}

// Test that WaitServiceLock returns no error if the lock is already held
func (t *ZZKTest) TestWaitServiceLockAnte(c *C) {
	conn := getTestConn(c, "/ante")

	lock := ServiceLock(conn)
	lock.Lock()
	defer lock.Unlock()

	cancel := make(chan interface{})
	go func() {
		defer close(cancel)
		<-time.After(time.Second)
	}()

	err := WaitServiceLock(cancel, conn, true)
	c.Assert(err, IsNil)
}

// Test that WaitServiceLock returns properly if the lock is acquired
//  while we are waiting
func (t *ZZKTest) TestWaitServiceLockPost(c *C) {
	conn := getTestConn(c, "/post")

	cancel := make(chan interface{})
	go func() {
		defer close(cancel)
		<-time.After(5 * time.Second)
	}()

	// Wait one second, then acquire a lock
	wait := time.Second
	exit := make(chan interface{})
	defer close(exit)
	go func() {
		<-time.After(wait)
		lock := ServiceLock(conn)
		lock.Lock()
		<-exit
		lock.Unlock()
	}()

	// Wait for lock; this should take about one second
	before := time.Now()
	err := WaitServiceLock(nil, conn, true)
	dt := time.Since(before)
	c.Assert(err, IsNil)

	// Check that it took about one second
	expected := 200 * time.Millisecond
	if dt < wait-expected || dt > wait+expected {
		c.Errorf("Failed to wait for service lock.  Expected wait %v; actual %v", wait, dt)
	}
}

// Test cancelling call to WaitServiceLock
func (t *ZZKTest) TestWaitServiceLockCancel(c *C) {
	conn := getTestConn(c, "/cancel")

	cancel := make(chan interface{})
	go func() {
		defer close(cancel)
		<-time.After(time.Second)
	}()

	err := WaitServiceLock(cancel, conn, true)
	c.Assert(err, NotNil)
}

// Test EnsureServiceLocked will acquire lock and release it at exit channel close
func (t *ZZKTest) TestEnsureServiceLock(c *C) {
	conn := getTestConn(c, "/ensure")

	locked, err := IsServiceLocked(conn)
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, false)

	exit := make(chan interface{})
	err = EnsureServiceLock(nil, exit, conn)
	c.Assert(err, IsNil)

	locked, err = IsServiceLocked(conn)
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, true)

	// Release lock
	close(exit)

	done := false
	interval := time.Tick(1 * time.Millisecond)
	timeout := time.Tick(10 * time.Second)
	for {
		select {
		case <-interval:
			locked, _ = IsServiceLocked(conn)
			if !locked {
				done = true
			}
		case <-timeout:
			done = true
		}
		if done {
			break
		}
	}

	locked, err = IsServiceLocked(conn)
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, false)
}

// Test EnsureServiceLocked handles overlapping locks
func (t *ZZKTest) TestEnsureServiceLockOverlap(c *C) {
	conn := getTestConn(c, "/overlap")

	locked, err := IsServiceLocked(conn)
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, false)

	lock := ServiceLock(conn)
	lock.Lock()

	exit := make(chan interface{})
	err = EnsureServiceLock(nil, exit, conn)
	c.Assert(err, IsNil)

	locked, err = IsServiceLocked(conn)
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, true)

	lock.Unlock()

	locked, err = IsServiceLocked(conn)
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, true)

	// Release lock
	close(exit)
	runtime.Gosched()

	locked, err = IsServiceLocked(conn)
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, false)
}
