// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import (
	"errors"

	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/facade"
)

// GetResourcePools returns all ResourcePools
func (s *Server) GetResourcePools(empty struct{}, poolsReply *[]*pool.ResourcePool) error {
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

// GetPoolIPs gets all ips available to a pool
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

// AddVirtualIP adds a specific virtual IP to a pool
func (s *Server) AddVirtualIP(requestVirtualIP pool.VirtualIP, _ *struct{}) error {
	return s.f.AddVirtualIP(s.context(), requestVirtualIP)
}

// RemoveVirtualIP removes a specific virtual IP from a pool
func (s *Server) RemoveVirtualIP(requestVirtualIP pool.VirtualIP, _ *struct{}) error {
	return s.f.RemoveVirtualIP(s.context(), requestVirtualIP)
}
