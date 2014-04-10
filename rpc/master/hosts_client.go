// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import (
	"github.com/zenoss/serviced/domain/host"
)

//GetHost gets the host for the give hostID or nil
func (c *Client) GetHost(hostID string) (*host.Host, error) {
	response := host.New()
	if err := c.call("ControlPlane.GetHost", hostID, response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetHosts returns all hosts or empty array
func (c *Client) GetHosts() ([]*host.Host, error) {
	response := make([]*host.Host, 0)
	if err := c.rpcClient.Call("GetHosts", nil, &response); err != nil {
		return []*host.Host{}, err
	}
	return response, nil
}

func (c *Client) AddHost(host host.Host) error {
	return c.call("AddHost", host, nil)
}

func (c *Client) UpdateHost(host host.Host) error {
	return c.call("UpdateHost", host, nil)
}

func (c *Client) RemoveHost(hostId string) error {
	return c.call("RemoveHost", hostId, nil)
}


// FindHostsInPool returns all hosts in a pool
func (c *Client) FindHostsInPool(poolID string) ([]*host.Host, error) {
	response := make([]*host.Host, 0)
	if err := c.rpcClient.Call("FindHostsInPool", nil, &response); err != nil {
		return []*host.Host{}, err
	}
	return response, nil
}
