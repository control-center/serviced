package api

import (
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
)

const ()

var ()

// PoolConfig is the deserialized data from the command-line
type PoolConfig struct {
}

// ListPools returns a list of all pools
func (a *api) ListPools() ([]pool.Pool, error) {
	return nil, nil
}

// GetPool gets information about a pool given a PoolID
func (a *api) GetPool(id string) (*pool.Pool, error) {
	return nil, nil
}

// AddPool adds a new pool
func (a *api) AddPool(config PoolConfig) (*pool.Pool, error) {
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
