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

package virtualips_test

import (
	h "github.com/control-center/serviced/domain/host"
	. "github.com/control-center/serviced/zzk/virtualips"

	. "github.com/control-center/serviced/utils/checkers"
	. "gopkg.in/check.v1"
)

var _ = Suite(&RandomHostSelectionTestSuite{})

type RandomHostSelectionTestSuite struct {
	randomStrategy HostSelectionStrategy
}

func (s *RandomHostSelectionTestSuite) SetUpSuite(c *C) {
	s.randomStrategy = &RandomHostSelectionStrategy{}
}

func (s *RandomHostSelectionTestSuite) TestSelectsAHost(c *C) {
	hosts := []h.Host{
		h.Host{ID: "host1"},
		h.Host{ID: "host2"},
		h.Host{ID: "host3"},
	}

	selectedHost, err := s.randomStrategy.Select(hosts)
	found := false
	for _, host := range hosts {
		if selectedHost.ID == host.ID {
			found = true
			break
		}
	}

	c.Assert(found, IsTrue)
	c.Assert(err, IsNil)
}

func (s *RandomHostSelectionTestSuite) TestSelectEmptyHostListReturnsErrNoHosts(c *C) {
	_, err := s.randomStrategy.Select([]h.Host{})
	c.Assert(err, Equals, ErrNoHosts)
}

func (s *RandomHostSelectionTestSuite) TestSelectSingleHostReturnsThatHost(c *C) {
	hosts := []h.Host{h.Host{ID: "host1"}}
	selectedHost, err := s.randomStrategy.Select(hosts)
	c.Assert(selectedHost.ID, Equals, "host1")
	c.Assert(err, IsNil)
}
