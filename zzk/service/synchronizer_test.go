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

package service_test

import (
	p "github.com/control-center/serviced/domain/pool"
	. "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/service/mocks"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

var _ = Suite(&SynchronizerTestSuite{})

type SynchronizerTestSuite struct {
	handler      mocks.AssignmentHandler
	synchronizer VirtualIPSynchronizer
	pool         p.ResourcePool
}

func (s *SynchronizerTestSuite) SetUpTest(c *C) {
	s.handler = mocks.AssignmentHandler{}

	s.handler.On("Assign", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.handler.On("Unassign", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	s.pool = p.ResourcePool{ID: "pool", VirtualIPs: []p.VirtualIP{}}
	s.synchronizer = NewZKVirtualIPSynchronizer(&s.handler)
}

func (s *SynchronizerTestSuite) TestSyncsVirtualIPAssignments(c *C) {
	// Two virtual IPs, 1.2.3.4 and 7.7.7.7
	s.pool.VirtualIPs = []p.VirtualIP{
		p.VirtualIP{
			PoolID:        "pool",
			IP:            "1.2.3.4",
			Netmask:       "255.255.255.0",
			BindInterface: "eth1",
		},
		p.VirtualIP{
			PoolID:        "pool",
			IP:            "7.7.7.7",
			Netmask:       "255.255.255.0",
			BindInterface: "eth1",
		},
	}

	// Two assignments, 4.3.2.1 and 7.7.7.7
	assignments := map[string]string{
		"4.3.2.1": "host",
		"7.7.7.7": "host",
	}

	cancel := make(<-chan interface{})
	s.synchronizer.Sync(s.pool, assignments, cancel)

	// 1.2.3.4 has a virtual IP but is not assigned so Assign should be called
	s.handler.AssertCalled(c, "Assign", s.pool.ID, "1.2.3.4", "255.255.255.0", "eth1", cancel)

	// 4.3.2.1 has an assignment but does not have a virtual IP so it should be unassigned
	s.handler.AssertCalled(c, "Unassign", s.pool.ID, "4.3.2.1")

	// 7.7.7.7 has an assignment and virtual IP so nothing should be done
	s.handler.AssertNotCalled(c, "Assign", s.pool.ID, "7.7.7.7", "255.255.255.0", "eth1", cancel)
	s.handler.AssertNotCalled(c, "Unassign", s.pool.ID, "7.7.7.7")

}

func (s *SynchronizerTestSuite) TestSyncsAssignsAllIfNoneAssigned(c *C) {
	// Two virtual IPs, 1.2.3.4 and 7.7.7.7
	s.pool.VirtualIPs = []p.VirtualIP{
		p.VirtualIP{
			PoolID:        "pool",
			IP:            "1.2.3.4",
			Netmask:       "255.255.255.0",
			BindInterface: "eth1",
		},
		p.VirtualIP{
			PoolID:        "pool",
			IP:            "7.7.7.7",
			Netmask:       "255.255.255.0",
			BindInterface: "eth1",
		},
	}

	// No assignments
	assignments := map[string]string{}

	cancel := make(<-chan interface{})
	s.synchronizer.Sync(s.pool, assignments, cancel)

	s.handler.AssertCalled(c, "Assign", s.pool.ID, "1.2.3.4", "255.255.255.0", "eth1", cancel)
	s.handler.AssertCalled(c, "Assign", s.pool.ID, "7.7.7.7", "255.255.255.0", "eth1", cancel)
}

func (s *SynchronizerTestSuite) TestSyncUnassignsAll(c *C) {
	// No virtual ips
	s.pool.VirtualIPs = nil

	// Two assignments, 4.3.2.1 and 7.7.7.7
	assignments := map[string]string{
		"4.3.2.1": "host",
		"7.7.7.7": "host",
	}

	cancel := make(<-chan interface{})
	s.synchronizer.Sync(s.pool, assignments, cancel)

	s.handler.AssertCalled(c, "Unassign", s.pool.ID, "4.3.2.1")
	s.handler.AssertCalled(c, "Unassign", s.pool.ID, "7.7.7.7")
}
