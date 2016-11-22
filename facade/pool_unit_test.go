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

package facade_test

import (
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (ft *FacadeUnitTest) Test_GetResourcePool(c *C) {
	ft.setupMockDFSLocking()
	poolID := "somePoolID"
	expectedPool := pool.ResourcePool{ID: poolID}
	key := pool.Key(poolID)
	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, mock.AnythingOfType("string")).
		Return([]host.Host{
			{ID: "host1", Cores: 2, Memory: 3},
			{ID: "host2", Cores: 3, Memory: 4},
		}, nil)
	ft.poolStore.On("Get", ft.ctx, key, mock.AnythingOfType("*pool.ResourcePool")).
		Return(nil).
		Run(func(args mock.Arguments) {
			pool := args.Get(2).(*pool.ResourcePool)
			*pool = expectedPool
		})

	result, err := ft.Facade.GetResourcePool(ft.ctx, poolID)

	c.Assert(err, IsNil)
	c.Assert(result.ID, Equals, poolID)
	c.Assert(result.CoreCapacity, Equals, 5)
	c.Assert(result.MemoryCapacity, Equals, uint64(7))
}

func (ft *FacadeUnitTest) Test_GetResourcePoolWithNoHosts(c *C) {
	ft.setupMockDFSLocking()
	poolID := "somePoolID"
	expectedPool := pool.ResourcePool{ID: poolID}
	key := pool.Key(poolID)
	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, mock.AnythingOfType("string")).
		Return([]host.Host{}, nil)
	ft.poolStore.On("Get", ft.ctx, key, mock.AnythingOfType("*pool.ResourcePool")).
		Return(nil).
		Run(func(args mock.Arguments) {
			pool := args.Get(2).(*pool.ResourcePool)
			*pool = expectedPool
		})

	result, err := ft.Facade.GetResourcePool(ft.ctx, poolID)

	c.Assert(err, IsNil)
	c.Assert(result, Not(IsNil))
	c.Assert(result.ID, Equals, poolID)
	c.Assert(result.CoreCapacity, Equals, 0)
	c.Assert(result.MemoryCapacity, Equals, uint64(0))
}

func (ft *FacadeUnitTest) Test_GetResourcePoolFailsForNoSuchEntity(c *C) {
	ft.setupMockDFSLocking()
	poolID := "somePoolID"
	key := pool.Key(poolID)

	ft.poolStore.On("Get", ft.ctx, key, mock.AnythingOfType("*pool.ResourcePool")).Return(datastore.ErrNoSuchEntity{})

	result, err := ft.Facade.GetResourcePool(ft.ctx, poolID)

	c.Assert(result, IsNil)
	c.Assert(err, IsNil)
}

func (ft *FacadeUnitTest) Test_GetResourcePoolFailsForOtherDBError(c *C) {
	ft.setupMockDFSLocking()
	poolID := "somePoolID"
	key := pool.Key(poolID)
	expectedError := datastore.ErrEmptyKind

	ft.poolStore.On("Get", ft.ctx, key, mock.AnythingOfType("*pool.ResourcePool")).Return(expectedError)

	result, err := ft.Facade.GetResourcePool(ft.ctx, poolID)

	c.Assert(result, IsNil)
	c.Assert(err, Equals, expectedError)
}

func (ft *FacadeUnitTest) Test_GetReadPoolsShouldReturnCorrectValues(c *C) {
	ft.setupMockDFSLocking()

	resourcePool := pool.ResourcePool{
		ID:                "firstPool",
		Description:       "The first pool",
		MemoryCapacity:    0,
		MemoryCommitment:  0,
		CoreCapacity:      0,
		ConnectionTimeout: 10,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Permissions:       pool.DFSAccess,
	}

	firstHost := host.Host{
		ID:     "firstHost",
		Cores:  6,
		Memory: 12000,
	}

	secondHost := host.Host{
		ID:     "secondHost",
		Cores:  8,
		Memory: 10000,
	}

	firstService := service.Service{
		ID: "firstService",
		RAMCommitment: utils.EngNotation{
			Value: uint64(1000),
		},
	}

	secondService := service.Service{
		ID: "secondService",
		RAMCommitment: utils.EngNotation{
			Value: uint64(2000),
		},
	}

	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, resourcePool.ID).
		Return([]host.Host{firstHost, secondHost}, nil)

	ft.poolStore.On("GetResourcePools", ft.ctx).
		Return([]pool.ResourcePool{resourcePool}, nil)

	ft.serviceStore.On("GetServicesByPool", ft.ctx, resourcePool.ID).
		Return([]service.Service{firstService, secondService}, nil)

	pools, err := ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p := pools[0]
	c.Assert(p.ID, Equals, resourcePool.ID)
	c.Assert(p.CoreCapacity, Equals, 14)
	c.Assert(p.MemoryCapacity, Equals, uint64(22000))
	c.Assert(p.MemoryCommitment, Equals, uint64(3000))
	c.Assert(p.ConnectionTimeout, Equals, 10)
	c.Assert(p.CreatedAt, TimeEqual, resourcePool.CreatedAt)
	c.Assert(p.UpdatedAt, TimeEqual, resourcePool.UpdatedAt)
	c.Assert(p.Permissions, Equals, resourcePool.Permissions)
}
