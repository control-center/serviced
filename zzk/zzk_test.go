// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package zzk

import (
	"sync"
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client"
)

func TestPathExists(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()

	// Path not exists
	t.Log("Test non-existant path")
	exists, err := PathExists(conn, "/test")
	if err != nil {
		t.Errorf("Unexpected error when checking a non-existant path: %s", err)
	}
	if exists {
		t.Errorf("Path found!")
	}

	// Path exists
	t.Log("Test existing path")
	if err := conn.CreateDir("/test"); err != nil {
		t.Fatalf("Error creating node: %s", err)
	}
	exists, err = PathExists(conn, "/test")
	if err != nil {
		t.Errorf("Unexpected error when checking an existing path: %s", err)
	}
	if !exists {
		t.Errorf("Path not found!")
	}
}

func TestReady(t *testing.T) {
	conn := client.NewTestConnection()
	defer conn.Close()

	path := "/test/some/path"

	var (
		wg  sync.WaitGroup
		err error
	)

	// Test shutdown
	shutdown := make(chan interface{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = Ready(shutdown, conn, path)
	}()

	<-time.After(3 * time.Second)
	t.Log("Testing shutdown")
	close(shutdown)
	wg.Wait()
	if err != ErrShutdown {
		t.Errorf("Expected: %s; Got: %s", ErrShutdown, err)
	}

	// Test path found
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = Ready(make(<-chan interface{}), conn, path)
	}()

	<-time.After(3 * time.Second)
	t.Log("Testing path found")
	if err := conn.CreateDir(path); err != nil {
		t.Fatalf("Error trying to create path: %s", err)
	}
	wg.Wait()
	if err != nil {
		t.Errorf("Error checking path: %s", err)
	}
}