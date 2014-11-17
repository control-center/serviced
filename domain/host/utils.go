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

package host

import (
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
	"github.com/kr/pretty"
	"github.com/zenoss/glog"

	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// currentHost creates a Host object of the reprsenting the host where this method is invoked. The passed in poolID is
// used as the resource pool in the result.
func currentHost(ip string, rpcPort int, poolID string) (host *Host, err error) {
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
	host.RPCPort = rpcPort

	host.ID = hostidStr
	host.Cores = cpus
	host.Memory = memory

	// get embedded host information
	host.ServiceD.Version = servicedversion.Version
	host.ServiceD.Gitbranch = servicedversion.Gitbranch
	host.ServiceD.Gitcommit = servicedversion.Gitcommit
	host.ServiceD.Giturl = servicedversion.Giturl
	host.ServiceD.Date = servicedversion.Date
	host.ServiceD.Buildtag = servicedversion.Buildtag

	host.KernelVersion, host.KernelRelease, err = getOSKernelData()
	if err != nil {
		return nil, err
	}

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

func getOSKernelData() (string, string, error) {
	output, err := exec.Command("uname", "-r", "-v").Output()
	if err != nil {
		return "There was an error retrieving kernel data", "There was an error retrieving kernel data", err
	}

	kernelVersion, kernelRelease := parseOSKernelData(string(output))
	return kernelVersion, kernelRelease, err
}

func parseOSKernelData(data string) (string, string) {
	parts := strings.Split(data, " ")
	return parts[1], parts[0]
}

// getIPResources does the actual work of determining the IPs on the host. Parameters are the IPs to filter on
func getIPResources(hostID string, ipaddress ...string) ([]HostIPResource, error) {

	//make a map of all ipaddresses to interface
	ips, err := getInterfaceMap()
	if err != nil {
		return []HostIPResource{}, err
	}
	keys := make([]string, len(ips))
	i := 0
	for key, _ := range ips {
		keys[i] = key
		i += 1
	}
	glog.V(4).Infof("localIPs: %v", keys)

	glog.V(4).Infof("Interfaces on this host %v", ips)

	hostIPResources := make([]HostIPResource, 0)

	for _, ipaddr := range ipaddress {
		glog.Infof("looking for '%s'", ipaddr)
		iface, found := ips[ipaddr]
		if !found {
			return []HostIPResource{}, fmt.Errorf("IP address %v not valid for this host", ipaddr)
		}
		if isLoopBack(ipaddr) {
			return []HostIPResource{}, fmt.Errorf("loopback address %v cannot be used as an IP Resource", ipaddr)
		}
		hostIP := HostIPResource{}
		hostIP.HostID = hostID
		hostIP.IPAddress = ipaddr
		hostIP.InterfaceName = iface.Name
		hostIP.MACAddress = iface.HardwareAddr.String()
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
	glog.V(5).Infof("looking for %s in %#", ip, pretty.Formatter(interfaces))
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
