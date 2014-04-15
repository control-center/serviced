// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import (
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/facade"
)

func (c *Client) GetPoolIPs(poolId string) (*facade.PoolIPs, error) {

	var poolIPs facade.PoolIPs
	if err := c.call("GetPoolsIPInfo", poolId, &poolIPs); err != nil {
		return nil, err
	}
	return &poolIPs, nil
}

//GetHost gets the host for the give hostID or nil
func (c *Client) GetResourcePool(poolID string) (*pool.ResourcePool, error) {
	response := pool.New(poolID)
	if err := c.call("ControlPlane.GetResourcePools", poolID, response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetResourcePools returns all pools or empty array
func (c *Client) GetResourcePools() ([]*pool.ResourcePool, error) {
	response := make([]*pool.ResourcePool, 0)
	if err := c.rpcClient.Call("GetResourcePools", nil, &response); err != nil {
		return []*pool.ResourcePool{}, err
	}
	return response, nil
}

func (c *Client) AddResourcePool(pool pool.ResourcePool) error {
	return c.call("AddResourcePool", pool, nil)
}

func (c *Client) UpdateResourcePool(pool pool.ResourcePool) error {
	return c.call("UpdateResourcePool", pool, nil)
}

func (c *Client) RemoveResourcePool(poolID string) error {
	return c.call("RemoveResourcePool", poolID, nil)
}
