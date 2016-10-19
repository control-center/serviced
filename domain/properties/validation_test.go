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

package properties

import (
	. "gopkg.in/check.v1"
)

type validationSuite struct{}

var _ = Suite(&validationSuite{})

func (s *validationSuite) Test_Success(c *C) {
     props := New()
     props.SetCCVersion("9.9.9")
     err := props.ValidEntity()
     c.Assert(err, IsNil)
}

func (s *validationSuite) Test_NoVersion(c *C) {
     props := New()
     err := props.ValidEntity()
     c.Assert(err, NotNil)
}

func (s *validationSuite) Test_NoProps(c *C) {
     props := &StoredProperties{}
     err := props.ValidEntity()
     c.Assert(err, NotNil)
}

