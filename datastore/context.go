// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package datastore

// Context is the context of the application or request being made
type Context interface {
	// Get a connection to the datastore
	Connection() (Connection, error)
}

//Register a driver to use for the context
func Register(driver Driver) {
	ctx = newCtx(driver)
}

//Get returns the global Context
func Get() Context {
	return ctx
}

var ctx Context

//new Creates a new context with a Driver to a datastore
func newCtx(driver Driver) Context {
	return &context{driver}
}

type context struct {
	driver Driver
}

func (c *context) Connection() (Connection, error) {
	return c.driver.GetConnection()
}
