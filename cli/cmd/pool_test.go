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

// +build unit

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/utils"
)

const (
	NilPool = "NilPool"
)

var DefaultTestPools = []pool.ResourcePool{
	{
		ID:          "test-pool-id-1",
		CoreLimit:   8,
		MemoryLimit: 0,
	}, {
		ID:          "test-pool-id-2",
		CoreLimit:   4,
		MemoryLimit: 4 * 1024 * 1024 * 1024,
	}, {
		ID:          "test-pool-id-3",
		CoreLimit:   2,
		MemoryLimit: 512 * 1024 * 1024,
	},
}

var DefaultTestHostIPs = []host.HostIPResource{
	{
		HostID:        "test-host-id-1",
		IPAddress:     "127.0.0.1",
		InterfaceName: "test-interface-name-1",
	}, {
		HostID:        "test-host-id-2",
		IPAddress:     "192.168.0.1",
		InterfaceName: "test-interface-name-2",
	}, {
		HostID:        "test-host-id-3",
		IPAddress:     "0.0.0.0",
		InterfaceName: "test-interface-name-3",
	},
}

var (
	ErrNoPoolFound = errors.New("no pool found")
	ErrInvalidPool = errors.New("invalid pool")
)

type PoolAPITest struct {
	api.API
	fail    bool
	pools   *[]pool.ResourcePool
	hostIPs []host.HostIPResource
}

func EmptyPoolAPI() PoolAPITest {
	return PoolAPITest{
		pools: &[]pool.ResourcePool{},
	}
}

func DefaultPoolAPI() PoolAPITest {
	test := PoolAPITest{
		pools:   &[]pool.ResourcePool{},
		hostIPs: DefaultTestHostIPs,
	}
	*test.pools = append(*test.pools, DefaultTestPools[:]...)
	return test
}

func RunCmd(test api.API, args ...string) {
	c := New(test, utils.TestConfigReader(make(map[string]string)), MockLogControl{})
	c.exitDisabled = true
	c.Run(args)
}

func (t PoolAPITest) GetResourcePools() ([]pool.ResourcePool, error) {
	if t.fail {
		return nil, ErrInvalidPool
	}

	return *t.pools, nil
}

func (t PoolAPITest) GetResourcePool(id string) (*pool.ResourcePool, error) {
	if t.fail {
		return nil, ErrInvalidPool
	}

	for _, p := range *t.pools {
		if p.ID == id {
			return &p, nil
		}
	}

	return nil, nil
}

func (t PoolAPITest) AddResourcePool(config api.PoolConfig) (*pool.ResourcePool, error) {
	if p, err := t.GetResourcePool(config.PoolID); p != nil || err != nil {
		return nil, ErrInvalidPool
	} else if config.PoolID == NilPool {
		return nil, nil
	}

	p := &pool.ResourcePool{
		ID:          config.PoolID,
		CoreLimit:   config.CoreLimit,
		MemoryLimit: config.MemoryLimit,
		Permissions: config.Permissions,
	}

	*t.pools = append(*t.pools, *p)
	return p, nil
}

func (t PoolAPITest) RemoveResourcePool(id string) error {
	if t.fail {
		return ErrInvalidPool
	}

	for i, p := range *t.pools {
		if p.ID == id {
			tmp := *t.pools
			*t.pools = append(tmp[:i], tmp[i+1:]...)
			return nil
		}
	}
	return ErrNoPoolFound
}

func (t PoolAPITest) GetPoolIPs(id string) (*pool.PoolIPs, error) {
	p, err := t.GetResourcePool(id)
	if err != nil {
		return nil, err
	} else if p == nil {
		return nil, ErrNoPoolFound
	}

	return &pool.PoolIPs{PoolID: p.ID, HostIPs: t.hostIPs}, nil
}

func (t PoolAPITest) UpdateResourcePool(pool pool.ResourcePool) error {
	for i, p := range *t.pools {
		if p.ID == pool.ID {
			(*t.pools)[i] = pool
			return nil
		}
	}
	return ErrInvalidPool
}

func TestServicedCLI_CmdPoolList_one(t *testing.T) {
	poolID := "test-pool-id-1"

	test := DefaultPoolAPI()
	expected, err := test.GetResourcePool(poolID)
	if err != nil {
		t.Fatal(err)
	}

	var actual pool.ResourcePool
	output := captureStdout(func() { RunCmd(test, "serviced", "pool", "list", poolID) })
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshalling resource: %s", err)
	}

	// Did you remember to update ResourcePool.Equals?
	if !actual.Equals(expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
}

