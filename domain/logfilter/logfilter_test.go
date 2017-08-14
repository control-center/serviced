// Copyright 2017 The Serviced Authors.
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

// +build unit integration

package logfilter

import (
	"testing"

	. "gopkg.in/check.v1"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

type unitTestSuite struct{}

var _ = Suite(&unitTestSuite{})

func (s *unitTestSuite) Test_GetType(c *C) {
	c.Assert(GetType(), Equals, kind)
	actual := &LogFilter{
		Name:    "filter name",
		Version: "1.0",
		Filter:  "filter it",
	}
	c.Assert(actual.GetType(), Equals, kind)
}

func (s *unitTestSuite) Test_String(c *C) {
	lf := &LogFilter{
		Name:    "filter name",
		Version: "1.0",
		Filter:  "filter it",
	}

	c.Assert(lf.String(), Equals, "filter name-1.0")
}

func (s *unitTestSuite) Test_Equals(c *C) {
	a := &LogFilter{
		Name:    "filter name",
		Version: "1.0",
		Filter:  "filter it",
	}
	a2 := *a
	c.Assert(a.Equals(a), Equals, true)
	c.Assert(a.Equals(&a2), Equals, true)

	nameDiffers := *a
	nameDiffers.Name = "foo"
	c.Assert(a.Equals(&nameDiffers), Equals, false)

	versionDiffers := *a
	versionDiffers.Version = "1.2"
	c.Assert(a.Equals(&versionDiffers), Equals, false)

	filterDiffers := *a
	filterDiffers.Filter = "something different"
	c.Assert(a.Equals(&filterDiffers), Equals, false)
}
