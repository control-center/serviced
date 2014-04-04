// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import "github.com/zenoss/serviced/facade"

func (c *Client) GetPoolIPs(poolId string) (*facade.PoolIPs, error) {

	var poolIPs facade.PoolIPs
	if err := c.call("GetPoolsIPInfo", poolId, &poolIPs); err != nil {
		return nil, err
	}
	return &poolIPs, nil
}
