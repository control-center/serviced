/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

// Serviced is a PaaS runtime based on docker. The serviced package exposes the
// interfaces for the key parts of this runtime.
package serviced

import (
	"time"
  "github.com/zenoss/serviced/dao"
)

// Network protocol type.
type ProtocolType string

const (
	TCP string = "tcp"
	UDP string = "udp"
)

// A user defined string that describes an exposed application endpoint.
type ApplicationType string

// The state of a container as reported by Docker.
type ContainerState struct {
	ID      string
	Created time.Time
	Path    string
	Args    []string
	Config  struct {
		Hostname        string
		User            string
		Memory          uint64
		MemorySwap      uint64
		CpuShares       int
		AttachStdin     bool
		AttachStdout    bool
		AttachStderr    bool
		PortSpecs       []string
		Tty             bool
		OpenStdin       bool
		StdinOnce       bool
		Env             []string
		Cmd             []string
		Dns             []string
		Image           string
		Volumes         map[string]struct{}
		VolumesFrom     string
		WorkingDir      string
		Entrypoint      []string
		NetworkDisabled bool
		Privileged      bool
	}
	State struct {
		Running   bool
		Pid       int
		ExitCode  int
		StartedAt string
		Ghost     bool
	}
	Image           string
	NetworkSettings struct {
		IPAddress   string
		IPPrefixLen int
		Gateway     string
		Bridge      string
		PortMapping map[string]map[string]string
	}
	SysInitPath    string
	ResolvConfPath string
	Volumes        map[string]string
	VolumesRW      map[string]bool
}

// An exposed service endpoint
type ApplicationEndpoint struct {
	ServiceId     string
	ContainerPort uint16
	HostPort      uint16
	HostIp        string
	ContainerIp   string
	Protocol      string
}

// The API for a service proxy.
type LoadBalancer interface {
	GetServiceEndpoints(serviceId string, endpoints *map[string][]*ApplicationEndpoint) error
}

// The Agent interface is the API for a serviced agent.
type Agent interface {
	GetInfo(unused int, host *dao.Host) error
}
