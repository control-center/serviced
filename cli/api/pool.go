package api

import (
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/facade"
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

// Returns a list of all pools
func (a *api) GetResourcePools() ([]*pool.ResourcePool, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.GetResourcePools()
}

// Gets information about a pool given a PoolID
func (a *api) GetResourcePool(id string) (*pool.ResourcePool, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.GetResourcePool(id)
}

// Adds a new pool
func (a *api) AddResourcePool(config PoolConfig) (*pool.ResourcePool, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	p := pool.ResourcePool{
		ID:          config.PoolID,
		CoreLimit:   config.CoreLimit,
		MemoryLimit: config.MemoryLimit,
		Priority:    config.Priority,
	}

	if err := client.AddResourcePool(p); err != nil {
		return nil, err
	}

	return a.GetResourcePool(p.ID)
}

// Removes an existing pool
func (a *api) RemoveResourcePool(id string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.RemoveResourcePool(id)
}

// Returns a list of Host IPs for a given pool
func (a *api) GetPoolIPs(id string) (*facade.PoolIPs, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.GetPoolIPs(id)
}

// Add a VirtualIP to a specific pool
func (a *api) AddVirtualIP(poolID string, ipAddress string, netmask string, bindInterface string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.AddVirtualIP(poolID, ipAddress, netmask, bindInterface)
}

// Add a VirtualIP to a specific pool
func (a *api) RemoveVirtualIP(virtualIPID string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.RemoveVirtualIP(virtualIPID)
}
