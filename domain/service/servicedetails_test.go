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

// +build unit

package service_test

import (
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/service"
	. "gopkg.in/check.v1"
)

func (s *ServiceDomainUnitTestSuite) TestValidServiceDetails(c *C) {
	details := service.ServiceDetails{
		ID:        "1234566789",
		Name:      "Service",
		PoolID:    "Pool",
		Instances: 1,
		InstanceLimits: domain.MinMax{
			Min:     0,
			Max:     2,
			Default: 0,
		},
	}

	c.Assert(details.ValidEntity(), IsNil)
}

func (s *ServiceDomainUnitTestSuite) TestValidServiceDetailsWithNoMaxInstances(c *C) {
	details := service.ServiceDetails{
		ID:        "1234566789",
		Name:      "Service",
		PoolID:    "Pool",
		Instances: 1,
		InstanceLimits: domain.MinMax{
			Min:     0,
			Max:     0,
			Default: 0,
		},
	}

	c.Assert(details.ValidEntity(), IsNil)
}

func (s *ServiceDomainUnitTestSuite) TestInvalidServiceDetailsID(c *C) {
	details := service.ServiceDetails{
		Name:      "Service",
		PoolID:    "Pool",
		Instances: 1,
		InstanceLimits: domain.MinMax{
			Min:     0,
			Max:     2,
			Default: 0,
		},
	}

	c.Assert(details.ValidEntity(), Not(IsNil))
}

func (s *ServiceDomainUnitTestSuite) TestInvalidServiceDetailsName(c *C) {
	details := service.ServiceDetails{
		ID:        "1234566789",
		PoolID:    "Pool",
		Instances: 1,
		InstanceLimits: domain.MinMax{
			Min:     0,
			Max:     2,
			Default: 0,
		},
	}

	c.Assert(details.ValidEntity(), Not(IsNil))
}

func (s *ServiceDomainUnitTestSuite) TestInvalidServiceDetailsPoolID(c *C) {
	details := service.ServiceDetails{
		ID:        "1234566789",
		Name:      "Service",
		Instances: 1,
		InstanceLimits: domain.MinMax{
			Min:     0,
			Max:     2,
			Default: 0,
		},
	}

	c.Assert(details.ValidEntity(), Not(IsNil))
}

func (s *ServiceDomainUnitTestSuite) TestInvalidServiceDetailsInstancesWithNoMax(c *C) {
	details := service.ServiceDetails{
		ID:        "1234566789",
		Name:      "Service",
		PoolID:    "Pool",
		Instances: 1,
		InstanceLimits: domain.MinMax{
			Min:     2,
			Max:     0,
			Default: 0,
		},
	}

	c.Assert(details.ValidEntity(), Not(IsNil))
}

func (s *ServiceDomainUnitTestSuite) TestInvalidServiceDetailsInstancesOutsideRange(c *C) {
	details := service.ServiceDetails{
		ID:        "1234566789",
		Name:      "Service",
		PoolID:    "Pool",
		Instances: 5,
		InstanceLimits: domain.MinMax{
			Min:     1,
			Max:     3,
			Default: 0,
		},
	}

	c.Assert(details.ValidEntity(), Not(IsNil))
}

func (s *ServiceDomainUnitTestSuite) TestValidServiceDetailsWithZeroInstances(c *C) {
	details := service.ServiceDetails{
		ID:        "1234566789",
		Name:      "Service",
		PoolID:    "Pool",
		Instances: 0,
		InstanceLimits: domain.MinMax{
			Min:     1,
			Max:     3,
			Default: 1,
		},
	}

	c.Assert(details.ValidEntity(), IsNil)
}
