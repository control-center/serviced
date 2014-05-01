package cmd

import (
	"errors"

	"github.com/zenoss/serviced/cli/api"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/facade"
)

const (
	NilPool = "NilPool"
)

var DefaultPoolAPITest = PoolAPITest{pools: DefaultTestPools, hostIPs: DefaultTestHostIPs}

var DefaultTestPools = []*pool.ResourcePool{
	{
		ID:          "test-pool-id-1",
		ParentID:    "",
		Priority:    1,
		CoreLimit:   8,
		MemoryLimit: 0,
	}, {
		ID:          "test-pool-id-2",
		ParentID:    "test-pool-id-1",
		Priority:    2,
		CoreLimit:   4,
		MemoryLimit: 4 * 1024 * 1024 * 1024,
	}, {
		ID:          "test-pool-id-3",
		ParentID:    "test-pool-id-1",
		Priority:    3,
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
	pools   []*pool.ResourcePool
	hostIPs []host.HostIPResource
}

func InitPoolAPITest(args ...string) {
	New(DefaultPoolAPITest).Run(args)
}

func (t PoolAPITest) GetResourcePools() ([]*pool.ResourcePool, error) {
	return t.pools, nil
}

func (t PoolAPITest) GetResourcePool(id string) (*pool.ResourcePool, error) {
	for _, p := range t.pools {
		if p.ID == id {
			return p, nil
		}
	}

	return nil, ErrNoPoolFound
}

func (t PoolAPITest) AddResourcePool(config api.PoolConfig) (*pool.ResourcePool, error) {
	for _, p := range t.pools {
		if p.ID == config.PoolID {
			return nil, ErrInvalidPool
		}
	}

	if config.PoolID == NilPool {
		return nil, nil
	}

	p := &pool.ResourcePool{
		ID:          config.PoolID,
		ParentID:    "",
		Priority:    0,
		CoreLimit:   config.CoreLimit,
		MemoryLimit: config.MemoryLimit,
	}

	return p, nil
}

func (t PoolAPITest) RemoveResourcePool(id string) error {
	_, err := t.GetResourcePool(id)
	return err
}

func (t PoolAPITest) GetPoolIPs(id string) (*facade.PoolIPs, error) {
	for _, p := range t.pools {
		if p.ID == id {
			return &facade.PoolIPs{HostIPs: t.hostIPs}, nil
		}
	}

	return nil, ErrNoPoolFound
}

func ExampleServicedCli_cmdPoolList() {
	InitPoolAPITest("serviced", "pool", "list", "-v")

	// Output:

	// [
	//    {
	//      "ID": "test-pool-id-1",
	//      "Description": "",
	//      "ParentID": "",
	//      "Priority": 1,
	//      "CoreLimit": 8,
	//      "MemoryLimit": 0,
	//      "CreatedAt": "0001-01-01T00:00:00Z",
	//      "UpdatedAt": "0001-01-01T00:00:00Z"
	//    },
	//    {
	//      "ID": "test-pool-id-2",
	//      "Description": "",
	//      "ParentID": "test-pool-id-1",
	//      "Priority": 2,
	//      "CoreLimit": 4,
	//      "MemoryLimit": 4294967296,
	//      "CreatedAt": "0001-01-01T00:00:00Z",
	//      "UpdatedAt": "0001-01-01T00:00:00Z"
	//    },
	//    {
	//      "ID": "test-pool-id-3",
	//      "Description": "",
	//      "ParentID": "test-pool-id-1",
	//      "Priority": 3,
	//      "CoreLimit": 2,
	//      "MemoryLimit": 536870912,
	//      "CreatedAt": "0001-01-01T00:00:00Z",
	//      "UpdatedAt": "0001-01-01T00:00:00Z"
	//    }
	//  ]
}

func ExampleServicedCli_cmdPoolAdd() {
	// Existing pool
	InitPoolAPITest("serviced", "pool", "add", "test-pool-id-1", "4", "1024", "3")
	// Received nil resource pool
	InitPoolAPITest("serviced", "pool", "add", NilPool, "4", "1024", "3")
	// Success
	InitPoolAPITest("serviced", "pool", "add", "test-pool-add", "4", "1024", "3")

	// Output:
	// test-pool-add
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
	//    serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdPoolAdd_badparam() {
	InitPoolAPITest("serviced", "pool", "add", "bad-pool-1", "abc", "1024", "3")
	InitPoolAPITest("serviced", "pool", "add", "bad-pool-2", "4", "abc", "3")
	InitPoolAPITest("serviced", "pool", "add", "bad-pool-3", "4", "1024", "abc")

	// Output:
	// CORE_LIMIT must be a number
	// MEMORY_LIMIT must be a number
	// PRIORITY must be a number
}

func ExampleServicedCli_cmdPoolRemove() {
	InitPoolAPITest("serviced", "pool", "remove", "test-pool-id-1", "test-pool-id-2", "test-pool-id-0")

	// Output:
	// test-pool-id-1
	// test-pool-id-2
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

func ExampleServicedCli_cmdPoolListIPs() {
	InitPoolAPITest("serviced", "pool", "list-ips", "test-pool-id-1", "--verbose")

	// Output:
	// [
	//    {
	//      "HostID": "test-host-id-1",
	//      "IPAddress": "127.0.0.1",
	//      "InterfaceName": "test-interface-name-1"
	//    },
	//    {
	//      "HostID": "test-host-id-2",
	//      "IPAddress": "192.168.0.1",
	//      "InterfaceName": "test-interface-name-2"
	//    },
	//    {
	//      "HostID": "test-host-id-3",
	//      "IPAddress": "0.0.0.0",
	//      "InterfaceName": "test-interface-name-3"
	//    }
	//  ]
}
