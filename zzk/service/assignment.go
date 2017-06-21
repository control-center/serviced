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
	"errors"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
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
	Assign(poolID, ipAddress, netmask, binding string) error
	Unassign(poolID, ipAddress string) error
}

// ZKAssignmentHandler implements the AssignmentHandler interface.  Assignments are
// created using ZooKeeper by creating and deleting the appropriate nodes.  The
// following paths are used:
//
// 		/pools/poolid/ips/hostid-ipaddress
// 		/pools/poolid/hosts/hostid/ips/hostid-ipaddress
//
type ZKAssignmentHandler struct {
	Timeout               time.Duration
	connection            client.Connection
	hostHandler           RegisteredHostHandler
	hostSelectionStrategy HostSelectionStrategy
	mu                    *sync.Mutex
	timeouts              map[string]map[string]time.Time
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
		mu:                    &sync.Mutex{},
		Timeout:               time.Second * 10,
		timeouts:              make(map[string]map[string]time.Time),
	}
}

// Assign will assign the provided virtual IP to a host. If an IP is assigned to host there is a 10s timeout until that
// IP can be assigned to the host again.  This prevents spamming of errors. If a IP address is already assigned to a host,
// ErrAlreadyAssigned will be returned.  If no hosts are availd, ErrNoHosts will be returned.
func (h *ZKAssignmentHandler) Assign(poolID, ipAddress, netmask, binding string) error {
	_, err := h.getAssignedHostID(poolID, ipAddress)
	if err == ErrNoAssignedHost {
		return h.assignToHost(poolID, ipAddress, netmask, binding)
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
		"ipaddress": ipAddress,
		"host":      assignedHost,
	}).Debug("Unassigning IP")

	request := IPRequest{PoolID: poolID, HostID: assignedHost, IPAddress: ipAddress}
	return DeleteIP(h.connection, request)
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

func getIDString(hosts []host.Host) string {
	hostIDs := []string{}

	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID+":"+h.PoolID)
	}

	return strings.Join(hostIDs, ",")
}

func (h *ZKAssignmentHandler) assignToHost(poolID, ipAddress, netmask, binding string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	logger := plog.WithFields(log.Fields{
		"poolid":    poolID,
		"ipaddress": ipAddress,
	})

	logger.Debug("Assigning IP")

	hosts, err := h.hostHandler.GetRegisteredHosts(poolID)
	if err != nil {
		return err
	}

	logger.WithFields(log.Fields{
		"count":     len(hosts),
		"ipaddress": getIDString(hosts),
	}).Debug("Found hosts")

	hosts = h.filterOutExcludedHosts(ipAddress, hosts)
	if len(hosts) == 0 {
		return ErrNoHosts
	}

	logger.WithFields(log.Fields{
		"count":     len(hosts),
		"ipaddress": getIDString(hosts),
	}).Debug("Selecting from hosts")

	host, err := h.hostSelectionStrategy.Select(hosts)
	if err != nil {
		return err
	}

	h.addExcludeHost(ipAddress, host)

	plog.WithField("host", host.ID).Debug("Assigning IP")

	request := IPRequest{PoolID: poolID, HostID: host.ID, IPAddress: ipAddress}
	return CreateIP(h.connection, request, netmask, binding)
}

func (h *ZKAssignmentHandler) addExcludeHost(ipAddress string, host host.Host) {
	if _, ok := h.timeouts[ipAddress]; !ok {
		h.timeouts[ipAddress] = make(map[string]time.Time)
	}

	h.timeouts[ipAddress][host.ID] = time.Now().Add(h.Timeout)
}

func (h *ZKAssignmentHandler) getExcludedHostIDs(ipAddress string) []string {
	excludedHostIDs := []string{}
	now := time.Now()

	if excludeHosts, ok := h.timeouts[ipAddress]; ok {
		for hostID, timeout := range excludeHosts {
			if timeout.After(now) {
				excludedHostIDs = append(excludedHostIDs, hostID)
			}
		}
	}

	return excludedHostIDs
}

func (h *ZKAssignmentHandler) filterOutExcludedHosts(ipAddress string, hosts []host.Host) []host.Host {
	excludedHostIDs := h.getExcludedHostIDs(ipAddress)
	filteredHosts := []host.Host{}

	for _, host := range hosts {
		if !containsHostID(excludedHostIDs, host.ID) {
			filteredHosts = append(filteredHosts, host)
		}
	}

	return filteredHosts
}

func containsHostID(hostIDs []string, id string) bool {
	for _, hostID := range hostIDs {
		if hostID == id {
			return true
		}
	}

	return false
}
