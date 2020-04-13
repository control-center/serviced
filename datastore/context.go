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

import (
	"github.com/control-center/serviced/metrics"
)

// Context is the context of the application or request being made
type Context interface {
	// Returns a connection to the datastore
	Connection() (Connection, error)

	// Returns the Metrics object from the context
	Metrics() *metrics.Metrics

	// WithUser returns a copy of the current Context configured with the given user.
	WithUser(user string) Context

	// User returns the user name for audit logging
	User() string
}

type context struct {
	driver  Driver
	metrics *metrics.Metrics
	user    string
}

var (
	ctx         Context
	savedDriver Driver
)

//Register a driver to use for the context
func Register(driver Driver) {
	savedDriver = driver
	ctx = newContext()
}

//GetContext returns the global Context
func GetContext() Context {
	return ctx
}

// newContext returns a new global context.
// This function is not intended for production use, but is for the purpose
// of getting fresh contexts for performance testing with metrics for troubleshooting.
func newContext() Context {
	return &context{savedDriver, metrics.NewMetrics(), "system"}
}

func (c *context) Connection() (Connection, error) {
	return c.driver.GetConnection()
}

func (c *context) Metrics() *metrics.Metrics {
	return c.metrics
}

func (c *context) WithUser(user string) Context {
	return &context{c.driver, c.metrics, user}
}

func (c *context) User() string {
	return c.user
}
