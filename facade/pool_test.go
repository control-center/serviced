// Copyright 2014 The Serviced Authors.
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

// +build integration

package facade

import (
	"fmt"
	"time"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	. "gopkg.in/check.v1"
)

func (ft *FacadeIntegrationTest) Test_NewResourcePool(t *C) {
	fmt.Println(" ##### Test_NewResourcePool: starting")
	poolID := "Test_NewResourcePool"
	result, err := ft.Facade.GetResourcePool(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(result, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, poolID)

	rp := pool.ResourcePool{}
	err = ft.Facade.AddResourcePool(ft.CTX, &rp)
	if err == nil {
		t.Errorf("Expected failure to create resource pool %-v", rp)
	}

	rp.ID = poolID
	err = ft.Facade.AddResourcePool(ft.CTX, &rp)
	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", rp, err)
		t.Fail()
	}

	err = ft.Facade.AddResourcePool(ft.CTX, &rp)
	if err == nil {
		t.Errorf("Expected error creating redundant resource pool %-v", rp)
		t.Fail()
	}
	fmt.Println(" ##### Test_NewResourcePool: PASSED")
}

func (ft *FacadeIntegrationTest) Test_UpdateResourcePool(t *C) {
	fmt.Println(" ##### Test_UpdateResourcePool: starting")
	poolID := "Test_UpdateResourcePool"
	result, err := ft.Facade.GetResourcePool(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(result, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, poolID)

	myPool := pool.New(poolID)
	ft.Facade.AddResourcePool(ft.CTX, myPool)

	myPool.CoreLimit = 1
	myPool.MemoryLimit = 1
	err = ft.Facade.UpdateResourcePool(ft.CTX, myPool)
	if err != nil {
		t.Errorf("Failure updating resource pool %-v with error: %s", myPool, err)
		t.Fail()
	}

	result, err = ft.Facade.GetResourcePool(ft.CTX, poolID)
	result.CreatedAt = myPool.CreatedAt
	result.UpdatedAt = myPool.UpdatedAt

	if !myPool.Equals(result) {
		t.Errorf("%+v != %+v", myPool, result)
		t.Fail()
	}
	fmt.Println(" ##### Test_UpdateResourcePool: PASSED")
}

func (ft *FacadeIntegrationTest) Test_GetResourcePool(t *C) {
	fmt.Println(" ##### Test_GetResourcePool: starting")
	poolID := "Test_GetResourcePool"
	result, err := ft.Facade.GetResourcePool(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(result, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, poolID)

	ft.Facade.RemoveResourcePool(ft.CTX, poolID)
	rp := pool.New(poolID)
	rp.CoreLimit = 1
	rp.MemoryLimit = 1
	if err := ft.Facade.AddResourcePool(ft.CTX, rp); err != nil {
		t.Fatalf("Failed to add resource pool: %v", err)
	}

	result, err = ft.Facade.GetResourcePool(ft.CTX, poolID)
	result.CreatedAt = rp.CreatedAt
	result.UpdatedAt = rp.UpdatedAt
	if err == nil {
		if !rp.Equals(result) {
			t.Errorf("Unexpected ResourcePool: expected=%+v, actual=%+v", rp, result)
		}
	} else {
		t.Errorf("Unexpected Error Retrieving ResourcePool: %v", err)
	}
	fmt.Println(" ##### Test_GetResourcePool: PASSED")
}

func (ft *FacadeIntegrationTest) Test_RemoveResourcePool(t *C) {
	fmt.Println(" ##### Test_RemoveResourcePool: starting")
	poolID := "Test_RemoveResourcePool"

	result, err := ft.Facade.GetResourcePool(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(result, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, poolID)

	err = ft.Facade.RemoveResourcePool(ft.CTX, poolID)
	t.Assert(err, IsNil)

	rp := pool.New(poolID)
	err = ft.Facade.AddResourcePool(ft.CTX, rp)
	t.Assert(err, IsNil)

	rp, err = ft.Facade.GetResourcePool(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(rp.ID, Equals, poolID)

	err = ft.Facade.RemoveResourcePool(ft.CTX, poolID)
	t.Assert(err, IsNil)
	rp, err = ft.Facade.GetResourcePool(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(rp, IsNil)
	fmt.Println(" ##### Test_RemoveResourcePool: PASSED")
}

func (ft *FacadeIntegrationTest) TestRestoreResourcePools(c *C) {
	pools1 := []pool.ResourcePool{
		{
			ID:    "testpool-1",
			Realm: "default",
			VirtualIPs: []pool.VirtualIP{
				{
					PoolID:        "testpool-1",
					IP:            "122.34.56.7",
					Netmask:       "255.255.255.0",
					BindInterface: "eth0",
				},
			},
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
		},
	}
	defer ft.Facade.RemoveResourcePool(ft.CTX, "testpool-1")
	err := ft.Facade.RestoreResourcePools(ft.CTX, pools1)
	c.Assert(err, IsNil)
	actual, err := ft.Facade.GetResourcePools(ft.CTX)
	c.Assert(err, IsNil)
	for i := range actual {
		actual[i].DatabaseVersion = 0
		actual[i].CreatedAt = time.Time{}
		actual[i].UpdatedAt = time.Time{}
	}
	c.Assert(actual, DeepEquals, pools1)

	pools2 := []pool.ResourcePool{
		{
			ID:    "testpool-1",
			Realm: "default",
			VirtualIPs: []pool.VirtualIP{
				{
					PoolID:        "testpool-1",
					IP:            "122.34.56.8",
					Netmask:       "255.255.255.1",
					BindInterface: "eth1",
				},
			},
			CoreLimit:   1,
			MemoryLimit: 1,
			CreatedAt:   time.Time{},
			UpdatedAt:   time.Time{},
		}, {
			ID:    "testpool-2",
			Realm: "default",
			VirtualIPs: []pool.VirtualIP{
				{
					PoolID:        "testpool-2",
					IP:            "122.34.56.7",
					Netmask:       "255.255.255.0",
					BindInterface: "eth0",
				},
			},
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
		},
	}
	defer ft.Facade.RemoveResourcePool(ft.CTX, "testpool-2")
	err = ft.Facade.RestoreResourcePools(ft.CTX, pools2)
	c.Assert(err, IsNil)
	actual, err = ft.Facade.GetResourcePools(ft.CTX)
	c.Assert(err, IsNil)
	for i := range actual {
		actual[i].DatabaseVersion = 0
		actual[i].CreatedAt = time.Time{}
		actual[i].UpdatedAt = time.Time{}
	}
	c.Assert(actual, DeepEquals, pools2)
}

func (ft *FacadeIntegrationTest) Test_GetResourcePools(t *C) {
	result, err := ft.Facade.GetResourcePools(ft.CTX)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		pools := make([]pool.ResourcePool, len(result))
		for i, pool := range result {
			pools[i] = pool
		}

		t.Fatalf("unexpected pool found: %v", pools)
	}

	poolID := "Test_GetResourcePools"
	defer ft.Facade.RemoveResourcePool(ft.CTX, poolID)
	rp := pool.New(poolID)
	rp.CoreLimit = 2
	rp.MemoryLimit = 3
	ft.Facade.AddResourcePool(ft.CTX, rp)
	time.Sleep(time.Second * 2)
	result, err = ft.Facade.GetResourcePools(ft.CTX)
	if err == nil && len(result) == 1 {
		result[0].CreatedAt = rp.CreatedAt
		result[0].UpdatedAt = rp.UpdatedAt
		if !result[0].Equals(rp) {
			t.Fatalf("expected [%+v] actual=[%+v]", rp, result)
		}
	} else {
		t.Fatalf("Unexpected Error Retrieving ResourcePools: %v", err)
	}
}

func (ft *FacadeIntegrationTest) Test_GetPoolsIPs(t *C) {
	assignIPsPool := pool.New("Test_GetPoolsIPs")
	err := ft.Facade.AddResourcePool(ft.CTX, assignIPsPool)
	defer func() {
		ft.Facade.RemoveResourcePool(ft.CTX, assignIPsPool.ID)
	}()

	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", assignIPsPool, err)
		t.Fail()
	}

	hostID := "deadb21f"
	ipAddress1 := "192.168.100.10"
	ipAddress2 := "10.50.9.1"

	assignIPsHostIPResources := []host.HostIPResource{}
	oneHostIPResource := host.HostIPResource{}
	oneHostIPResource.HostID = hostID
	oneHostIPResource.IPAddress = ipAddress1
	oneHostIPResource.InterfaceName = "eth0"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)
	oneHostIPResource.HostID = "A"
	oneHostIPResource.IPAddress = ipAddress2
	oneHostIPResource.InterfaceName = "eth1"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)

	assignIPsHost, err := host.Build("", "65535", assignIPsPool.ID, "", []string{}...)
	if err != nil {
		t.Fatalf("could not build host for test: %v", err)
	}
	assignIPsHost.ID = hostID
	assignIPsHost.PoolID = assignIPsPool.ID
	assignIPsHost.IPs = assignIPsHostIPResources
	err = ft.Facade.AddHost(ft.CTX, assignIPsHost)
	if err != nil {
		t.Fatalf("failed to add host: %v", err)
	}
	defer func() {
		ft.Facade.RemoveHost(ft.CTX, assignIPsHost.ID)
	}()
	time.Sleep(2 * time.Second)
	IPs, err := ft.Facade.GetPoolIPs(ft.CTX, assignIPsPool.ID)
	if err != nil {
		t.Error("GetPoolIps failed")
	}
	if len(IPs.HostIPs) != 2 {
		t.Fatalf("Expected 2 addresses, found %v", len(IPs.HostIPs))
	}

	if IPs.HostIPs[0].IPAddress != ipAddress1 {
		t.Errorf("Unexpected IP address: %v", IPs.HostIPs[0].IPAddress)
	}
	if IPs.HostIPs[1].IPAddress != ipAddress2 {
		t.Errorf("Unexpected IP address: %v", IPs.HostIPs[1].IPAddress)
	}

}

func (ft *FacadeIntegrationTest) Test_VirtualIPs(t *C) {
	fmt.Println(" ##### Test_VirtualIPs")
	myPoolID := "Test_VirtualIPs"
	assignIPsPool := pool.New(myPoolID)
	err := ft.Facade.AddResourcePool(ft.CTX, assignIPsPool)
	defer func() {
		ft.Facade.RemoveResourcePool(ft.CTX, assignIPsPool.ID)
	}()

	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", assignIPsPool, err)
		t.Fail()
	}

	hostID := "deadb22f"
	ipAddress1 := "192.168.100.10"

	assignIPsHostIPResources := []host.HostIPResource{}
	oneHostIPResource := host.HostIPResource{}
	oneHostIPResource.HostID = hostID
	oneHostIPResource.IPAddress = ipAddress1
	myInterfaceName := "eth0"
	oneHostIPResource.InterfaceName = myInterfaceName
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)

	assignIPsHost, err := host.Build("", "65535", assignIPsPool.ID, "", []string{}...)
	if err != nil {
		t.Fatalf("could not build host for test: %v", err)
	}
	assignIPsHost.ID = hostID
	assignIPsHost.PoolID = assignIPsPool.ID
	assignIPsHost.IPs = assignIPsHostIPResources
	err = ft.Facade.AddHost(ft.CTX, assignIPsHost)
	if err != nil {
		t.Fatalf("failed to add host: %v", err)
	}
	defer func() {
		ft.Facade.RemoveHost(ft.CTX, assignIPsHost.ID)
	}()
	time.Sleep(2 * time.Second)
	someIPAddresses := []string{"192.168.100.20", "192.168.100.30", "192.168.100.40", "192.168.100.50"}
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: someIPAddresses[0], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err != nil {
		t.Errorf("AddVirtualIP failed: %v", err)
	}
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: someIPAddresses[1], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err != nil {
		t.Errorf("AddVirtualIP failed: %v", err)
	}
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: someIPAddresses[2], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err != nil {
		t.Errorf("AddVirtualIP failed: %v", err)
	}
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: someIPAddresses[3], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err != nil {
		t.Errorf("AddVirtualIP failed: %v", err)
	}
	IPs, err := ft.Facade.GetPoolIPs(ft.CTX, assignIPsPool.ID)
	if err != nil {
		t.Errorf("GetPoolIps failed: %v", err)
	}
	if len(IPs.VirtualIPs) != 4 {
		t.Fatalf("Expected 4 addresses, found %v", len(IPs.VirtualIPs))
	}

	for _, vip := range IPs.VirtualIPs {
		found := false
		for _, anIPAddress := range someIPAddresses {
			if anIPAddress == vip.IP {
				fmt.Println(" ##### Found: ", vip.IP)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Did not find %v in the model...", vip.IP)
		}
	}

	if err := ft.Facade.RemoveVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: someIPAddresses[0], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err != nil {
		t.Errorf("RemoveVirtualIP failed: %v", err)
	}
	if err := ft.Facade.RemoveVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: someIPAddresses[1], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err != nil {
		t.Errorf("RemoveVirtualIP failed: %v", err)
	}
	if err := ft.Facade.RemoveVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: someIPAddresses[3], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err != nil {
		t.Errorf("RemoveVirtualIP failed: %v", err)
	}
	IPs, err = ft.Facade.GetPoolIPs(ft.CTX, assignIPsPool.ID)
	if err != nil {
		t.Errorf("GetPoolIps failed: %v", err)
	}
	fmt.Println(" ##### IPs.VirtualIPs: ", IPs.VirtualIPs)
	if len(IPs.VirtualIPs) != 1 {
		t.Fatalf("Expected 1 address, found %v", len(IPs.VirtualIPs))
	}

	if IPs.VirtualIPs[0].IP != someIPAddresses[2] {
		t.Fatalf("Expected %v but found %v", someIPAddresses[2], IPs.VirtualIPs[0].IP)
	}

	if err := ft.Facade.RemoveVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: someIPAddresses[2], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err != nil {
		t.Errorf("RemoveVirtualIP failed: %v", err)
	}
}

