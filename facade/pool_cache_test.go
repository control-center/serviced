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
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/utils"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

type poolCacheEnv struct {
	resourcePool  pool.ResourcePool
	firstHost     host.Host
	secondHost    host.Host
	firstService  service.Service
	secondService service.Service
}

func newPoolCacheEnv() *poolCacheEnv {
	timeStamp := time.Now()
	return &poolCacheEnv{
		resourcePool: pool.ResourcePool{
			ID:                "firstPool",
			Description:       "The first pool",
			MemoryCapacity:    0,
			MemoryCommitment:  0,
			CoreCapacity:      0,
			ConnectionTimeout: 10,
			CreatedAt:         timeStamp,
			UpdatedAt:         timeStamp,
			Permissions:       pool.DFSAccess,
		},
		firstHost: host.Host{
			ID:     "firstHost",
			PoolID: "firstPool",
			Cores:  6,
			Memory: 12000,
		},
		secondHost: host.Host{
			ID:     "secondHost",
			PoolID: "firstPool",
			Cores:  8,
			Memory: 10000,
		},
		firstService: service.Service{
			ID: "firstService",
			RAMCommitment: utils.EngNotation{
				Value: uint64(1000),
			},
		},
		secondService: service.Service{
			ID: "secondService",
			RAMCommitment: utils.EngNotation{
				Value: uint64(2000),
			},
		},
	}
}

// Test_PoolCacheEditPool tests that the cache is invalidated when a
// Pool's permissions change, and is subsequently updated on the next get
func (ft *FacadeUnitTest) Test_PoolCacheEditPool(c *C) {
	ft.setupMockDFSLocking()

	pc := newPoolCacheEnv()

	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, pc.resourcePool.ID).
		Return([]host.Host{pc.firstHost, pc.secondHost}, nil)

	ft.poolStore.On("GetResourcePools", ft.ctx).
		Return([]pool.ResourcePool{pc.resourcePool}, nil).Once()

	ft.serviceStore.On("GetServicesByPool", ft.ctx, pc.resourcePool.ID).
		Return([]service.Service{pc.firstService, pc.secondService}, nil)

	ft.serviceStore.On("GetServiceDetails", ft.ctx, pc.firstService.ID).
		Return(&service.ServiceDetails{
			ID:            pc.firstService.ID,
			RAMCommitment: pc.firstService.RAMCommitment,
		}, nil)

	pools, err := ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p := pools[0]

	c.Assert(p.ID, Equals, pc.resourcePool.ID)
	c.Assert(p.CreatedAt, TimeEqual, pc.resourcePool.CreatedAt)
	c.Assert(p.UpdatedAt, TimeEqual, pc.resourcePool.UpdatedAt)
	c.Assert(p.Permissions, Equals, pc.resourcePool.Permissions)

	pc.resourcePool.Permissions = pool.AdminAccess & pool.DFSAccess

	ft.poolStore.On("GetResourcePools", ft.ctx).
		Return([]pool.ResourcePool{pc.resourcePool}, nil).Once()

	ft.poolStore.On("Get", ft.ctx, pool.Key(pc.resourcePool.ID), mock.AnythingOfType("*pool.ResourcePool")).
		Return(nil).
		Run(func(args mock.Arguments) {
			*args.Get(2).(*pool.ResourcePool) = pc.resourcePool
		})

	ft.poolStore.On("Put", ft.ctx, pool.Key(pc.resourcePool.ID), mock.AnythingOfType("*pool.ResourcePool")).
		Return(nil)

	ft.zzk.On("UpdateResourcePool", mock.AnythingOfType("*pool.ResourcePool")).
		Return(nil)

	ft.Facade.UpdateResourcePool(ft.ctx, &pc.resourcePool)

	// GetReadPools should see that the cache is dirty, and update itself
	pools, err = ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p = pools[0]
	c.Assert(p.ID, Equals, pc.resourcePool.ID)
	c.Assert(p.CreatedAt, TimeEqual, pc.resourcePool.CreatedAt)
	c.Assert(p.UpdatedAt, Not(TimeEqual), pc.resourcePool.UpdatedAt)
	c.Assert(p.Permissions, Equals, (pool.AdminAccess & pool.DFSAccess))
}

