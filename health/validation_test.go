// Copyright 2018 The Serviced Authors.
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

// +build unit

package health_test

import (
	. "github.com/control-center/serviced/health"
	. "gopkg.in/check.v1"
)

type ValidationSuite struct{}

var _ = Suite(&ValidationSuite{})

func (vs *ValidationSuite) Test_Validation_Invalid_HealthCheck(c *C) {
	// health checks with kill exit codes must set a kill limit
	hc := HealthCheck{
		KillExitCodes: []int{28},
	}
	err := hc.ValidEntity()
	c.Assert(err, NotNil)

	// make sure this is valid if we specify the limit
	hc.KillCountLimit = 3
	err = hc.ValidEntity()
	c.Assert(err, IsNil)
}
