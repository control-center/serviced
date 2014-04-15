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
		return errors.New("pool not found")
	}
	*reply = *response
	return nil
}

// GetResourcePools returns all ResourcePools
func (s *Server) GetResourcePools(empty interface{}, poolsReply *[]*pool.ResourcePool) error {
	pools, err := s.f.GetResourcePools(s.context())
	if err != nil {
		return err
	}

	*poolsReply = pools
	return nil
}

// AddResourcePool adds the pool
func (s *Server) AddResourcePool(pool pool.ResourcePool, _ *struct{}) error {
	return s.f.AddResourcePool(s.context(), &pool)
}

// UpdateResourcePool updates the pool
func (s *Server) UpdateResourcePool(pool pool.ResourcePool, _ *struct{}) error {
	return s.f.UpdateResourcePool(s.context(), &pool)
}

// GetResourcePool gets the pool
func (s *Server) GetResourcePool(poolID string, reply *pool.ResourcePool) error {
	response, err := s.f.GetResourcePool(s.context(), poolID)
	if err != nil {
		return err
	}
	if response == nil {
		return errors.New("pool not found")
	}
	*reply = *response
	return nil
}

// RemoveResourcePool removes the pool
func (s *Server) RemoveResourcePool(poolID string, _ *struct{}) error {
	return s.f.RemoveResourcePool(s.context(), poolID)
}
