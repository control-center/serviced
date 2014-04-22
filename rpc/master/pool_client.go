// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import (
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/facade"
)

//GetPoolIPs returns a all IPs in a ResourcePool.
func (c *Client) GetPoolIPs(poolID string) (*facade.PoolIPs, error) {
	var poolIPs facade.PoolIPs
	if err := c.call("GetPoolIPs", poolID, &poolIPs); err != nil {
		return nil, err
	}
	return &poolIPs, nil
}

//GetResourcePool gets the host for the give hostID or nil
func (c *Client) GetResourcePool(poolID string) (*pool.ResourcePool, error) {
	response := pool.New(poolID)
	if err := c.call("GetResourcePools", poolID, response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetResourcePools returns all pools or empty array
func (c *Client) GetResourcePools() ([]*pool.ResourcePool, error) {
	response := make([]*pool.ResourcePool, 0)
	if err := c.call("GetResourcePools", empty, &response); err != nil {
		return []*pool.ResourcePool{}, err
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
