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

package virtualips

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
)

var (
	// ErrNoAssignedHost is returned when the call expects the virtual IP to
	// already be assigned to a host
	ErrNoAssignedHost = errors.New("unable to find assigned host")

	// ErrAlreadyAssigned is returned when the call expects the virtual IP is not
	// already assigned to a host
	ErrAlreadyAssigned = errors.New("virtual IP has already been assigned")
)

// AssignmentHandler is used to assign, unassign, and watch virtual IP assignments
// to hosts
type AssignmentHandler interface {
	Assign(poolID, ipAddress, netmask, binding string, cancel <-chan interface{}) error
	Unassign(poolID, ipAddress string) error
	Watch(poolID, ipAddress string, stop <-chan struct{}) (<-chan client.Event, error)
}

// ZKAssignmentHandler implements the AssignmentHandler interface.  Assignments are
// created using ZooKeeper by creating and deleting the appropriate nodes.  The
// following paths are used:
//
// 		/pools/poolid/ips/hostid-ipaddress
// 		/pools/poolid/hosts/hostid/ips/hostid-ipaddress
//
type ZKAssignmentHandler struct {
	connection            client.Connection
	hostHandler           RegisteredHostHandler
	hostSelectionStrategy HostSelectionStrategy
}

// NewZKAssignmentHandler returns a new ZKAssignmentHandler with the provided
// dependencies.
func NewZKAssignmentHandler(strategy HostSelectionStrategy,
	handler RegisteredHostHandler,
	connection client.Connection) *ZKAssignmentHandler {
	return &ZKAssignmentHandler{
		hostSelectionStrategy: strategy,
		hostHandler:           handler,
		connection:            connection,
	}
}

// Assign will assign the provided virtual IP to a host.  If no host is present,
// the call will block until host comes online.  The cancel channel parameter can be used
// to cancel the assignment request.  If a IP address is already assigned to a host, ErrAlreadyAssigned
// will be returned.
func (h *ZKAssignmentHandler) Assign(poolID, ipAddress, netmask, binding string, cancel <-chan interface{}) error {
	_, err := h.getAssignedHostID(poolID, ipAddress)
	if err == ErrNoAssignedHost {
		return h.assignToHost(poolID, ipAddress, netmask, binding, cancel)
	} else if err == nil {
		return ErrAlreadyAssigned
	}
	return err
}

// Unassign will unassign a virtual IP if it is currently assigned to a host.  ErrNoHostAssigned will be
// returned if there is no host for the virtual IP when Unassign is called.
func (h *ZKAssignmentHandler) Unassign(poolID, ipAddress string) error {
	assignedHost, err := h.getAssignedHostID(poolID, ipAddress)
	if err != nil {
		return err
	}

	plog.WithFields(log.Fields{
		"poolid":    poolID,
		"ipAddress": ipAddress,
		"host":      assignedHost,
	}).Debug("Unassigning IP")

	request := IPRequest{PoolID: poolID, HostID: assignedHost, IPAddress: ipAddress}
	return DeleteIP(h.connection, request)
}

// Watch will return a channel that will recieve events when an assignment changes.  An example would
// be if a host were to go offline and delete the nodes for the assignment in ZooKeeper.  If Watch is
// called for an IP with no assignment, ErrNoAssignedHost will be returned.
func (h *ZKAssignmentHandler) Watch(poolID, ipAddress string, stop <-chan struct{}) (<-chan client.Event, error) {
	assignedHost, err := h.getAssignedHostID(poolID, ipAddress)
	if err != nil {
		return nil, err
	}

	request := IPRequest{PoolID: poolID, HostID: assignedHost, IPAddress: ipAddress}
	path := Base().Pools().ID(poolID).IPs().ID(request.IPID()).Path()
	return h.connection.GetW(path, &PoolIP{}, stop)
}

func (h *ZKAssignmentHandler) assignToHost(poolID, ipAddress, netmask, binding string, cancel <-chan interface{}) error {
	hosts, err := h.hostHandler.GetRegisteredHosts(cancel)
	if err != nil {
		return err
	}

	host, err := h.hostSelectionStrategy.Select(hosts)
	if err != nil {
		return err
	}

	plog.WithFields(log.Fields{
		"poolid":    poolID,
		"ipAddress": ipAddress,
		"host":      host.ID,
	}).Debug("Assigning IP")

	request := IPRequest{PoolID: poolID, HostID: host.ID, IPAddress: ipAddress}
	return CreateIP(h.connection, request, netmask, binding)
}

func (h *ZKAssignmentHandler) getAssignedHostID(poolID, ipAddress string) (string, error) {
	ipsPath := Base().Pools().ID(poolID).IPs().Path()
	exists, err := h.connection.Exists(ipsPath)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", ErrNoAssignedHost
	}

	ipIDs, err := h.connection.Children(Base().Pools().ID(poolID).IPs().Path())
	if err != nil {
		return "", err
	}

	for _, ipID := range ipIDs {
		host, ip, err := ParseIPID(ipID)
		if err != nil {
			return "", err
		}
		if ip == ipAddress {
			return host, nil
		}
	}
	return "", ErrNoAssignedHost
}
