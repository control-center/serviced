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
