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

var DefaultPoolAPITest = PoolAPITest{pools: DefaultTestPools, hostIPs: DefaultTestHostIPs}

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
	pools   []pool.ResourcePool
	hostIPs []host.HostIPResource
}

func InitPoolAPITest(args ...string) {
	New(DefaultPoolAPITest, utils.TestConfigReader(make(map[string]string))).Run(args)
}

func (t PoolAPITest) GetResourcePools() ([]pool.ResourcePool, error) {
	if t.fail {
		return nil, ErrInvalidPool
	}

	return t.pools, nil
}

func (t PoolAPITest) GetResourcePool(id string) (*pool.ResourcePool, error) {
	if t.fail {
		return nil, ErrInvalidPool
	}

	for _, p := range t.pools {
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
	}

	return p, nil
}

func (t PoolAPITest) RemoveResourcePool(id string) error {
	if p, err := t.GetResourcePool(id); err != nil {
		return err
	} else if p == nil {
		return ErrNoPoolFound
	}

	return nil
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

func TestServicedCLI_CmdPoolList_one(t *testing.T) {
	poolID := "test-pool-id-1"

	expected, err := DefaultPoolAPITest.GetResourcePool(poolID)
	if err != nil {
		t.Fatal(err)
	}

	var actual pool.ResourcePool
	output := pipe(InitPoolAPITest, "serviced", "pool", "list", poolID)
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshalling resource: %s", err)
	}

	// Did you remember to update ResourcePool.Equals?
	if !actual.Equals(expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
}

func TestServicedCLI_CmdPoolList_all(t *testing.T) {
	expected, err := DefaultPoolAPITest.GetResourcePools()
	if err != nil {
		t.Fatal(err)
	}

	var actual []*pool.ResourcePool
	output := pipe(InitPoolAPITest, "serviced", "pool", "list", "--verbose")
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
	InitPoolAPITest("serviced", "pool", "list")
}

func ExampleServicedCLI_CmdPoolList_fail() {
	DefaultPoolAPITest.fail = true
	defer func() { DefaultPoolAPITest.fail = false }()
	// Error retrieving pool
	pipeStderr(InitPoolAPITest, "serviced", "pool", "list", "test-pool-id-1")
	// Error retrieving all pools
	pipeStderr(InitPoolAPITest, "serviced", "pool", "list")

	// Output:
	// invalid pool
	// invalid pool
}

func ExampleServicedCLI_CmdPoolList_err() {
	DefaultPoolAPITest.pools = make([]pool.ResourcePool, 0)
	defer func() { DefaultPoolAPITest.pools = DefaultTestPools }()
	// Pool not found
	pipeStderr(InitPoolAPITest, "serviced", "pool", "list", "test-pool-id-0")
	// No pools found
	pipeStderr(InitPoolAPITest, "serviced", "pool", "list")

	// Output:
	// pool not found
	// no resource pools found
}

func ExampleServicedCLI_CmdPoolList_complete() {
	InitPoolAPITest("serviced", "pool", "list", "--generate-bash-completion")

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
	InitPoolAPITest("serviced", "pool", "add", "test-pool", "3")

	// Output:
	// test-pool
}

func ExampleServicedCLI_CmdPoolAdd_usage() {
	InitPoolAPITest("serviced", "pool", "add")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    add - Adds a new resource pool
	//
	// USAGE:
	//    command add [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced pool add POOLID
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdPoolAdd_err() {
	pipeStderr(InitPoolAPITest, "serviced", "pool", "add", NilPool, "4", "1024", "3")

	// Output:
	// received nil resource pool
}

func ExampleServicedCLI_CmdPoolRemove() {
	InitPoolAPITest("serviced", "pool", "remove", "test-pool-id-1")

	// Output:
	// test-pool-id-1
}

func ExampleServicedCLI_CmdPoolRemove_usage() {
	InitPoolAPITest("serviced", "pool", "rm")

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
	pipeStderr(InitPoolAPITest, "serviced", "pool", "remove", "test-pool-id-0")

	// Output:
	// test-pool-id-0: pool not found
}

func ExampleServicedCLI_CmdPoolRemove_complete() {
	InitPoolAPITest("serviced", "pool", "rm", "--generate-bash-completion")
	fmt.Println("")
	InitPoolAPITest("serviced", "pool", "rm", "test-pool-id-2", "--generate-bash-completion")

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

	var expected []host.HostIPResource
	if ips, err := DefaultPoolAPITest.GetPoolIPs(poolID); err != nil {
		t.Fatal(err)
	} else {
		expected = ips.HostIPs
	}

	var actual []host.HostIPResource
	output := pipe(InitPoolAPITest, "serviced", "pool", "list-ips", poolID, "--verbose")
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshalling resource: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
}

func ExampleServicedCLI_CmdPoolListIPs() {
	// Gofmt cleans up the spaces at the end of each row
	InitPoolAPITest("serviced", "pool", "list-ips", "test-pool-id-1")
}

func ExampleServicedCLI_CmdPoolListIPs_usage() {
	InitPoolAPITest("serviced", "pool", "list-ips")

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
	pipeStderr(InitPoolAPITest, "serviced", "pool", "list-ips", "test-pool-id-0")

	// Output:
	// no pool found
}

func ExampleServicedCLI_CmdPoolListIPs_err() {
	DefaultPoolAPITest.hostIPs = nil
	defer func() { DefaultPoolAPITest.hostIPs = DefaultTestHostIPs }()
	pipeStderr(InitPoolAPITest, "serviced", "pool", "list-ips", "test-pool-id-1")

	// Output:
	// no resource pool IPs found
}
