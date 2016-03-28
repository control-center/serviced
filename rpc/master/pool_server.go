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
	"errors"

	"github.com/control-center/serviced/domain/pool"
)

// GetResourcePools returns all ResourcePools
func (s *Server) GetResourcePools(empty struct{}, poolsReply *[]pool.ResourcePool) error {
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
func (s *Server) GetPoolIPs(poolID string, reply *pool.PoolIPs) error {
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
