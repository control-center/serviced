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

// Simple iptables controller based on Docker's
package iptables

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

type Action string

const (
	Add    Action = "-A"
	Delete Action = "-D"
)

var (
	ErrIptablesNotFound = errors.New("iptables not found")
	nat                 = []string{"-t", "nat"}
	supportsXlock       = false
)

type Chain struct {
	Name string
}

type Address struct {
	IP   string
	Port int
}

func init() {
	supportsXlock = exec.Command("iptables", "--wait", "-L", "-n").Run() == nil
}

func NewChain(name string) *Chain {
	return &Chain{
		Name: name,
	}
}

func NewAddress(ip string, port int) *Address {
	return &Address{IP: ip, Port: port}
}

func (c *Chain) Inject() error {
	if output, err := RunIptablesCommand(append(nat, "-N", c.Name)...); err != nil {
		return err
	} else if len(output) != 0 {
		return fmt.Errorf("Error creating new iptables chain: %s", output)
	}
	if err := c.Prerouting(Add, "-m", "addrtype", "--dst-type", "LOCAL"); err != nil {
		return fmt.Errorf("Failed to inject serviced in PREROUTING chain: %s", err)
	}
	return nil
}

func (c *Chain) Prerouting(action Action, args ...string) error {
	a := append(nat, fmt.Sprint(action), "PREROUTING")
	if len(args) > 0 {
		a = append(a, args...)
	}
	if output, err := RunIptablesCommand(append(a, "-j", c.Name)...); err != nil {
		return err
	} else if len(output) != 0 {
		return fmt.Errorf("Error iptables prerouting: %s", output)
	}
	return nil
}

func (c *Chain) Forward(action Action, proto string, dest, fwdto *Address) error {
	var daddr string
	if dest.IP == "" {
		daddr = "0/0"
	} else {
		daddr = dest.IP
	}
	if output, err := RunIptablesCommand(append(nat, fmt.Sprint(action), c.Name,
		"-p", proto,
		"-d", daddr,
		"--dport", strconv.Itoa(dest.Port),
		"-j", "DNAT",
		"--to-destination", net.JoinHostPort(fwdto.IP, strconv.Itoa(fwdto.Port)),
	)...); err != nil {
		return err
	} else if len(output) != 0 {
		return fmt.Errorf("Error iptables forward: %s", output)
	}

	fAction := action
	if fAction == Add {
		fAction = "-I"
	}
	if output, err := RunIptablesCommand(string(fAction), "FORWARD",
		"-p", proto,
		"-d", fwdto.IP,
		"--dport", strconv.Itoa(fwdto.Port),
		"-j", "ACCEPT"); err != nil {
		return err
	} else if len(output) != 0 {
		return fmt.Errorf("Error iptables forward: %s", output)
	}
	return nil
}

func (c *Chain) Remove() error {
	// Ignore errors - This could mean the chains were never set up
	c.Prerouting(Delete, "-m", "addrtype", "--dst-type", "LOCAL")
	c.Prerouting(Delete)

	RunIptablesCommand(append(nat, "-F", c.Name)...)
	RunIptablesCommand(append(nat, "-X", c.Name)...)

	return nil
}

func RunIptablesCommand(args ...string) ([]byte, error) {
	path, err := exec.LookPath("iptables")
	if err != nil {
		return nil, ErrIptablesNotFound
	}
	if supportsXlock {
		args = append([]string{"--wait"}, args...)
	}
	output, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("iptables failed: iptables %v: %s (%s)", strings.Join(args, " "), output, err)
	}
	// ignore iptables' message about xtables lock
	if strings.Contains(string(output), "waiting for it to exit") {
		output = []byte("")
	}
	return output, err
}