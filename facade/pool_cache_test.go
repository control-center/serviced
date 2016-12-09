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
	"fmt"
	"time"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/utils"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (ft *FacadeUnitTest) Test_PoolCache(c *C) {
	fmt.Println("\033[2J-----------Starting Test_PoolCache---------")
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
	fmt.Printf("after creation: resourcePool created at: %v\n", resourcePool.CreatedAt)

	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, resourcePool.ID).
		Run(func(args mock.Arguments) { fmt.Printf("Calling mock FindHostsWithPoolID at %v\n", time.Now()) }).
		Return([]host.Host{firstHost, secondHost}, nil)

	ft.poolStore.On("GetResourcePools", ft.ctx).
		Run(func(args mock.Arguments) { fmt.Printf("Calling mock GetResourcePools at %v\n", time.Now()) }).
		Return([]pool.ResourcePool{resourcePool}, nil)

	ft.serviceStore.On("GetServicesByPool", ft.ctx, resourcePool.ID).
		Run(func(args mock.Arguments) { fmt.Printf("Calling mock GetServicesByPool at %v\n", time.Now()) }).
		Return([]service.Service{firstService, secondService}, nil)

	ft.serviceStore.On("GetServiceDetails", ft.ctx, firstService.ID).
		Run(func(args mock.Arguments) { fmt.Printf("Calling mock GetServiceDetails at %v\n", time.Now()) }).
		Return(&service.ServiceDetails{
			ID:            firstService.ID,
			RAMCommitment: firstService.RAMCommitment,
		}, nil)

	ft.serviceStore.On("Get", ft.ctx, firstService.ID).
		Run(func(args mock.Arguments) { fmt.Printf("Calling mock Get at %v\n", time.Now()) }).
		Return(&firstService, nil)

	ft.serviceStore.On("Put", ft.ctx, mock.AnythingOfType("*service.Service")).
		Run(func(args mock.Arguments) { fmt.Printf("Calling mock Put at %v\n", time.Now()) }).
		Return(nil)

	ft.configStore.On("GetConfigFiles", ft.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) { fmt.Printf("Calling mock GetConfigFiles at %v\n", time.Now()) }).
		Return([]*serviceconfigfile.SvcConfigFile{}, nil)

	ft.zzk.On("UpdateService", ft.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), false, false).
		Run(func(args mock.Arguments) { fmt.Printf("Calling mock UpdateService at %v\n", time.Now()) }).
		Return(nil)

	fmt.Printf("calling getreadpools: resourcePool created at: %v\n", resourcePool.CreatedAt)
	pools, err := ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p := pools[0]
	fmt.Printf("after getreadpools: p created at: %v\n", p.CreatedAt)
	fmt.Printf("after getreadpools: resourcePool created at: %v\n", resourcePool.CreatedAt)

	c.Assert(p.ID, Equals, resourcePool.ID)
	c.Assert(p.CoreCapacity, Equals, 14)
	c.Assert(p.MemoryCapacity, Equals, uint64(22000))
	c.Assert(p.MemoryCommitment, Equals, uint64(3000))
	c.Assert(p.ConnectionTimeout, Equals, 10)
	//c.Assert(p.CreatedAt, TimeEqual, resourcePool.CreatedAt)
	//c.Assert(p.UpdatedAt, TimeEqual, resourcePool.UpdatedAt)
	c.Assert(p.Permissions, Equals, resourcePool.Permissions)

	firstService.RAMCommitment = utils.EngNotation{
		Value: uint64(2000),
	}
	ft.Facade.UpdateService(ft.ctx, firstService)

	pools, err = ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p = pools[0]
	c.Assert(p.ID, Equals, resourcePool.ID)
	c.Assert(p.CoreCapacity, Equals, 14)
	c.Assert(p.MemoryCapacity, Equals, uint64(22000))
	c.Assert(p.MemoryCommitment, Equals, uint64(4000))
	c.Assert(p.ConnectionTimeout, Equals, 10)
	//c.Assert(p.CreatedAt, TimeEqual, resourcePool.CreatedAt)
	//c.Assert(p.UpdatedAt, TimeEqual, resourcePool.UpdatedAt)
	c.Assert(p.Permissions, Equals, resourcePool.Permissions)
}
