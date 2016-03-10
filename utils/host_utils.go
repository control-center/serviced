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

package utils

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)


var Platform = determinePlatform()

const (
   Rhel = iota
   Debian
   Darwin
)


// HostID retrieves the system's unique id, on linux this maps
// to /usr/bin/hostid.
func HostID() (hostid string, err error) {
	return getHostID()
}

// GetIPAddress attempts to find the IP address to the default outbound interface.
func GetIPAddress() (ip string, err error) {
	ip, err = getIPAddrFromOutGoingConnection()
	switch {
	case err != nil:
		return "", err
	case err == nil && strings.HasPrefix(ip, "127"):
		return "", fmt.Errorf("unable to identify local ip address")
	default:
		return ip, err
	}
}

// GetIPAddresses returns a list of all IPv4 interface addresses
func GetIPv4Addresses() (ips []string, err error) {
	ips = []string{}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("unable to use InterfaceAddrs to find local ipv4 addresses: %v", err)
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && nil != ipnet.IP.To4() {
			ips = append(ips, ipnet.IP.String())
		}
	}

	if len(ips) == 0 {
		return ips, fmt.Errorf("unable to identify local ip address")
	}

	return ips, nil
}

// GetMemorySize attempts to get the size of the installed RAM.
func GetMemorySize() (size uint64, err error) {
	return getMemorySize()
}

// RouteEntry represents a entry from the route command
type RouteEntry struct {
	Destination string
	Gateway     string
	Genmask     string
	Flags       string
	Metric      int
	Ref         int
	Use         int
	Iface       string
}

// RouteCmd wrapper around the route command
func RouteCmd() (routes []RouteEntry, err error) {
	return getRoutes()
}

// getIPAddrFromHostname returns the ip address associated with hostname -i.
func getIPAddrFromHostname() (ip string, err error) {
	output, err := exec.Command("hostname", "-i").Output()
	if err != nil {
		return ip, err
	}
	return strings.TrimSpace(string(output)), err
}

// getIPAddrFromOutGoingConnection get the IP bound to the interface which
// handles the default route traffic.
func getIPAddrFromOutGoingConnection() (ip string, err error) {
	addr, err := net.ResolveUDPAddr("udp4", "8.8.8.8:53")
	if err != nil {
		return "", err
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return "", err
	}

	localAddr := conn.LocalAddr()
	parts := strings.Split(localAddr.String(), ":")
	return parts[0], nil
}
