// Copyright 2016 The Serviced Authors.
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

package checkers

import (
	c "gopkg.in/check.v1"
)

// ------------------------------------------------------------------
// IsTrue Checker

type isTrueChecker struct {
	*c.CheckerInfo
}

// IsTrue tests whether the obtained value is true
//
// Example: c.Assert(value, IsTrue)
//
var IsTrue c.Checker = &isTrueChecker{&c.CheckerInfo{Name: "IsTrue", Params: []string{"value"}}}

func (c *isTrueChecker) Check(params []interface{}, names []string) (result bool, error string) {
	var ok bool
	var first bool

	first, ok = params[0].(bool)
	if !ok {
		return false, "value is not a boolean"
	}
	return first, ""
}

// ------------------------------------------------------------------
// IsFalse Checker

type isFalseChecker struct {
	*c.CheckerInfo
}

// IsFalse tests whether the obtained value is false
//
// Example: c.Assert(value, IsFalse)
//
var IsFalse c.Checker = &isFalseChecker{&c.CheckerInfo{Name: "IsFalse", Params: []string{"value"}}}

func (c *isFalseChecker) Check(params []interface{}, names []string) (result bool, error string) {
	var ok bool
	var first bool

	first, ok = params[0].(bool)
	if !ok {
		return false, "value is not a boolean"
	}
	return !first, ""
}
