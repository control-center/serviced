// Copyright 2017 The Serviced Authors.
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
	"github.com/control-center/serviced/coordinator/client"
)

// HostUnassignmentHandler will handle unassigning the virtual ips that are assigned
// to a host
type HostUnassignmentHandler interface {
	UnassignAll(poolID, hostID string) error
}

// ZKHostUnassignmentHandler implements HostUnassignmentHandler.  It will remove the nodes
// in ZooKeeper for a host. The paths affected are the following:
//
// 		/pools/poolid/hosts/hostid/ips/hostid-ipaddress
// 		/pools/poolid/ips/hostid-ipaddress
//
type ZKHostUnassignmentHandler struct {
	connection client.Connection
}

// NewZKHostUnassignmentHandler returns a new ZKHostUnassignmentHandler instance.
func NewZKHostUnassignmentHandler(connection client.Connection) *ZKHostUnassignmentHandler {
	return &ZKHostUnassignmentHandler{
		connection: connection,
	}
}

// UnassignAll will remove all virtual IP nodes for a host in ZooKeeper.
func (h *ZKHostUnassignmentHandler) UnassignAll(poolID, hostID string) error {
	path := Base().Pools().ID(poolID).Hosts().ID(hostID).IPs().Path()
	exists, err := h.connection.Exists(path)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	ipIDs, err := h.connection.Children(path)
	if err != nil {
		return err
	}

	for _, ipID := range ipIDs {
		_, ip, err := ParseIPID(ipID)
		if err != nil {
			return err
		}

		request := IPRequest{
			PoolID:    poolID,
			HostID:    hostID,
			IPAddress: ip,
		}

		err = DeleteIP(h.connection, request)
		if err != nil {
			return err
		}
	}

	return nil
}
