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

package node

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
)

var (
	re = regexp.MustCompile(`\d+:\s+(\S+)\s+inet6?\s+(\S+).*?(\S+)?\\`)
)

// VIP is the interface for managing virtual ips
type VIP interface {
	GetAll() []IP
	Find(ipprefix string) *IP
	Release(ipaddr, device string) error
	Bind(ipaddr, device string) error
}

// IP describes an ip binding
type IP struct {
	Addr   string
	Device string
	Label  string
}

// Matches returns true if the address and device match
func (ip *IP) Matches(ipaddr, device string) bool {
	return ip.Addr == ipaddr && ip.Device == device
}

// VirtualIPManager manages virtual ip bindings
type VirtualIPManager struct {
	label string
	mu    *sync.Mutex
}

// NewVirtualIPManager instantiates an instance of the VirtualIPManager
func NewVirtualIPManager(label string) *VirtualIPManager {
	return &VirtualIPManager{
		label: label,
		mu:    &sync.Mutex{},
	}
}

// GetAll returns all the bound virtual ips
func (v *VirtualIPManager) GetAll() (ips []IP) {
	cmd := exec.Command("ip", "-o", "a", "show", "label", fmt.Sprintf("*:%s*", v.label))
	stdout, _ := cmd.StdoutPipe()

	cmd.Start()
	defer cmd.Wait()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if ip := v.loadIP(scanner.Bytes()); ip != nil {
			ips = append(ips, *ip)
		}
	}

	return
}

// Find returns the matching binding for the given ip address
func (v *VirtualIPManager) Find(ipprefix string) *IP {
	cmd := exec.Command("ip", "-o", "a", "show", "label", fmt.Sprintf("*:%s*", v.label), "to", ipprefix)
	output, _ := cmd.CombinedOutput()
	return v.loadIP(output)
}

// Release releases the virtual ip binding
func (v *VirtualIPManager) Release(ipaddr, device string) error {
	logger := plog.WithFields(logrus.Fields{
		"ipaddress": ipaddr,
		"device":    device,
	})

	cmd := exec.Command("ip", "a", "del", ipaddr, "dev", device)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"command": cmd,
			"output":  string(output),
		}).Debug("Could not remove virtual ip")
		return err
	}

	logger.Debug("Removed virtual ip")
	return nil
}

// Bind binds a virtual ip to a device
func (v *VirtualIPManager) Bind(ipaddr, device string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	logger := plog.WithFields(logrus.Fields{
		"ipaddress": ipaddr,
		"device":    device,
	})

	cmd := exec.Command("ip", "a", "add", ipaddr, "dev", device, "label", v.nextLabel(device))
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"command": cmd,
			"output":  string(output),
		}).Debug("Could not add virtual ip")
		return err
	}

	logger.Debug("Added virtual ip")
	return nil
}

func (v *VirtualIPManager) loadIP(b []byte) *IP {
	if !re.Match(b) {
		return nil
	}

	fields := re.FindSubmatch(b)
	return &IP{
		Addr:   string(fields[2]),
		Device: string(fields[1]),
		Label:  string(fields[3]),
	}
}

func (v *VirtualIPManager) nextLabel(device string) string {
	ips := v.GetAll()
	var ids []int
	for _, ip := range ips {
		indexStr := strings.TrimPrefix(ip.Label, fmt.Sprintf("%s:%s", ip.Device, v.label))
		val, err := strconv.Atoi(indexStr)
		if err == nil {
			ids = append(ids, val)
		}
	}
	sort.Ints(ids)
	nextIndex := 0
	for i, id := range ids {
		if i == id {
			nextIndex++
		} else {
			break
		}
	}

	return fmt.Sprintf("%s:%s%d", device, v.label, nextIndex)
}

// BindIP creates a virtual ip if it doesn't already exist.
func (a *HostAgent) BindIP(ipprefix, netmask, iface string) error {
	logger := plog.WithFields(logrus.Fields{
		"ipaddress": ipprefix,
		"netmask":   netmask,
		"interface": iface,
	})

	// calculate the cidr bytes
	cidr, _ := net.IPMask(net.ParseIP(netmask).To4()).Size()
	ip := fmt.Sprintf("%s/%d", ipprefix, cidr)

	if vip := a.vip.Find(ipprefix); vip != nil {
		// return if the ips match
		if !vip.Matches(ip, iface) {
			if err := a.vip.Release(vip.Addr, vip.Device); err != nil {
				logger.WithError(err).Error("Could not release virtual ip for update")
				return err
			}
		} else {
			return nil
		}
	}

	// check the interface
	if _, err := net.InterfaceByName(iface); err != nil {
		logger.WithError(err).Error("Could not look up interface to bind virtual ip")
		return err
	}

	// bind the virtual ip
	if err := a.vip.Bind(ip, iface); err != nil {
		logger.WithError(err).Error("Could not bind virtual ip")
		return err
	}

	return nil
}

// ReleaseIP releases a virtual ip if hasn't yet been released.
func (a *HostAgent) ReleaseIP(ipprefix string) error {
	logger := plog.WithField("ipaddress", ipprefix)
	if vip := a.vip.Find(ipprefix); vip != nil {
		if err := a.vip.Release(vip.Addr, vip.Device); err != nil {
			logger.WithError(err).Error("Could not release virtual ip")
			return err
		}
	}

	return nil
}
