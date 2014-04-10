// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/serviced/domain/pool"
	. "gopkg.in/check.v1"

	"time"
)

func (ft *FacadeTest) Test_NewResourcePool(t *C) {

	pool := pool.ResourcePool{}
	err := ft.facade.AddResourcePool(ft.ctx, &pool)
	if err == nil {
		t.Errorf("Expected failure to create resource pool %-v", pool)
	}

	pool.ID = "default"
	defer ft.facade.RemoveResourcePool(ft.ctx, pool.ID)
	err = ft.facade.AddResourcePool(ft.ctx, &pool)
	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", pool, err)
		t.Fail()
	}

	err = ft.facade.AddResourcePool(ft.ctx, &pool)
	if err == nil {
		t.Errorf("Expected error creating redundant resource pool %-v", pool)
		t.Fail()
	}
}

func (ft *FacadeTest) TestDao_UpdateResourcePool(t *C) {
	defer ft.facade.RemoveResourcePool(ft.ctx, "default")

	pool := pool.New("default")
	ft.facade.AddResourcePool(ft.ctx, pool)

	pool.Priority = 1
	pool.CoreLimit = 1
	pool.MemoryLimit = 1
	err := ft.facade.UpdateResourcePool(ft.ctx, pool)
	if err != nil {
		t.Errorf("Failure updating resource pool %-v with error: %s", pool, err)
		t.Fail()
	}

	result, err := ft.facade.GetResourcePool(ft.ctx, "default")
	result.CreatedAt = pool.CreatedAt
	result.UpdatedAt = pool.UpdatedAt
	if *pool != *result {
		t.Errorf("%+v != %+v", pool, result)
		t.Fail()
	}
}

func (ft *FacadeTest) TestDao_GetResourcePool(t *C) {
	defer ft.facade.RemoveResourcePool(ft.ctx, "default")

	ft.facade.RemoveResourcePool(ft.ctx, "default")
	pool := pool.New("default")
	pool.Priority = 1
	pool.CoreLimit = 1
	pool.MemoryLimit = 1
	if err := ft.facade.AddResourcePool(ft.ctx, pool); err != nil {
		t.Fatalf("Failed to add resource pool: %v", err)
	}

	result, err := ft.facade.GetResourcePool(ft.ctx, "default")
	result.CreatedAt = pool.CreatedAt
	result.UpdatedAt = pool.UpdatedAt
	if err == nil {
		if *pool != *result {
			t.Errorf("Unexpected ResourcePool: expected=%+v, actual=%+v", pool, result)
		}
	} else {
		t.Errorf("Unexpected Error Retrieving ResourcePool: err=%s", err)
	}
}

func (ft *FacadeTest) TestDao_GetResourcePools(t *C) {
	defer ft.facade.RemoveResourcePool(ft.ctx, "default")

	pool := pool.New("default")
	pool.Priority = 1
	pool.CoreLimit = 2
	pool.MemoryLimit = 3
	ft.facade.AddResourcePool(ft.ctx, pool)
	time.Sleep(time.Second * 2)
	result, err := ft.facade.GetResourcePools(ft.ctx)
	if err == nil && len(result) == 1 {
		result[0].CreatedAt = pool.CreatedAt
		result[0].UpdatedAt = pool.UpdatedAt
		if *result[0] != *pool {
			t.Fatalf("expected [%+v] actual=%s", pool, result)
		}
	} else {
		t.Fatalf("Unexpected Error Retrieving ResourcePools: err=%s", result)
	}
}

//
//func (ft *FacadeTest)TestDaoGetPoolsIPInfo(t *C) {
//	assignIPsPool:= pool.New("assignIPsPoolID")
//	err = ft.facade.AddResourcePool(*assignIPsPool, &id)
//	if err != nil {
//		t.Errorf("Failure creating resource pool %-v with error: %s", assignIPsPool, err)
//		t.Fail()
//	}
//
//	ipAddress1 := "192.168.100.10"
//	ipAddress2 := "10.50.9.1"
//
//	assignIPsHostIPResources := []host.HostIPResource{}
//	oneHostIPResource := host.HostIPResource{}
//	oneHostIPResource.HostId = HOSTID
//	oneHostIPResource.IPAddress = ipAddress1
//	oneHostIPResource.InterfaceName = "eth0"
//	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)
//	oneHostIPResource.HostId = HOSTID
//	oneHostIPResource.IPAddress = ipAddress2
//	oneHostIPResource.InterfaceName = "eth1"
//	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)
//
//	assignIPsHost := dao.Host{}
//	assignIPsHost.Id = HOSTID
//	assignIPsHost.PoolId = assignIPsPool.Id
//	assignIPsHost.IPs = assignIPsHostIPResources
//	err = ft.facade.AddHost(assignIPsHost, &id)
//
//	var poolsIpInfo []dao.HostIPResource
//	err := ft.facade.GetPoolsIPInfo(assignIPsPool.Id, &poolsIpInfo)
//	if err != nil {
//		t.Error("GetPoolIps failed")
//	}
//	if len(poolsIpInfo) != 2 {
//		t.Error("Expected number of addresses: ", len(poolsIpInfo))
//	}
//
//	if poolsIpInfo[0].IPAddress != ipAddress1 {
//		t.Error("Unexpected IP address: ", poolsIpInfo[0].IPAddress)
//	}
//	if poolsIpInfo[1].IPAddress != ipAddress2 {
//		t.Error("Unexpected IP address: ", poolsIpInfo[1].IPAddress)
//	}
//
//	defer ft.facade.RemoveResourcePool(assignIPsPool.Id, &unused)
//	defer ft.facade.RemoveHost(assignIPsHost.Id, &unused)
//}
