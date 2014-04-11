// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/utils"

	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
)

// currentHost creates a Host object of the reprsenting the host where this method is invoked. The passed in poolID is
// used as the resource pool in the result.
func currentHost(ip string, poolID string) (host *Host, err error) {
	cpus := runtime.NumCPU()
	memory, err := utils.GetMemorySize()
	if err != nil {
		return nil, err
	}
	host = New()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	host.Name = hostname
	hostidStr, err := utils.HostID()
	if err != nil {
		return nil, err
	}

	if ip != "" {
		if !ipExists(ip) {
			return nil, fmt.Errorf("requested IP %v is not available on host", ip)
		}
		if isLoopBack(ip) {
			return nil, fmt.Errorf("loopback address %s cannot be used to register a host", ip)
		}

		host.IPAddr = ip
	} else {
		host.IPAddr, err = utils.GetIPAddress()
		if err != nil {
			return host, err
		}
	}

	host.ID = hostidStr
	host.Cores = cpus
	host.Memory = memory

	routes, err := utils.RouteCmd()
	if err != nil {
		return nil, err
	}
	for _, route := range routes {
		if route.Iface == "docker0" {
			host.PrivateNetwork = route.Destination + "/" + route.Genmask
			break
		}
	}
	host.PoolID = poolID
	return host, err
}

// getIPResources does the actual work of determining the IPs on the host. Parameters are the IPs to filter on
func getIPResources(hostID string, ipaddress ...string) ([]HostIPResource, error) {

	//make a map of all ipaddresses to interface
	ips, err := getInterfaceMap()
	if err != nil {
		return []HostIPResource{}, err
	}

	glog.V(4).Infof("Interfaces on this host %v", ips)

	hostIPResources := make([]HostIPResource, 0)

	for _, ipaddr := range ipaddress {
		normalIP := strings.Trim(strings.ToLower(ipaddr), " ")
		iface, found := ips[normalIP]
		if !found {
			return []HostIPResource{}, fmt.Errorf("IP address %v not valid for this host", ipaddr)
		}
		if isLoopBack(normalIP) {
			return []HostIPResource{}, fmt.Errorf("loopback address %v cannot be used as an IP Resource", ipaddr)
		}
		hostIP := HostIPResource{}
		hostIP.HostID = hostID
		hostIP.IPAddress = ipaddr
		hostIP.InterfaceName = iface.Name
		hostIPResources = append(hostIPResources, hostIP)
	}
	return hostIPResources, nil
}

// getInterfaceMap returns a map of ip string to net.Interface
func getInterfaceMap() (map[string]net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		glog.Error("Problem reading interfaces: ", err)
		return nil, err
	}
	//make a  of all ipaddresses to interface
	ips := make(map[string]net.Interface)
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			glog.Error("Problem reading interfaces: ", err)
			return nil, err
		}
		for _, ip := range addrs {
			normalIP := strings.SplitN(ip.String(), "/", 2)[0]
			normalIP = strings.Trim(strings.ToLower(normalIP), " ")

			ips[normalIP] = iface
		}
	}
	return ips, nil
}

func normalizeIP(ip string) string {
	return strings.Trim(strings.ToLower(ip), " ")
}

func ipExists(ip string) bool {
	interfaces, err := getInterfaceMap()
	if err != nil {
		glog.Error("Problem reading interfaces: ", err)
		return false
	}
	normalIP := normalizeIP(ip)
	_, found := interfaces[normalIP]
	return found
}

func isLoopBack(ip string) bool {
	if strings.HasPrefix(ip, "127") {
		return true
	}
	return false
}
