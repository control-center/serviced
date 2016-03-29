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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

// Serviced is a PaaS runtime based on docker. The serviced package exposes the
// interfaces for the key parts of this runtime.
package node

import (
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/health"
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
		Ports       map[string][]domain.HostIPAndPort
	}
	SysInitPath    string
	ResolvConfPath string
	Volumes        map[string]string
	VolumesRW      map[string]bool
}

type ServiceInstanceRequest struct {
	ServiceID  string
	InstanceID int
}

type HealthCheckRequest struct {
	ServiceID  string
	InstanceID int
}

// The API for a service proxy.
type LoadBalancer interface {
	// SendLogMessage allows the proxy to send messages/logs to the master (to be displayed on the serviced master)
	SendLogMessage(serviceLogInfo ServiceLogInfo, _ *struct{}) error

	GetServiceEndpoints(serviceId string, endpoints *map[string][]applicationendpoint.ApplicationEndpoint) error

	// GetProxySnapshotQuiece blocks until there is a snapshot request
	GetProxySnapshotQuiece(serviceId string, snapshotId *string) error

	// AckProxySnapshotQuiece is called by clients when the snapshot command has
	// shown the service is quieced; the agent returns a response when the snapshot is complete
	AckProxySnapshotQuiece(snapshotId string, unused *interface{}) error

	// GetTenantId retrieves a service's tenant id
	GetTenantId(serviceId string, tenantId *string) error

	GetHealthCheck(req HealthCheckRequest, healthCheck *map[string]health.HealthCheck) error

	LogHealthCheck(result domain.HealthCheckResult, unused *int) error

	// ReportHealthStatus writes the health check status to the cache
	ReportHealthStatus(req dao.HealthStatusRequest, unused *int) error

	// ReportInstanceDead removes all health checks for the provided instance from the
	// cache.
	ReportInstanceDead(req dao.ServiceInstanceRequest, unused *int) error

	// GetService retrieves a service object with templates evaluated.
	GetService(serviceId string, response *service.Service) error

	// GetServiceInstance retrieves a service object with templates evaluated using a
	// given instance ID.
	GetServiceInstance(req ServiceInstanceRequest, response *service.Service) error

	// Ping waits for the specified time then returns the server time
	Ping(waitFor time.Duration, timestamp *time.Time) error
}