func (ft *FacadeIntegrationTest) Test_InvalidVirtualIPs(t *C) {
	fmt.Println(" ##### Test_InvalidVirtualIPs")
	myPoolID := "Test_InvalidVirtualIPs"
	assignIPsPool := pool.New(myPoolID)
	err := ft.Facade.AddResourcePool(ft.CTX, assignIPsPool)
	defer func() {
		ft.Facade.RemoveResourcePool(ft.CTX, assignIPsPool.ID)
	}()

	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", assignIPsPool, err)
		t.Fail()
	}

	hostID := "deadb22f"
	ipAddress1 := "192.168.100.10"

	assignIPsHostIPResources := []host.HostIPResource{}
	oneHostIPResource := host.HostIPResource{}
	oneHostIPResource.HostID = hostID
	oneHostIPResource.IPAddress = ipAddress1
	myInterfaceName := "eth0"
	oneHostIPResource.InterfaceName = myInterfaceName
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)

	assignIPsHost, err := host.Build("", "65535", assignIPsPool.ID, "", []string{}...)
	if err != nil {
		t.Fatalf("could not build host for test: %v", err)
	}
	assignIPsHost.ID = hostID
	assignIPsHost.PoolID = assignIPsPool.ID
	assignIPsHost.IPs = assignIPsHostIPResources
	err = ft.Facade.AddHost(ft.CTX, assignIPsHost)
	if err != nil {
		t.Fatalf("failed to add host: %v", err)
	}
	defer func() {
		ft.Facade.RemoveHost(ft.CTX, assignIPsHost.ID)
	}()
	time.Sleep(2 * time.Second)

	invalidIPAddresses := []string{"192.F.100.20", "192.168.100.3*", "192.168.100", "192..168.100.50"}
	// try adding invalid IPs
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: invalidIPAddresses[0], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("AddVirtualIP should have failed on: %v", invalidIPAddresses[0])
	}
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: invalidIPAddresses[1], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("AddVirtualIP should have failed on: %v", invalidIPAddresses[1])
	}
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: invalidIPAddresses[2], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("AddVirtualIP should have failed on: %v", invalidIPAddresses[2])
	}
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: invalidIPAddresses[3], Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("AddVirtualIP should have failed on: %v", invalidIPAddresses[3])
	}

	validIPAddress := "192.168.100.20"
	invalidPoolID := "invalidPoolID"
	// try adding a with an invalid poolID
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: invalidPoolID, IP: validIPAddress, Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("AddVirtualIP should have failed on invalid pool ID: %v", invalidPoolID)
	}

	// add an already present static IP
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: ipAddress1, Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("Added an IP that was already (%v) there... should have failed.", validIPAddress)
	}

	// add a virtual IP
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: validIPAddress, Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err != nil {
		t.Errorf("AddVirtualIP failed: %v", err)
	}

	// try to add an already added virtual IP
	if err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: validIPAddress, Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("Added an IP that was already (%v) there... should have failed.", validIPAddress)
	}

	notAddedIPAddress := "192.168.100.30"
	// try removing a virtual IP that has not been added
	if err := ft.Facade.RemoveVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: notAddedIPAddress, Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("Tried to remove a virtual IP that was NOT in the pool: %v", notAddedIPAddress)
	}

	// try removing a static IP
	if err := ft.Facade.RemoveVirtualIP(ft.CTX, pool.VirtualIP{PoolID: myPoolID, IP: ipAddress1, Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("Tried to remove a virtual IP that was NOT in the pool: %v", notAddedIPAddress)
	}

	// try removing with an invalid pool ID
	if err := ft.Facade.RemoveVirtualIP(ft.CTX, pool.VirtualIP{PoolID: invalidPoolID, IP: validIPAddress, Netmask: "255.255.255.0", BindInterface: myInterfaceName}); err == nil {
		t.Errorf("Invalid Pool ID (%v) should have failed.", invalidPoolID)
	}
}

