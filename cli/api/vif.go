package api

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

type vif struct {
	name     string
	ip       string
	hostname string
	tcpPorts map[string]string
	udpPorts map[string]string
}

type VIFRegistry struct {
	vifs map[string]*vif
}

func NewVIFRegistry() *VIFRegistry {
	return &VIFRegistry{make(map[string]*vif)}
}

func (reg *VIFRegistry) nextIP() string {
	return fmt.Sprintf("10.3.0.%d", len(reg.vifs)+2)
}

func (reg *VIFRegistry) RegisterVirtualAddress(address, toport, protocol string) error {
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
		viface = &vif{
			hostname: host,
			ip:       reg.nextIP(),
			name:     "eth0:" + host,
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
		return fmt.Errorf("Invalid protocol: %s", protocol)
	}
	if _, ok := (*portmap)[toport]; !ok {
		// dest isn't there, let's DO IT!!!!!
		if err := viface.redirectCommand(port, toport, protocol); err != nil {
			return err
		}
		(*portmap)[toport] = port
	}
	return nil
}

// TODO: Replace with ip instead of ifconfig
func (viface *vif) createCommand() error {
	command := []string{
		"ifconfig",
		viface.name,
		viface.ip,
		"netmask",
		"255.255.255.0",
		"up",
	}
	if err := exec.Command(command[0], command[1:]...).Run(); err != nil {
		return err
	}
	return nil
}

func (viface *vif) redirectCommand(from, to, protocol string) error {
	// TODO: REMOVE. This is for demo.
	cmd := []string{"apt-get", "-y", "install", "iptables"}
	exec.Command(cmd[0], cmd[1:]...).Run()

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
		if err := exec.Command(command[0], command[1:]...).Run(); err != nil {
			return err
		}
	}
	return nil
}
