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

// +build unit

package logfilter

import (
	. "gopkg.in/check.v1"
)

type validationSuite struct{}

var _ = Suite(&validationSuite{})


func (s *validationSuite) TestLogFilter_Success(c *C) {
	filter := LogFilter{
		Name:    "filter name",
		Version: "1.0",
		Filter:  "filter it",
	}
	err := filter.ValidEntity()
	c.Assert(err, IsNil)
}

func (s *validationSuite) TestLogFilter_NonBlankName(c *C) {
	filter := LogFilter{
		Name:    "",
		Version: "1.0",
		Filter:  "filter it",
	}
	err := filter.ValidEntity()
	c.Assert(err, NotNil)
}

func (s *validationSuite) TestLogFilter_NonBlankVersion(c *C) {
	filter := LogFilter{
		Name:    "filter name",
		Version: "",
		Filter:  "filter it",
	}
	err := filter.ValidEntity()
	c.Assert(err, NotNil)
}

func (s *validationSuite) TestLogFilter_NonBlankFilter(c *C) {
	filter := LogFilter{
		Name:    "filter name",
		Version: "1.0",
		Filter:  "",
	}
	err := filter.ValidEntity()
	c.Assert(err, NotNil)
}
