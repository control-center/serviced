// Copyright 2016 The Serviced Authors.
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

package service

import (
	p "github.com/control-center/serviced/domain/pool"
)

// VirtualIPSynchronizer will sync virtual IP assignments for a pool.
type VirtualIPSynchronizer interface {
	Sync(pool p.ResourcePool, assignments map[string]string, cancel <-chan interface{}) error
}

// ZKVirtualIPSynchronizer implements the VirtualIPSynchronizer interface for ZooKeeper.
type ZKVirtualIPSynchronizer struct {
	handler AssignmentHandler
}

// NewZKVirtualIPSynchronizer returns a new ZKVirtualIPSynchronizer
func NewZKVirtualIPSynchronizer(handler AssignmentHandler) *ZKVirtualIPSynchronizer {
	return &ZKVirtualIPSynchronizer{handler: handler}
}

// Sync will synchronize virtual IP assignments for a pool.  It will assign virtual IPs that
// are not assigned to a host.  It will unassign any active assignments if the virtual IP as been
// removed from the pool.
func (s *ZKVirtualIPSynchronizer) Sync(pool p.ResourcePool, assignments map[string]string, cancel <-chan interface{}) error {
	virtualIPMap := s.getVirtualIPMap(pool)

	for _, ip := range s.virtualIPsWithNoAssignment(virtualIPMap, assignments) {
		s.handler.Assign(ip.PoolID, ip.IP, ip.Netmask, ip.BindInterface, cancel)
	}

	for _, ip := range s.assignmentsWithNoVirtualIP(virtualIPMap, assignments) {
		s.handler.Unassign(pool.ID, ip)
	}

	return nil
}

func (s *ZKVirtualIPSynchronizer) getVirtualIPMap(pool p.ResourcePool) map[string]p.VirtualIP {
	ips := map[string]p.VirtualIP{}
	for _, ip := range pool.VirtualIPs {
		ips[ip.IP] = ip
	}
	return ips
}

func (s *ZKVirtualIPSynchronizer) virtualIPsWithNoAssignment(virtualIPs map[string]p.VirtualIP, assignments map[string]string) []p.VirtualIP {
	unassigned := []p.VirtualIP{}
	for _, ip := range virtualIPs {
		if _, ok := assignments[ip.IP]; !ok {
			unassigned = append(unassigned, ip)
		}
	}
	return unassigned
}

func (s *ZKVirtualIPSynchronizer) assignmentsWithNoVirtualIP(virtualIPs map[string]p.VirtualIP, assignments map[string]string) []string {
	orphaned := []string{}
	for assignmentIP := range assignments {
		if _, ok := virtualIPs[assignmentIP]; !ok {
			orphaned = append(orphaned, assignmentIP)
		}
	}
	return orphaned
}
