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

package container

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/validation"
	"github.com/zenoss/glog"
)

const defaultSubnet string = "10.3" // /16 subnet for virtual addresses

type vif struct {
	name     string
	ip       string
	hostname string
	tcpPorts map[string]string
	udpPorts map[string]string
}

// VIFRegistry holds state regarding virtual interfaces. It is meant to be
// created in the proxy to manage vifs in the running service container.
type VIFRegistry struct {
	sync.RWMutex
	subnet string
	vifs   map[string]*vif
}

// NewVIFRegistry initializes a new VIFRegistry.
func NewVIFRegistry() *VIFRegistry {
	return &VIFRegistry{subnet: defaultSubnet, vifs: make(map[string]*vif)}
}

// SetSubnet sets the subnet used for virtual addresses
func (reg *VIFRegistry) SetSubnet(subnet string) error {
	if err := validation.IsSubnet16(subnet); err != nil {
		return err
	}
	reg.subnet = subnet
	glog.Infof("vif subnet is: %s", reg.subnet)
	return nil
}

func (reg *VIFRegistry) nextIP() (string, error) {
	n := len(reg.vifs) + 2
	if n > (255 * 255) {
		return "", fmt.Errorf("unable to allocate IPs for %d interfaces", n)
	}
	o3 := (n / 255)
	o4 := (n - (o3 * 255))
	// ZEN-11478: made the subnet configurable
	return fmt.Sprintf("%s.%d.%d", reg.subnet, o3, o4), nil
}

// RegisterVirtualAddress takes care of the entire virtual address setup. It
// creates a virtual interface if one does not yet exist, allocates an IP
// address, assigns it to the virtual interface, adds an entry to /etc/hosts,
// and sets up the iptables rule to redirect traffic to the specified port.
func (reg *VIFRegistry) RegisterVirtualAddress(address, toport, protocol string) error {
	glog.Infof("RegisterVirtualAddress address:%s toport:%s protocol:%s", address, toport, protocol)
	reg.Lock()
	defer reg.Unlock()
	glog.V(2).Infof("RegisterVirtualAddress address:%s toport:%s protocol:%s  locked", address, toport, protocol)

	var (
		host, port string
		viface     *vif
		err        error
		ok         bool
		portmap    *map[string]string
	)
	if host, port, err = net.SplitHostPort(address); err != nil {
		return err
	}
	if viface, ok = reg.vifs[host]; !ok {
		// vif doesn't exist yet
		ip, err := reg.nextIP()
		if err != nil {
			return err
		}
		viface = &vif{
			hostname: host,
			ip:       ip,
			name:     "eth0-" + host,
			tcpPorts: make(map[string]string),
			udpPorts: make(map[string]string),
		}
		if err = viface.createCommand(); err != nil {
			return err
		}
		reg.vifs[host] = viface
	}
	switch strings.ToLower(protocol) {
	case "tcp":
		portmap = &viface.tcpPorts
	case "udp":
		portmap = &viface.udpPorts
	default:
		return fmt.Errorf("invalid protocol: %s", protocol)
	}

	glog.V(2).Infof("RegisterVirtualAddress portmap: %+v", *portmap)
	if _, ok := (*portmap)[toport]; !ok {
		// dest isn't there, let's DO IT!!!!!
		if err := viface.redirectCommand(port, toport, protocol); err != nil {
			return err
		}
		(*portmap)[toport] = port
	}
	return nil
}

func (viface *vif) createCommand() error {
	command := []string{
		"ip", "link", "add", "link", "eth0",
		"name", viface.name,
		"type", "veth",
		"peer", "name", viface.name + "-peer",
	}
	c := exec.Command(command[0], command[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stdout

	if err := c.Run(); err != nil {
		glog.Errorf("Adding virtual interface failed using cmd:%+v  error:%+v", command, err)
		return err
	}
	command = []string{
		"ip", "addr", "add", viface.ip + "/16", "dev", viface.name,
	}
	c = exec.Command(command[0], command[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stdout
	if err := c.Run(); err != nil {
		glog.Errorf("Adding IP to virtual interface failed using cmd:%+v  error:%+v", command, err)
		return err
	}
	command = []string{
		"ip", "link", "set", viface.name, "up",
	}
	c = exec.Command(command[0], command[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stdout
	if err := c.Run(); err != nil {
		glog.Errorf("Bringing interface %s up failed using cmd:%+v  error:%+v", viface.name, command, err)
		return err
	}
	return nil
}

func (viface *vif) redirectCommand(from, to, protocol string) error {
	glog.Infof("Trying to set up redirect %s:%s->:%s %s", viface.hostname, from, to, protocol)
	for _, chain := range []string{"OUTPUT", "PREROUTING"} {
		command := []string{
			"iptables",
			"-t", "nat",
			"-A", chain,
			"-d", viface.ip,
			"-p", protocol,
			"--dport", from,
			"-j", "REDIRECT",
			"--to-ports", to,
		}
		c := exec.Command(command[0], command[1:]...)

		c.Stdout = os.Stdout
		c.Stderr = os.Stdout
		if err := c.Run(); err != nil {
			glog.Errorf("Unable to set up redirect %s:%s->:%s %s command:%+v", viface.hostname, from, to, protocol, command)
			return err
		}
	}

	glog.Infof("AddToEtcHosts(%s, %s)", viface.hostname, viface.ip)
	err := node.AddToEtcHosts(viface.hostname, viface.ip)
	if err != nil {
		glog.Errorf("Unable to add %s %s to /etc/hosts", viface.ip, viface.hostname)
		return err
	}
	return nil
}
