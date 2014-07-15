// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package volume

import (
	"testing"
)

type TestDriver struct{}

func (d TestDriver) Mount(volumeName, root string) (Conn, error) {
	return TestConn{volumeName, root}, nil
}

type TestConn struct {
	name string
	root string
}

func (c TestConn) Name() string {
	return c.name
}

func (c TestConn) Path() string {
	return c.root
}

func (c TestConn) Snapshot(label string) error {
	return nil
}

func (c TestConn) Snapshots() ([]string, error) {
	return []string{}, nil
}

func (c TestConn) RemoveSnapshot(label string) error {
	return nil
}

func (c TestConn) Rollback(label string) error {
	return nil
}

func TestNilRegistration(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// expect to recover
		}
	}()

	Register("nilregistration", nil)
	t.Fatal("nil registration didn't panic")
}

func TestRedundantRegistration(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// expect to recover
		}
	}()

	driver := TestDriver{}
	Register("redundant", driver)
	Register("redundant", driver)
	t.Fatal("redunant registration didn't panic")
}

func TestRegistration(t *testing.T) {
	Register("registration", TestDriver{})
	if _, ok := Registered("registration"); !ok {
		t.Fatal("test driver is not registered")
	}
}

func TestUnregistered(t *testing.T) {
	if _, ok := Registered("unregistered"); ok {
		t.Fatal("xyzzy should not be registered")
	}
}

func TestMount(t *testing.T) {
	driver := TestDriver{}
	Register("testmount", driver)
	v, err := Mount("testmount", "testmount", "/opt/testmount")
	switch {
	case err != nil:
		t.Fatalf("Mount failed: %v", err)
	case v == nil:
		t.Fatal("nil volume")
	}
}

func TestBadMount(t *testing.T) {
	if _, err := Mount("badmount", "badmount", "/opt/badmount"); err == nil {
		t.Fatal("bad mount should not suceed")
	}
}
