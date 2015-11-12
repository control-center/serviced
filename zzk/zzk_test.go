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

package zzk

import (
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

var _ = Suite(&ZZKTest{})

type ZZKTest struct {
	ZZKTestSuite
}

func Test(t *testing.T) {
	TestingT(t)
}

func (t *ZZKTest) TestPathExists(c *C) {
	conn, err := GetLocalConnection("/")
	if err != nil {
		c.Fatalf("Could not connect: %s", err)
	}

	// Path not exists
	c.Log("Test non-existant path")
	exists, err := PathExists(conn, "/test")
	if err != nil {
		c.Errorf("Unexpected error when checking a non-existant path: %s", err)
	}
	if exists {
		c.Errorf("Path found!")
	}

	// Path exists
	c.Log("Test existing path")
	if err := conn.CreateDir("/test"); err != nil {
		c.Fatalf("Error creating node: %s", err)
	}
	exists, err = PathExists(conn, "/test")
	if err != nil {
		c.Errorf("Unexpected error when checking an existing path: %s", err)
	}
	if !exists {
		c.Errorf("Path not found!")
	}
}

func (t *ZZKTest) TestReady(c *C) {
	conn, err := GetLocalConnection("/")
	if err != nil {
		c.Fatalf("Could not connect: %s", err)
	}

	path := "/test/some/path"
	errC := make(chan error)

	c.Log("Testing shutdown")
	shutdown := make(chan interface{})
	go func() {
		errC <- Ready(shutdown, conn, path)
	}()

	time.Sleep(time.Second)
	close(shutdown)
	select {
	case err := <-errC:
		c.Assert(err, Equals, ErrShutdown)
	case <-time.After(ZKTestTimeout):
		c.Errorf("timeout waiting for shutdown")
	}

	c.Log("Testing path found")
	go func() {
		errC <- Ready(nil, conn, path)
	}()

	time.Sleep(time.Second)
	if err := conn.CreateDir(path); err != nil {
		c.Fatalf("could not create path %s: %s", path, err)
	}
	select {
	case err := <-errC:
		c.Assert(err, IsNil)
	case <-time.After(ZKTestTimeout):
		c.Errorf("timeout waiting for signal")
	}

	// Test connection to a non-existing path
	conn, err = GetLocalConnection("/notexists")
	if err != nil {
		c.Fatalf("Could not connect: %s", err)
	}

	path = "/test/some/path"
	go func() {
		errC <- Ready(nil, conn, path)
	}()
	select {
	case err := <-errC:
		c.Assert(err, NotNil)
	case <-time.After(time.Second):
		c.Errorf("timeout waiting for signal")
	}

}
