// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"errors"
	"strings"
	"time"
)

//Host that runs the control plane agent.
type Host struct {
	ID             string // Unique identifier, default to hostid
	Name           string // A label for the host, eg hostname, role
	PoolID         string // Pool that the Host belongs to
	IPAddr         string // The IP address the host can be reached at from a serviced master
	Cores          int    // Number of cores available to serviced
	Memory         uint64 // Amount of RAM (bytes) available to serviced
	PrivateNetwork string // The private network where containers run, eg 172.16.42.0/24
	CreatedAt      time.Time
	UpdatedAt      time.Time
	IPs            []HostIPResource // The static IP resources available on the hose
}

//HostIPResource contains information about a specific IP available as a resource
type HostIPResource struct {
	HostID        string
	IPAddress     string
	InterfaceName string
}

// New creates a new empty host
func New() *Host {
	host := &Host{}
	return host
}

// Build creates a Host type from the current host machine, filling out fields using the current machines attributes.
// The ip param is routable IP used to connecto to the Host, if empty a IP from the available IPs will be used. The poolid
// param is the pool the host should belong to.  Optional list of IP address strings to set as available IP
// resources. If any IP is not a valid IP on the machine return error.
func Build(ip string, poolid string, ipAddrs ...string) (*Host, error) {
	if strings.TrimSpace(poolid) == "" {
		return nil, errors.New("empty poolid not allowed")
	}
	host, err := currentHost(ip, poolid)
	if err != nil {
		return nil, err
	}

	if len(ipAddrs) == 0 {
		// use the default IP of the host if specific IPs have not been requested
		ipAddrs = append(ipAddrs, host.IPAddr)
	}
	hostIPs, err := getIPResources(host.ID, ipAddrs...)
	if err != nil {
		return nil, err
	}
	host.IPs = hostIPs
	*host = *host
	return host, nil
}