func TestServicedCLI_CmdPoolList_all(t *testing.T) {
	test := DefaultPoolAPI()
	expected, err := test.GetResourcePools()
	if err != nil {
		t.Fatal(err)
	}

	var actual []*pool.ResourcePool
	output := captureStdout(func() { RunCmd(test, "serviced", "pool", "list", "--verbose") })
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshalling resource: %s", err)
	}

	// Did you remember to update ResourcePool.Equals?
	if len(actual) != len(expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
	for i, _ := range actual {
		if !actual[i].Equals(&expected[i]) {
			t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
		}
	}
}

func ExampleServicedCLI_CmdPoolList() {
	// Gofmt cleans up the spaces at the end of each row
	RunCmd(DefaultPoolAPI(), "serviced", "pool", "list")
}

func ExampleServicedCLI_CmdPoolList_fail() {
	test := DefaultPoolAPI()
	test.fail = true
	// Error retrieving pool
	pipeStderr(func() { RunCmd(test, "serviced", "pool", "list", "test-pool-id-1") })
	// Error retrieving all pools
	pipeStderr(func() { RunCmd(test, "serviced", "pool", "list") })

	// Output:
	// invalid pool
	// invalid pool
}

func ExampleServicedCLI_CmdPoolList_err() {
	test := DefaultPoolAPI()
	*test.pools = make([]pool.ResourcePool, 0)

	// Pool not found
	pipeStderr(func() { RunCmd(test, "serviced", "pool", "list", "test-pool-id-0") })
	// No pools found
	pipeStderr(func() { RunCmd(test, "serviced", "pool", "list") })

	// Output:
	// pool not found
	// no resource pools found
}

func ExampleServicedCLI_CmdPoolList_complete() {
	RunCmd(DefaultPoolAPI(), "serviced", "pool", "list", "--generate-bash-completion")

	// Output:
	// test-pool-id-1
	// test-pool-id-2
	// test-pool-id-3
}

func ExampleServicedCLI_CmdPoolAdd() {
	// // Bad CoreLimit
	// InitPoolAPITest("serviced", "pool", "add", "test-pool", "abc", "1024", "3")
	// // Bad MemoryLimit
	// InitPoolAPITest("serviced", "pool", "add", "test-pool", "4", "abc", "3")
	// Success
	RunCmd(DefaultPoolAPI(), "serviced", "pool", "add", "test-pool", "3")

	// Output:
	// test-pool
}

func ExampleServicedCLI_CmdPoolAdd_err() {
	pipeStderr(func() { RunCmd(DefaultPoolAPI(), "serviced", "pool", "add", NilPool, "4", "1024", "3") })

	// Output:
	// received nil resource pool
}

func TestServicedCLI_CmdPoolAdd_perm(t *testing.T) {
	test := EmptyPoolAPI()
	assertPerm := func(poolID string, expected pool.Permission) {
		if p, err := test.GetResourcePool(poolID); err != nil {
			t.Fatalf("GetResourcePool(\"%s\"): %s", poolID, err.Error())
		} else {
			if p.Permissions != expected {
				t.Fatalf("Unexpected Permission for %s: %d != %d", poolID, p.Permissions, expected)
			}
		}
	}

	poolID := "poolID"
	RunCmd(test, "serviced", "pool", "add", poolID)
	assertPerm(poolID, 0)

	poolID = "pool_DFS"
	RunCmd(test, "serviced", "pool", "add", "--dfs", poolID)
	assertPerm(poolID, pool.DFSAccess)

	poolID = "pool_Admin"
	RunCmd(test, "serviced", "pool", "add", "--admin", poolID)
	assertPerm(poolID, pool.AdminAccess)

	poolID = "pool_Both"
	RunCmd(test, "serviced", "pool", "add", "--dfs", "--admin", poolID)
	assertPerm(poolID, pool.DFSAccess|pool.AdminAccess)
}

func ExampleServicedCLI_CmdPoolRemove() {
	pipeStderr(func() { RunCmd(DefaultPoolAPI(), "serviced", "pool", "remove", "test-pool-id-1") })

	// Output:
	// test-pool-id-1
}

