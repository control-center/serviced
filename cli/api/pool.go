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

package api

import (
	"github.com/control-center/serviced/domain/pool"
)

const ()

var ()

// PoolConfig is the deserialized data from the command-line
type PoolConfig struct {
	PoolID      string
	Realm       string
	CoreLimit   int
	MemoryLimit uint64
}

// Returns a list of all pools
func (a *api) GetResourcePools() ([]pool.ResourcePool, error) {
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
		Realm:       config.Realm,
		CoreLimit:   config.CoreLimit,
		MemoryLimit: config.MemoryLimit,
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
func (a *api) GetPoolIPs(id string) (*pool.PoolIPs, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.GetPoolIPs(id)
}

// Add a VirtualIP to a specific pool
func (a *api) AddVirtualIP(requestVirtualIP pool.VirtualIP) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.AddVirtualIP(requestVirtualIP)
}

// Add a VirtualIP to a specific pool
func (a *api) RemoveVirtualIP(requestVirtualIP pool.VirtualIP) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.RemoveVirtualIP(requestVirtualIP)
}
