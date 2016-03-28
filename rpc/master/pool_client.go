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

package master

import (
	"github.com/control-center/serviced/domain/pool"
)

//GetResourcePool gets the pool for the given poolID or nil
func (c *Client) GetResourcePool(poolID string) (*pool.ResourcePool, error) {
	response := pool.New(poolID)
	if err := c.call("GetResourcePool", poolID, response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetResourcePools returns all pools or empty array
func (c *Client) GetResourcePools() ([]pool.ResourcePool, error) {
	response := make([]pool.ResourcePool, 0)
	if err := c.call("GetResourcePools", empty, &response); err != nil {
		return []pool.ResourcePool{}, err
	}
	return response, nil
}

//AddResourcePool adds the ResourcePool
func (c *Client) AddResourcePool(pool pool.ResourcePool) error {
	return c.call("AddResourcePool", pool, nil)
}

//UpdateResourcePool adds the ResourcePool
func (c *Client) UpdateResourcePool(pool pool.ResourcePool) error {
	return c.call("UpdateResourcePool", pool, nil)
}

//RemoveResourcePool removes a ResourcePool
func (c *Client) RemoveResourcePool(poolID string) error {
	return c.call("RemoveResourcePool", poolID, nil)
}

//GetPoolIPs returns a all IPs in a ResourcePool.
func (c *Client) GetPoolIPs(poolID string) (*pool.PoolIPs, error) {
	var poolIPs pool.PoolIPs
	if err := c.call("GetPoolIPs", poolID, &poolIPs); err != nil {
		return nil, err
	}
	return &poolIPs, nil
}

//AddVirtualIP adds a VirtualIP to a specificpool
func (c *Client) AddVirtualIP(requestVirtualIP pool.VirtualIP) error {
	return c.call("AddVirtualIP", requestVirtualIP, nil)
}

//RemoveVirtualIP removes a VirtualIP from a specific pool
func (c *Client) RemoveVirtualIP(requestVirtualIP pool.VirtualIP) error {
	return c.call("RemoveVirtualIP", requestVirtualIP, nil)
}
