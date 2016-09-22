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

// +build unit

package service_test

import (
	"github.com/control-center/serviced/domain/service"
	. "gopkg.in/check.v1"
)

type roundTest struct {
	value    float64
	expected int64
}

var roundTests = []roundTest{
	roundTest{0.49, 0},
	roundTest{0.5, 1},
	roundTest{0.99, 1},
	roundTest{1.001, 1},
	roundTest{-0.4999, 0},
	roundTest{-0.999, -1},
	roundTest{-1.001, -1},
}

func (s *ServiceDomainUnitTestSuite) TestRound(c *C) {
	for _, test := range roundTests {
		c.Assert(service.Round(test.value), Equals, test.expected)
	}
}
