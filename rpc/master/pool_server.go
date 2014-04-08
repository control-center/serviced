// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import (
	"github.com/zenoss/serviced/facade"

	"errors"
	"github.com/zenoss/serviced/domain/pool"
)

// GetPoolIPs gets all ips available to a Pool
func (s *Server) GetPoolIPs(poolID string, reply *facade.PoolIPs) error {
	response, err := s.f.GetPoolIPs(s.context(), poolID)
	if err != nil {
		return err
	}
	if response == nil {
		return errors.New("host not found")
	}
	*reply = *response
	return nil
}

func (s *Server) GetResourcePools() ([]*pool.Pools, error) {
	return s.f.GetResourcePools(s.context())
}

//
//func (s *ControlClient) AddResourcePool(pool dao.ResourcePool, poolId *string) (err error) {
//	return s.rpcClient.Call("ControlPlane.AddResourcePool", pool, poolId)
//}
//
//func (s *ControlClient) UpdateResourcePool(pool dao.ResourcePool, unused *int) (err error) {
//	return s.rpcClient.Call("ControlPlane.UpdateResourcePool", pool, unused)
//}
//
//func (s *ControlClient) RemoveResourcePool(poolId string, unused *int) (err error) {
//	return s.rpcClient.Call("ControlPlane.RemoveResourcePool", poolId, unused)
//}
//
//func (s *ControlClient) GetHostsForResourcePool(poolId string, poolHosts *[]*dao.PoolHost) (err error) {
//	return s.rpcClient.Call("ControlPlane.GetHostsForResourcePool", poolId, poolHosts)
//}
//
//func (s *ControlClient) AddHostToResourcePool(poolHost dao.PoolHost, unused *int) error {
//	return s.rpcClient.Call("ControlPlane.AddHostToResourcePool", poolHost, unused)
//}
//
//func (s *ControlClient) RemoveHostFromResourcePool(poolHost dao.PoolHost, unused *int) error {
//	return s.rpcClient.Call("ControlPlane.RemoveHostFromResourcePool", poolHost, unused)
//}