// Test_PoolCacheEditService tests that the cache is invalidated when a
// service's ram commitment changes, and is subsequently updated on the next get
func (ft *FacadeUnitTest) Test_PoolCacheEditService(c *C) {
	ft.setupMockDFSLocking()

	pc := newPoolCacheEnv()

	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, pc.resourcePool.ID).
		Return([]host.Host{pc.firstHost, pc.secondHost}, nil)

	ft.poolStore.On("GetResourcePools", ft.ctx).
		Return([]pool.ResourcePool{pc.resourcePool}, nil)

	ft.serviceStore.On("GetServicesByPool", ft.ctx, pc.resourcePool.ID).
		Return([]service.Service{pc.firstService, pc.secondService}, nil).Once()

	ft.serviceStore.On("GetServiceDetails", ft.ctx, pc.firstService.ID).
		Return(&service.ServiceDetails{
			ID:            pc.firstService.ID,
			RAMCommitment: pc.firstService.RAMCommitment,
		}, nil)

	ft.serviceStore.On("Get", ft.ctx, pc.firstService.ID).
		Return(&pc.firstService, nil)

	ft.serviceStore.On("Put", ft.ctx, mock.AnythingOfType("*service.Service")).
		Return(nil)

	ft.serviceStore.On("GetServiceDetailsByParentID", ft.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).
		Return([]service.ServiceDetails{}, nil)

	ft.configfileStore.On("GetConfigFiles", ft.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return([]*serviceconfigfile.SvcConfigFile{}, nil)

	emptyMap := []*servicetemplate.ServiceTemplate{}
	ft.templateStore.On("GetServiceTemplates", ft.ctx).Return(emptyMap, nil)

	ft.zzk.On("UpdateService", ft.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), false, false).
		Return(nil)

	pools, err := ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p := pools[0]

	c.Assert(p.ID, Equals, pc.resourcePool.ID)
	c.Assert(p.MemoryCommitment, Equals, uint64(3000))

	pc.firstService.RAMCommitment = utils.EngNotation{
		Value: uint64(2000),
	}

	err = ft.Facade.UpdateService(ft.ctx, pc.firstService)
	c.Assert(err, IsNil)

	// Make sure that we return the new secondService with the updated RAMCommitment
	ft.serviceStore.On("GetServicesByPool", ft.ctx, pc.resourcePool.ID).
		Return([]service.Service{pc.firstService, pc.secondService}, nil).Once()

	// GetReadPools should see that the cache is dirty, and update itself
	pools, err = ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p = pools[0]
	c.Assert(p.ID, Equals, pc.resourcePool.ID)
	c.Assert(p.MemoryCommitment, Equals, uint64(4000))
}

