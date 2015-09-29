// Copyright 2015 The Serviced Authors.
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

package strategy_test

import (
	"github.com/control-center/serviced/scheduler/strategy"
	. "gopkg.in/check.v1"
)

func (s *StrategySuite) TestSimpleRequireSeparate(c *C) {

	hostA := newHost(5, 5)
	hostB := newHost(5, 5)

	// One host with 0 instances of service and few resources, the other with
	// lots of resources and 1 instance of service
	svc := newService(4, 4)
	svc2 := newService(1, 1)

	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{svc2})

	strat := strategy.RequireSeparateStrategy{}

	// Verify that it ended up on the host with no instances of the service,
	// despite more free resources on the other host
	host, err := strat.SelectHost(svc2, []strategy.Host{hostA, hostB})
	c.Assert(err, IsNil)
	c.Assert(host, Equals, hostA)
}

func (s *StrategySuite) TestRequireSeparateOversubscribed(c *C) {

	hostOversubscribed := newHost(1, 1)
	hostA := newHost(5, 5)
	hostB := newHost(5, 5)
	hostC := newHost(5, 5)

	// Oversubscribe the first host with an unrelated service
	svc := newService(1, 1)
	hostOversubscribed.On("RunningServices").Return([]strategy.ServiceConfig{svc})

	// Load up the rest of the hosts with different numbers of svc2, then fill
	// up with svc to make them all equal
	svc2 := newService(1, 1)
	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc2, svc2, svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{svc2, svc, svc})
	hostC.On("RunningServices").Return([]strategy.ServiceConfig{svc2, svc2, svc2})

	strat := strategy.RequireSeparateStrategy{}

	// Verify that it ended up on the host with fewest instances of the service,
	// despite more free resources on the other hosts
	host, err := strat.SelectHost(svc2, []strategy.Host{hostA, hostB, hostC, hostOversubscribed})
	c.Assert(err, IsNil)
	c.Assert(host, Equals, hostOversubscribed)
}

func (s *StrategySuite) TestNoAvailableHost(c *C) {
	hostA := newHost(2, 5)
	hostB := newHost(2, 6)

	svc := newService(3, 3)

	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{svc})

	strat := strategy.RequireSeparateStrategy{}

	host, err := strat.SelectHost(svc, []strategy.Host{hostA, hostB})
	c.Assert(err, IsNil)
	c.Assert(host, IsNil)
}
