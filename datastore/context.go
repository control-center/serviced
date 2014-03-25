// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

// Context is the context of the application or request being made
type Context interface {
	// Get a connection to the datastore
	Connection() (Connection, error)
}

//Creates a new context with an Driver to a datastore
func NewContext(driver Driver) Context {
	return &context{driver}
}

type context struct {
	driver Driver
}

func (c *context) Connection() (Connection, error) {
	return c.driver.GetConnection()
}
