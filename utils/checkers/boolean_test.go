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

package checkers_test

import (
	. "github.com/control-center/serviced/utils/checkers"
	. "gopkg.in/check.v1"
)

// Unit Tests to check the IsTrue Checker

var _ = Suite(&IsTrueTests{})

type IsTrueTests struct{}

func (t *IsTrueTests) TestIsTrueReturnsErrorWhenNotGivenBoolean(c *C) {
	values := make([]interface{}, 1)
	values[0] = "Not a boolean"

	_, err := IsTrue.Check(values, []string{})
	c.Assert(err, NotNil)
}

func (t *IsTrueTests) TestIsTruePassesForTrue(c *C) {
	values := make([]interface{}, 1)
	values[0] = true

	result, err := IsTrue.Check(values, []string{})
	c.Assert(err, HasLen, 0)
	c.Assert(result, Equals, true)
}

func (t *IsTrueTests) TestIsTrueFailsForFalse(c *C) {
	values := make([]interface{}, 1)
	values[0] = false

	result, err := IsTrue.Check(values, []string{})
	c.Assert(err, HasLen, 0)
	c.Assert(result, Equals, false)
}

// Unit tests to check the IsFalse checker

var _ = Suite(&IsFalseTests{})

type IsFalseTests struct{}

func (t *IsFalseTests) TestIsFalseReturnsErrorWhenNotGivenBoolean(c *C) {
	values := make([]interface{}, 1)
	values[0] = "Not a boolean"

	_, err := IsFalse.Check(values, []string{})
	c.Assert(err, NotNil)
}

func (t *IsFalseTests) TestIsFalseFailsForTrue(c *C) {
	values := make([]interface{}, 1)
	values[0] = true

	result, err := IsFalse.Check(values, []string{})
	c.Assert(err, HasLen, 0)
	c.Assert(result, Equals, false)
}

func (t *IsFalseTests) TestIsFalsePassesForFalse(c *C) {
	values := make([]interface{}, 1)
	values[0] = false

	result, err := IsFalse.Check(values, []string{})
	c.Assert(err, HasLen, 0)
	c.Assert(result, Equals, true)
}
