// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"

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
	memory, err := serviced.GetMemorySize()
	if err != nil {
		return nil, err
	}
	host = New()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	host.Name = hostname
	hostidStr, err := serviced.HostID()
	if err != nil {
		return nil, err
	}

	if ip != "" {
		if !validIP(ip){
			return nil, fmt.Errorf("Requested IP %v is not available on host", ip)
		}
		host.IpAddr = ip
	}else {
		host.IpAddr, err = serviced.GetIPAddress()
		if err != nil {
			return host, err
		}
	}



	host.Id = hostidStr
	host.Cores = cpus
	host.Memory = memory

	routes, err := serviced.RouteCmd()
	if err != nil {
		return nil, err
	}
	for _, route := range routes {
		if route.Iface == "docker0" {
			host.PrivateNetwork = route.Destination + "/" + route.Genmask
			break
		}
	}
	host.PoolId = poolID
	return host, err
}

// getIPResources does the actual work of determining the IPs on the host. Parameters are the IPs to filter on
func getIPResources(hostId string, ipaddress ...string) ([]HostIPResource, error) {

	//make a map of all ipaddresses to interface
	ips, err := getInterfaceMap()
	if err != nil {
		return []HostIPResource{}, err
	}

	glog.V(4).Infof("Interfaces on this host %v", ips)

	hostIPResources := make([]HostIPResource, 0)

	validate := func(iface net.Interface, ip string) error {
		if (uint(iface.Flags) & (1<<uint(net.FlagLoopback))) == 0 {
			return fmt.Errorf("Loopback address %v cannot be used to register a host", ip)
		}
		return nil
	}

	for _, ipaddr := range ipaddress {
		normalIP := strings.Trim(strings.ToLower(ipaddr), " ")
		iface, found := ips[normalIP]
		if !found {
			return []HostIPResource{}, fmt.Errorf("IP address %v not valid for this host", ipaddr)
		}
		err = validate(iface, normalIP)
		if err != nil {
			return []HostIPResource{}, err
		}
		hostIp := HostIPResource{}
		hostIp.HostId = hostId
		hostIp.IPAddress = ipaddr
		hostIp.InterfaceName = iface.Name
		hostIPResources = append(hostIPResources, hostIp)
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

func validIP(ip string) bool {
	interfaces, err := getInterfaceMap()
	if err != nil {
		glog.Error("Problem reading interfaces: ", err)
		return false
	}
	normalIP := normalizeIP(ip)
	_, found := interfaces[normalIP]
	return found
}

