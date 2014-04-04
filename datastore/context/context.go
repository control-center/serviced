// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package context

import (
	"github.com/zenoss/serviced/datastore/driver"
)

// Context is the context of the application or request being made
type Context interface {
	// Get a connection to the datastore
	Connection() (driver.Connection, error)
}

//Register a driver to use for the context
func Register(driver driver.Driver) {
	ctx = new(driver)
}

//Get returns the global Context
func Get() Context {
	return ctx
}

var ctx Context

//new Creates a new context with a Driver to a datastore
func new(driver driver.Driver) Context {
	return &context{driver}
}

type context struct {
	driver driver.Driver
}

func (c *context) Connection() (driver.Connection, error) {
	return c.driver.GetConnection()
}
