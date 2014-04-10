package api

import (
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
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
	return nil, nil
}

// GetPool gets information about a pool given a PoolID
func (a *api) GetPool(id string) (*pool.ResourcePool, error) {
	return nil, nil
}

// AddPool adds a new pool
func (a *api) AddPool(config PoolConfig) (*pool.ResourcePool, error) {
	return nil, nil
}

// RemovePool removes an existing pool
func (a *api) RemovePool(id string) error {
	return nil
}

// ListPoolIPs returns a list of Host IPs for a given pool
func (a *api) ListPoolIPs(id string) ([]host.HostIPResource, error) {
	return nil, nil
}
