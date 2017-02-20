// Copyright 2017 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build unit

package virtualips_test

import (
	. "github.com/control-center/serviced/zzk/virtualips"
	. "gopkg.in/check.v1"
)

var _ = Suite(&ZKPathUnitTestSuite{})

type ZKPathUnitTestSuite struct{}

func (s *ZKPathUnitTestSuite) TestPathReturnsCorrectValuesForNodes(c *C) {
	var values = []struct {
		actual   string
		expected string
	}{
		{Base().Path(), "/"},
		{Base().Pools().Path(), "/pools"},
		{Base().Hosts().Path(), "/hosts"},
		{Base().VirtualIPs().Path(), "/virtualIPs"},
		{Base().IPs().Path(), "/ips"},
		{Base().Online().Path(), "/online"},
	}

	for _, v := range values {
		c.Assert(v.actual, Equals, v.expected)
	}
}

func (s *ZKPathUnitTestSuite) TestPathReturnsPathWithID(c *C) {
	c.Assert(Base().Pools().ID("poolID").Path(), Equals, "/pools/poolID")
}

func (s *ZKPathUnitTestSuite) TestPathReturnsPathWhenIDHasExtraForwardSlashes(c *C) {
	c.Assert(Base().Pools().ID("/poolID/").Path(), Equals, "/pools/poolID")
}

func (s *ZKPathUnitTestSuite) TestEmptyIDDoesNotAppendAnything(c *C) {
	c.Assert(Base().Pools().ID("").Hosts().Path(), Equals, "/pools/hosts")
}

func (s *ZKPathUnitTestSuite) TestPathDoesNotChangePreExistingPath(c *C) {
	pools := Base().Pools()
	hosts := pools.ID("test").Hosts()

	c.Assert(pools.Path(), Equals, "/pools")
	c.Assert(hosts.Path(), Equals, "/pools/test/hosts")
}
