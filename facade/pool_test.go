// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade


//func TestDao_NewResourcePool(t *testing.T) {
//	controlPlaneDao.RemoveResourcePool("default", &unused)
//	pool := dao.ResourcePool{}
//	err := controlPlaneDao.AddResourcePool(pool, &id)
//	if err == nil {
//		t.Errorf("Expected failure to create resource pool %-v", pool)
//		t.Fail()
//	}
//
//	pool.Id = "default"
//	err = controlPlaneDao.AddResourcePool(pool, &id)
//	if err != nil {
//		t.Errorf("Failure creating resource pool %-v with error: %s", pool, err)
//		t.Fail()
//	}
//
//	err = controlPlaneDao.AddResourcePool(pool, &id)
//	if err == nil {
//		t.Errorf("Expected error creating redundant resource pool %-v", pool)
//		t.Fail()
//	}
//}
//func TestDao_UpdateResourcePool(t *testing.T) {
//	controlPlaneDao.RemoveResourcePool("default", &unused)
//
//	pool, _ := dao.NewResourcePool("default")
//	controlPlaneDao.AddResourcePool(*pool, &id)
//
//	pool.Priority = 1
//	pool.CoreLimit = 1
//	pool.MemoryLimit = 1
//	err := controlPlaneDao.UpdateResourcePool(*pool, &unused)
//
//	if err != nil {
//		t.Errorf("Failure updating resource pool %-v with error: %s", pool, err)
//		t.Fail()
//	}
//
//	result := dao.ResourcePool{}
//	controlPlaneDao.GetResourcePool("default", &result)
//	result.CreatedAt = pool.CreatedAt
//	result.UpdatedAt = pool.UpdatedAt
//	if *pool != result {
//		t.Errorf("%+v != %+v", *pool, result)
//		t.Fail()
//	}
//}
//
//func TestDao_GetResourcePool(t *testing.T) {
//	controlPlaneDao.RemoveResourcePool("default", &unused)
//	pool, _ := dao.NewResourcePool("default")
//	pool.Priority = 1
//	pool.CoreLimit = 1
//	pool.MemoryLimit = 1
//	controlPlaneDao.AddResourcePool(*pool, &id)
//
//	result := dao.ResourcePool{}
//	err := controlPlaneDao.GetResourcePool("default", &result)
//	result.CreatedAt = pool.CreatedAt
//	result.UpdatedAt = pool.UpdatedAt
//	if err == nil {
//		if *pool != result {
//			t.Errorf("Unexpected ResourcePool: expected=%+v, actual=%+v", pool, result)
//		}
//	} else {
//		t.Errorf("Unexpected Error Retrieving ResourcePool: err=%s", err)
//	}
//}
//
//func TestDao_GetResourcePools(t *testing.T) {
//	controlPlaneDao.RemoveResourcePool("default", &unused)
//
//	pool, _ := dao.NewResourcePool("default")
//	pool.Priority = 1
//	pool.CoreLimit = 2
//	pool.MemoryLimit = 3
//	controlPlaneDao.AddResourcePool(*pool, &id)
//
//	var result map[string]*dao.ResourcePool
//	err := controlPlaneDao.GetResourcePools(new(dao.EntityRequest), &result)
//	if err == nil && len(result) == 1 {
//		result["default"].CreatedAt = pool.CreatedAt
//		result["default"].UpdatedAt = pool.UpdatedAt
//		if *result["default"] != *pool {
//			t.Errorf("expected [%+v] actual=%s", *pool, result)
//			t.Fail()
//		}
//	} else {
//		t.Errorf("Unexpected Error Retrieving ResourcePools: err=%s", result)
//		t.Fail()
//	}
//}


//func TestDaoGetPoolsIPInfo(t *testing.T) {
//	assignIPsPool, _ := dao.NewResourcePool("assignIPsPoolID")
//	err = controlPlaneDao.AddResourcePool(*assignIPsPool, &id)
//	if err != nil {
//		t.Errorf("Failure creating resource pool %-v with error: %s", assignIPsPool, err)
//		t.Fail()
//	}
//
//	ipAddress1 := "192.168.100.10"
//	ipAddress2 := "10.50.9.1"
//
//	assignIPsHostIPResources := []dao.HostIPResource{}
//	oneHostIPResource := dao.HostIPResource{}
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
//	err = controlPlaneDao.AddHost(assignIPsHost, &id)
//
//	var poolsIpInfo []dao.HostIPResource
//	err := controlPlaneDao.GetPoolsIPInfo(assignIPsPool.Id, &poolsIpInfo)
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
//	defer controlPlaneDao.RemoveResourcePool(assignIPsPool.Id, &unused)
//	defer controlPlaneDao.RemoveHost(assignIPsHost.Id, &unused)
//}
