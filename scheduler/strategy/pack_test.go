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

func (s *StrategySuite) TestSimplePack(c *C) {

	hostA := newHost(5, 5)
	hostB := newHost(5, 5)

	svc := newService(3, 3)
	svc2 := newService(1, 1)
	svc3 := newService(1, 1)
	svc4 := newService(1, 1)
	svc5 := newService(1, 1)

	svc6 := newService(1, 1)

	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{svc2, svc3, svc4, svc5})

	strat := strategy.PackStrategy{}

	host, err := strat.SelectHost(svc6, []strategy.Host{hostA, hostB})
	c.Assert(err, IsNil)
	c.Assert(host, Equals, hostB)

}

func (s *StrategySuite) TestPackMoreContainers(c *C) {
	hostA := newHost(5, 5)
	hostB := newHost(5, 5)

	svc := newService(3, 3)
	svc2 := newService(1, 1)
	svc3 := newService(1, 1)
	svc4 := newService(1, 1)

	svc5 := newService(1, 1)

	// Evenly load both hosts, but one has 3 containers vs. 1
	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{svc2, svc3, svc4})

	strat := strategy.PackStrategy{}

	host, err := strat.SelectHost(svc5, []strategy.Host{hostA, hostB})
	c.Assert(err, IsNil)
	c.Assert(host, Equals, hostB)
}

func (s *StrategySuite) TestOversubscribed(c *C) {
	hostA := newHost(2, 5)
	hostB := newHost(2, 5)

	svc := newService(3, 3)
	svc2 := newService(2, 2)

	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{svc2})

	svc3 := newService(1, 1)

	strat := strategy.PackStrategy{}

	host, err := strat.SelectHost(svc3, []strategy.Host{hostA, hostB})
	c.Assert(err, IsNil)
	c.Assert(host, Equals, hostB)
}