// Test_PoolCacheRemoveHost tests that cache is invalidated when a host is
// removed from a pool, and is subsequently updated on the next get
func (ft *FacadeUnitTest) Test_PoolCacheRemoveHost(c *C) {
	ft.setupMockDFSLocking()

	pc := newPoolCacheEnv()

	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, pc.resourcePool.ID).
		Return([]host.Host{pc.firstHost, pc.secondHost}, nil).Once()

	ft.poolStore.On("GetResourcePools", ft.ctx).
		Return([]pool.ResourcePool{pc.resourcePool}, nil)

	ft.serviceStore.On("GetServicesByPool", ft.ctx, pc.resourcePool.ID).
		Return([]service.Service{pc.firstService, pc.secondService}, nil)

	ft.serviceStore.On("GetServiceDetails", ft.ctx, pc.firstService.ID).
		Return(&service.ServiceDetails{
			ID:            pc.firstService.ID,
			RAMCommitment: pc.firstService.RAMCommitment,
		}, nil)

	ft.hostStore.On("Get", ft.ctx, host.Key(pc.secondHost.ID), mock.AnythingOfType("*host.Host")).
		Return(nil).
		Run(func(args mock.Arguments) {
			*args.Get(2).(*host.Host) = pc.secondHost
		})

	ft.zzk.On("RemoveHost", &pc.secondHost).Return(nil)
	ft.zzk.On("UnregisterDfsClients", []host.Host{pc.secondHost}).Return(nil)

	ft.hostkeyStore.On("Delete", ft.ctx, pc.secondHost.ID).Return(nil)
	ft.hostStore.On("Delete", ft.ctx, host.Key(pc.secondHost.ID)).Return(nil)

	pools, err := ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p := pools[0]

	c.Assert(p.ID, Equals, pc.resourcePool.ID)
	c.Assert(p.CoreCapacity, Equals, 14)
	c.Assert(p.MemoryCapacity, Equals, uint64(22000))

	err = ft.Facade.RemoveHost(ft.ctx, pc.secondHost.ID)
	c.Assert(err, IsNil)

	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, pc.resourcePool.ID).
		Return([]host.Host{pc.firstHost}, nil).Once()

	pools, err = ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p = pools[0]
	c.Assert(p.ID, Equals, pc.resourcePool.ID)
	c.Assert(p.CoreCapacity, Equals, 6)
	c.Assert(p.MemoryCapacity, Equals, uint64(12000))
}

// Test_PoolCacheAddHost tests that cache is invalidated when a host is
// added to a pool, and is subsequently updated on the next get
func (ft *FacadeUnitTest) Test_PoolCacheAddHost(c *C) {
	ft.setupMockDFSLocking()

	pc := newPoolCacheEnv()

	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, pc.resourcePool.ID).
		Return([]host.Host{pc.firstHost}, nil).Once()

	ft.poolStore.On("GetResourcePools", ft.ctx).
		Return([]pool.ResourcePool{pc.resourcePool}, nil)

	ft.serviceStore.On("GetServicesByPool", ft.ctx, pc.resourcePool.ID).
		Return([]service.Service{pc.firstService, pc.secondService}, nil)

	ft.serviceStore.On("GetServiceDetails", ft.ctx, pc.firstService.ID).
		Return(&service.ServiceDetails{
			ID:            pc.firstService.ID,
			RAMCommitment: pc.firstService.RAMCommitment,
		}, nil)

	ft.hostStore.On("Get", ft.ctx, host.Key(pc.secondHost.ID), mock.AnythingOfType("*host.Host")).
		Return(datastore.ErrNoSuchEntity{}).
		Once()

	pools, err := ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p := pools[0]

	c.Assert(p.ID, Equals, pc.resourcePool.ID)
	c.Assert(p.CoreCapacity, Equals, 6)
	c.Assert(p.MemoryCapacity, Equals, uint64(12000))

	ft.poolStore.On("Get", ft.ctx, pool.Key(pc.resourcePool.ID), mock.AnythingOfType("*pool.ResourcePool")).
		Return(nil).
		Run(func(args mock.Arguments) {
			*args.Get(2).(*pool.ResourcePool) = pc.resourcePool
		})

	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, pc.resourcePool.ID).
		Return([]host.Host{pc.firstHost, pc.secondHost}, nil)

	ft.hostkeyStore.On("Put", ft.ctx, pc.secondHost.ID, mock.AnythingOfType("*hostkey.RSAKey")).
		Return(nil)

	ft.hostStore.On("Put", ft.ctx, host.Key(pc.secondHost.ID), &pc.secondHost).
		Return(nil)

	ft.zzk.On("AddHost", &pc.secondHost).Return(nil)

	_, err = ft.Facade.AddHost(ft.ctx, &pc.secondHost)
	c.Assert(err, IsNil)

	pools, err = ft.Facade.GetReadPools(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, Not(IsNil))
	c.Assert(len(pools), Equals, 1)

	p = pools[0]
	c.Assert(p.ID, Equals, pc.resourcePool.ID)
	c.Assert(p.CoreCapacity, Equals, 14)
	c.Assert(p.MemoryCapacity, Equals, uint64(22000))
}
