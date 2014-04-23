package cmd

import (
	"errors"

	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/serviced/api"
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
	InitPoolAPITest("serviced", "pool", "add", "test-pool-add", "4", "1024", "3")

	// Output:
	// test-pool-add
}

func ExampleServicedCli_cmdPoolRemove() {
	InitPoolAPITest("serviced", "pool", "remove", "test-pool-id-1", "test-pool-id-2", "test-pool-id-0")

	// Output:
	// test-pool-id-1
	// test-pool-id-2
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