func (ft *FacadeIntegrationTest) Test_PoolCapacity(t *C) {
	hostid := "deadb23f"
	poolid := "pool-id"

	//create pool for test
	rp := pool.New(poolid)
	if err := ft.Facade.AddResourcePool(ft.CTX, rp); err != nil {
		t.Fatalf("Could not add pool for test: %v", err)
	}

	//fill host with required values
	h, err := host.Build("", "65535", poolid, "", []string{}...)
	h.ID = hostid
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}

	err = ft.Facade.AddHost(ft.CTX, h)
	if err != nil {
		t.Errorf("Unexpected error adding host: %v", err)
	}

	// load pool with calculated capacity
	loadedPool, err := ft.Facade.GetResourcePool(ft.CTX, poolid)

	if err != nil {
		t.Fatalf("Unexpected error calculating pool capacity: %v", err)
	}

	if loadedPool.CoreCapacity <= 0 || loadedPool.MemoryCapacity <= 0 {
		t.Fatalf("Unexpected values calculated for %s capacity: CPU - %v : Memory - %v", loadedPool.ID, loadedPool.CoreCapacity, loadedPool.MemoryCapacity)
	}
}

func (ft *FacadeIntegrationTest) Test_PoolCommitment(t *C) {
	poolid := "pool-id"

	//create pool for test
	rp := pool.New(poolid)
	if err := ft.Facade.AddResourcePool(ft.CTX, rp); err != nil {
		t.Fatalf("Could not add pool for test: %v", err)
	}

	// load pool with calculated capacity
	loadedPool, err := ft.Facade.GetResourcePool(ft.CTX, poolid)

	commitmentErr := ft.Facade.calcPoolCommitment(ft.CTX, loadedPool)

	if commitmentErr != nil {
		t.Fatalf("Unexpected error calculating pool commitment: %v", err)
	}
}
