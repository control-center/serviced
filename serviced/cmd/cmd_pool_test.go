package cmd

import (
	"errors"
	"fmt"

	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/serviced/api"
)

var DefaultPoolAPITest = PoolAPITest{pools: DefaultTestPools, hostIPs: DefaultTestHostIPs}

var DefaultTestPools = []pool.ResourcePool{
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
		IpAddress:     "192.168.0.1",
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
	pools   []pool.ResourcePool
	hostIPs []host.HostIPResource
}

func InitPoolAPITest(args ...string) {
	New(DefaultPoolAPITest).Run(args)
}

func ExampleServicedCli_cmdPoolList() {
	New(api.New()).Run([]string{"serviced", "pool", "list"})

	// Output:
	// serviced pool list
}

func (t PoolAPITest) ListPools() ([]pool.ResourcePool, error) {
	return t.pools, nil
}

func (t PoolAPITest) GetPool(id string) (*pool.ResourcePool, error) {
	for _, p := range p.pools {
		if p.ID == id {
			return &p
		}
	}

	return ErrNoPoolFound
}

func (t PoolAPITest) AddPool(config PoolConfig) (*pool.ResourcePool, error) {
	for _, p := range t.pools {
		if p.ID == config.PoolID {
			return ErrInvalidPool
		}
	}

	p := &pool.ResourcePool{
		ID:          config.PoolID,
		ParentID:    "",
		Priority:    0,
		CoreLimit:   config.CoreLimit,
		MemoryLimit: cfg.MemoryLimit,
	}

	return p
}

func (t PoolAPITest) RemovePool(id string) error {
	for _, p := range t.pools {
		if p.ID == id {
			return nil
		}
	}

	return ErrNoPoolFound
}

func (t PoolAPITest) ListIPs(id string) ([]host.HostIPResource, error) {
	for _, p := range t.pools {
		if p.ID == id {
			return t.hostIPs, nil
		}
	}

	return ErrNoPoolFound
}

func ExampleServicedCli_cmdPoolList() {
	InitPoolAPITest("serviced", "pool", "list")

	// Output:
	//
}

func ExampleServicedCli_cmdPoolAdd() {
	InitPoolAPITest("serviced", "pool", "add", "test-pool-add", "4", "1024", "3")

	// Output:
	// test-pool-add
}

func ExampleServicedCli_cmdPoolRemove() {
	InitPoolAPITest("serviced", "pool", "remove", "test-pool-id-1")

	// Output:
	// test-pool-id-1
}

func ExampleServicedCli_cmdPoolListIPs() {
	InitPoolAPITest("serviced", "pool", "list-ips", "test-pool-id-1")

	// Output:
	//
}
