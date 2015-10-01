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
	"github.com/control-center/serviced/scheduler/strategy/mocks"
	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)

const (
	Gigabyte = 1024 * 1024 * 1024
)

func newHost(cores int, memgigs uint64) *mocks.Host {
	host := &mocks.Host{}
	host.On("TotalCores").Return(cores)
	host.On("TotalMemory").Return(memgigs * Gigabyte)
	id, _ := utils.NewUUID36()
	host.On("HostID").Return(id)
	return host
}

func newService(cores int, memgigs uint64) *mocks.ServiceConfig {
	id, _ := utils.NewUUID36()
	svc := &mocks.ServiceConfig{}
	svc.On("RequestedCorePercent").Return(cores * 100)
	svc.On("RequestedMemoryBytes").Return(memgigs * Gigabyte)
	svc.On("GetServiceID").Return(id)
	return svc
}

// Given two identical hosts, one of which has a service running on it, verify
// that the second host is deemed most appropriate
func (s *StrategySuite) TestSimpleScoring(c *C) {
	hostA := newHost(2, 2)
	hostB := newHost(2, 2)

	svc := newService(1, 1)
	svc2 := newService(1, 1)

	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{})

	under, over := strategy.ScoreHosts(svc2, []strategy.Host{hostA, hostB})
	c.Assert(under[0].Host, Equals, hostB)
	c.Assert(under[1].Host, Equals, hostA)
	c.Assert(over, HasLen, 0)
}

// Given two non-identical hosts, one of which has fewer overall resources but
// more free space, verify that the freer host is deemed most appropriate
func (s *StrategySuite) TestUnbalancedScoring(c *C) {
	hostA := newHost(3, 3)
	hostB := newHost(2, 2)

	svc := newService(2, 2)
	svc2 := newService(1, 1)

	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{})

	under, over := strategy.ScoreHosts(svc2, []strategy.Host{hostA, hostB})

	c.Assert(under[0].Host, Equals, hostB)
	c.Assert(under[1].Host, Equals, hostA)
	c.Assert(over, HasLen, 0)

}

// Given two identical hosts, one of which has plenty of memory but no
// available CPU, verify that it is not selected as the appropriate host
func (s *StrategySuite) TestNoCPUAvailable(c *C) {
	hostA := newHost(6, 6)
	hostB := newHost(6, 6)

	svc := newService(6, 1)
	svc2 := newService(4, 4)
	svc3 := newService(2, 2)

	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{svc2})

	under, over := strategy.ScoreHosts(svc3, []strategy.Host{hostA, hostB})

	c.Assert(under[0].Host, Equals, hostB)
	c.Assert(over[0].Host, Equals, hostA)
}

// Given two identical hosts, none of which has enough CPU free but one of
// which has enough memory, make sure the host with the memory is the one
// assigned
func (s *StrategySuite) TestNotEnoughResources(c *C) {
	hostA := newHost(5, 5)
	hostB := newHost(5, 5)

	svc := newService(4, 4)
	svc2 := newService(4, 3)
	svc3 := newService(2, 2)

	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{svc2})

	under, over := strategy.ScoreHosts(svc3, []strategy.Host{hostA, hostB})

	c.Assert(under, HasLen, 0)
	c.Assert(over[0].Host, Equals, hostB)
	c.Assert(over[1].Host, Equals, hostA)
}

// Given two identical hosts, all of which are overloaded both in terms of
// memory and CPU, make sure the one with the least memory spoken for is
// preferred over the one with the least CPU spoken for
func (s *StrategySuite) TestMemoryPreferred(c *C) {

	hostA := newHost(5, 5)
	hostB := newHost(5, 5)

	svc := newService(7, 6)
	svc2 := newService(6, 7)
	svc3 := newService(1, 1)

	hostA.On("RunningServices").Return([]strategy.ServiceConfig{svc})
	hostB.On("RunningServices").Return([]strategy.ServiceConfig{svc2})

	under, over := strategy.ScoreHosts(svc3, []strategy.Host{hostA, hostB})

	c.Assert(under, HasLen, 0)
	c.Assert(over[0].Host, Equals, hostA)
	c.Assert(over[1].Host, Equals, hostB)
}
