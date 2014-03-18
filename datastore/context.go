// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

type Context interface {
	Connection() (Connection, error)
}

type context struct {
	driver Driver
}

func NewContext(driver Driver) Context {
	return &context{driver}
}

func (c *context) Connection() (Connection, error) {
	return c.driver.GetConnection()
}
