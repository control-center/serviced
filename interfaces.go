// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

// Serviced is a PaaS runtime based on docker. The serviced package exposes the
// interfaces for the key parts of this runtime.
package serviced

import (
	"github.com/zenoss/serviced/dao"

	"time"
)

// Network protocol type.
type ProtocolType string

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
		Ports       map[string][]dao.HostIpAndPort
	}
	SysInitPath    string
	ResolvConfPath string
	Volumes        map[string]string
	VolumesRW      map[string]bool
}

// The API for a service proxy.
type LoadBalancer interface {
	GetServiceEndpoints(serviceId string, endpoints *map[string][]*dao.ApplicationEndpoint) error

	// GetProxySnapshotQuiece blocks until there is a snapshot request
	GetProxySnapshotQuiece(serviceId string, snapshotId *string) error

	// AckProxySnapshotQuiece is called by clients when the snapshot command has
	// shown the service is quieced; the agent returns a response when the snapshot is complete
	AckProxySnapshotQuiece(snapshotId string, unused *interface{}) error
}

// The Agent interface is the API for a serviced agent.  Get the host information from an agent.
// The ips argument is a list of IPs whose information should be included with the Host struct. If any of the IPs are
// not valid on the host and error will be returned.  If the ips argument is empty a default IP will be chosen and returned
// with thoe Host info
type Agent interface {
	GetInfo(ips []string, host *dao.Host) error
}
