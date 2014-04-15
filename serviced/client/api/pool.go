package api

import (
	host "github.com/zenoss/serviced/dao"
	pool "github.com/zenoss/serviced/dao"
)

const ()

var ()

// PoolConfig is the deserialized data from the command-line
type PoolConfig struct {
	PoolID      string
	CoreLimit   int
	MemoryLimit uint64
	Priority    int
}

// ListPools returns a list of all pools
func (a *api) ListPools() ([]pool.ResourcePool, error) {
	client, err := connect()
	if err != nil {
		return nil, err
	}

	var poolmap map[string]*pool.ResourcePool
	if err := client.GetResourcePools(&empty, &poolmap); err != nil {
		return nil, fmt.Errorf("could not get resource pools: %s", err)
	}

	pools := make([]pool.ResourcePool, len(poolmap))
	i := 0
	for _, p := range poolmap {
		pools[i] = *p
		i++
	}

	return pools, nil
}

// GetPool gets information about a pool given a PoolID
func (a *api) GetPool(id string) (*pool.ResourcePool, error) {
	client, err := connect()
	if err != nil {
		return nil, err
	}

	var poolmap map[string]*pool.ResourcePool
	if err := client.GetResourcePools(&empty, &poolmap); err != nil {
		return nil, fmt.Errorf("could not get resource pools: %s", err)
	}

	return poolmap[id], nil
}

// AddPool adds a new pool
func (a *api) AddPool(config PoolConfig) (*pool.ResourcePool, error) {
	client, err := connect()
	if err != nil {
		return nil, err
	}

	p := pool.ResourcePool{
		PoolId:      config.PoolID,
		CoreLimit:   config.CoreLimit,
		MemoryLimit: config.MemoryLimit,
		Priority:    config.Priority,
	}
	var id string
	if err := client.AddResourcePool(p, &id); err != nil {
		return nil, fmt.Errorf("could not add resource pool: %s", err)
	}

	var poolmap map[string]*pool.ResourcePool
	if err := client.GetResourcePools(&empty, &poolmap); err != nil {
		return nil, fmt.Errorf("could not get resource pools: %s", err)
	}

	return poolmap[id], nil
}

// RemovePool removes an existing pool
func (a *api) RemovePool(id string) error {
	client, err := connect()
	if err != nil {
		return nil, err
	}

	if err := client.RemoveResourcePool(id, &unusedInt); err != nil {
		fmt.Errorf("could not remove resource pool: %s", err)
	}

	return nil
}

// ListPoolIPs returns a list of Host IPs for a given pool
func (a *api) ListPoolIPs(id string) ([]host.HostIPResource, error) {
	client, err := connect()
	if err != nil {
		return nil, err
	}

	var ipinfo []host.HostIPResource
	if err := client.GetPoolsIPInfo(id, &ipinfo); err != nil {
		return nil, fmt.Errorf("could not obtain pool IP info: %s", err)
	}

	return ipinfo, nil
}