func ExampleServicedCLI_CmdPoolRemove_usage() {
	RunCmd(DefaultPoolAPI(), "serviced", "pool", "rm")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    remove - Removes an existing resource pool
	//
	// USAGE:
	//    command remove [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced pool remove POOLID ...
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdPoolRemove_err() {
	pipeStderr(func() { RunCmd(DefaultPoolAPI(), "serviced", "pool", "remove", "test-pool-id-0") })

	// Output:
	// test-pool-id-0: pool not found
}

func ExampleServicedCLI_CmdPoolRemove_complete() {
	test := DefaultPoolAPI()
	RunCmd(test, "serviced", "pool", "rm", "--generate-bash-completion")
	fmt.Println("")
	RunCmd(test, "serviced", "pool", "rm", "test-pool-id-2", "--generate-bash-completion")

	// Output:
	// test-pool-id-1
	// test-pool-id-2
	// test-pool-id-3
	//
	// test-pool-id-1
	// test-pool-id-3
}

func TestExampleServicedCLI_CmdPoolListIPs(t *testing.T) {
	poolID := "test-pool-id-1"
	test := DefaultPoolAPI()

	var expected []host.HostIPResource
	if ips, err := test.GetPoolIPs(poolID); err != nil {
		t.Fatal(err)
	} else {
		expected = ips.HostIPs
	}

	var actual []host.HostIPResource
	output := captureStdout(func() { RunCmd(test, "serviced", "pool", "list-ips", poolID, "--verbose") })
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshalling resource: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
}

func ExampleServicedCLI_CmdPoolListIPs() {
	// Gofmt cleans up the spaces at the end of each row
	RunCmd(DefaultPoolAPI(), "serviced", "pool", "list-ips", "test-pool-id-1")
}

func ExampleServicedCLI_CmdPoolListIPs_usage() {
	RunCmd(DefaultPoolAPI(), "serviced", "pool", "list-ips")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    list-ips - Lists the IP addresses for a resource pool
	//
	// USAGE:
	//    command list-ips [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced pool list-ips POOLID
	//
	// OPTIONS:
	//    --verbose, -v				Show JSON format
	//    --show-fields 'InterfaceName,IPAddress,Type'	Comma-delimited list describing which fields to display
}

func ExampleServicedCLI_CmdPoolListIPs_fail() {
	pipeStderr(func() { RunCmd(DefaultPoolAPI(), "serviced", "pool", "list-ips", "test-pool-id-0") })

	// Output:
	// no pool found
}

func ExampleServicedCLI_CmdPoolListIPs_err() {
	test := DefaultPoolAPI()
	test.hostIPs = nil
	pipeStderr(func() { RunCmd(test, "serviced", "pool", "list-ips", "test-pool-id-1") })

	// Output:
	// no resource pool IPs found
}

func TestServicedCLI_CmdPoolSetPermission(t *testing.T) {
	test := EmptyPoolAPI()
	assertPerm := func(poolID string, expected pool.Permission) {
		if p, err := test.GetResourcePool(poolID); err != nil {
			t.Fatalf("GetResourcePool(\"%s\"): %s", poolID, err.Error())
		} else {
			if p.Permissions != expected {
				t.Fatalf("Unexpected Permission for %s: %d != %d", poolID, p.Permissions, expected)
			}
		}
	}

	poolID := "poolID"
	RunCmd(test, "serviced", "pool", "add", poolID)
	assertPerm(poolID, 0)
	RunCmd(test, "serviced", "pool", "set-permission", "--dfs", poolID)
	assertPerm(poolID, pool.DFSAccess)
	RunCmd(test, "serviced", "pool", "set-permission", "--admin", poolID)
	assertPerm(poolID, pool.DFSAccess|pool.AdminAccess)
	RunCmd(test, "serviced", "pool", "set-permission", "--dfs=false", poolID)
	assertPerm(poolID, pool.AdminAccess)
	RunCmd(test, "serviced", "pool", "set-permission", "--admin=false", poolID)
	assertPerm(poolID, 0)

	poolID = "poolID_mixed"
	RunCmd(test, "serviced", "pool", "add", "--admin", poolID)
	assertPerm(poolID, pool.AdminAccess)
	RunCmd(test, "serviced", "pool", "set-permission", "--admin=false", "--dfs", poolID)
	assertPerm(poolID, pool.DFSAccess)
	RunCmd(test, "serviced", "pool", "set-permission", "--admin", "--dfs=false", poolID)
	assertPerm(poolID, pool.AdminAccess)
}
