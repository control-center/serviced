// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package context

import (
	"github.com/zenoss/serviced/datastore/key"
	"github.com/zenoss/serviced/datastore/driver"

	"testing"
)

type testDriver struct{}

func (d *testDriver) GetConnection() (driver.Connection, error) {
	return &testConn{}, nil
}

type testConn struct{}

func (c testConn) Put(key key.Key, data driver.JSONMessage) error {
	return nil
}

func (c testConn) Get(key key.Key) (driver.JSONMessage, error) {
	return nil, nil
}

func (c testConn) Delete(key key.Key) error {
	return nil
}

func (c testConn) Query(interface{}) ([]driver.JSONMessage, error) {
	return nil, nil
}

func TestContext(t *testing.T) {

	driver := testDriver{}
	ctx := new(&driver)

	conn, _ := ctx.Connection()
	if conn == nil {
		t.Error("Expected connection, got nil")
	}
}
