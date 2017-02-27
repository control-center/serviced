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
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
)

// IsLoopbackError is an error type for IP addresses that are loopback.
type IsLoopbackError string

func (err IsLoopbackError) Error() string {
	return fmt.Sprintf("IP %s is a loopback address", string(err))
}

// InvalidIPAddress is an error for Invalid IPs
type InvalidIPAddress string

func (err InvalidIPAddress) Error() string {
	return fmt.Sprintf("IP %s is not a valid address", string(err))
}

// currentHost creates a Host object of the representing the host where this method is invoked. The passed in poolID is
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
		plog.WithError(err).WithFields(log.Fields{
			"ip": ip,
			"rpcport": rpcPort,
			"poolid": poolID,
		}).Debug("Unable to retrieve host ID")
		return nil, err
	}

	if ip != "" {
		if isLoopBack(ip) {
			return nil, IsLoopbackError(ip)
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
	host.ServiceD.Date = servicedversion.Date
	host.ServiceD.Buildtag = servicedversion.Buildtag
	host.ServiceD.Release = servicedversion.Release

	host.KernelVersion, host.KernelRelease, err = getOSKernelData()
	if err != nil {
		return nil, err
	}

	routes, err := utils.RouteCmd()
	if err != nil {
		plog.WithError(err).WithFields(log.Fields{
			"ip": ip,
			"rpcport": rpcPort,
			"poolid": poolID,
		}).Debug("Unable to get network routes")
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
func getIPResources(hostID string, hostIP string, natIP string, staticIPs ...string) ([]HostIPResource, error) {
	//make a map of all ipaddresses to interface
	ifacemap, err := getInterfaceMap()
	if err != nil {
		return nil, err
	}
	hostLogger := plog.WithFields(log.Fields{
		"hostid": hostID,
		"hostip": hostIP,
	})
	hostLogger.WithFields(log.Fields{
		"interfaces": ifacemap,
	}).Debug("Interfaces on this host")

	// Get a unique list of ips from staticIPs and hostIP.
	ips := func() []string {
		for _, ip := range staticIPs {
			if hostIP == ip {
				return staticIPs
			}
		}
		return append(staticIPs, hostIP)
	}()

	hostIPResources := make([]HostIPResource, len(ips))
	for i, ip := range ips {
		hostLogger.WithFields(log.Fields{
			"ip": ip,
		}).Debug("Checking IP")
		if iface, ok := ifacemap[ip]; ok {
			if isLoopBack(ip) {
				return nil, IsLoopbackError(ip)
			}
			hostIPResource := HostIPResource{
				HostID:        hostID,
				IPAddress:     ip,
				InterfaceName: iface.Name,
				MACAddress:    iface.HardwareAddr.String(),
			}
			hostIPResources[i] = hostIPResource
		} else {
			// The requested IP doesn't belong to any of the host interfaces.  If the NAT IP was set,
			// check to see if this matches.
			if len(natIP) > 0 && ip == natIP {
				// This is the NAT IP from the config.  Return a host IP resource for the NAT.
				hostIPResource := HostIPResource{
					HostID:        hostID,
					IPAddress:     ip,
					InterfaceName: "nat",
					MACAddress:    "00:00:00:00:00:00",
				}
				hostIPResources[i] = hostIPResource
			} else {
				// Unknown IP address.
				return nil, InvalidIPAddress(ip)
			}
		}
	}

	return hostIPResources, nil
}

// getInterfaceMap returns a map of ip string to net.Interface
func getInterfaceMap() (map[string]net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		plog.WithError(err).Debug("Unable to read network interfaces")
		return nil, err
	}
	//make a  of all ipaddresses to interface
	ips := make(map[string]net.Interface)
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			plog.WithField("interface", iface.Name).WithError(err).Debug("Unable to read interface addresses")
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

func isLoopBack(ip string) bool {
	if strings.HasPrefix(ip, "127") {
		return true
	}
	return false
}
