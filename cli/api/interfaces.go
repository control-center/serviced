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

package api

import (
	"io"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	template "github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/script"
	"github.com/control-center/serviced/volume"
)

// API is the intermediary between the command-line interface and the dao layer
type API interface {

	// Server
	StartServer() error
	ServicedHealthCheck(IServiceNames []string) ([]isvcs.IServiceHealthResult, error)

	// Hosts
	GetHosts() ([]host.Host, error)
	GetHost(string) (*host.Host, error)
	GetHostMap() (map[string]host.Host, error)
	AddHost(HostConfig) (*host.Host, []byte, error)
	RemoveHost(string) error
	GetHostMemory(string) (*metrics.MemoryUsageStats, error)
	SetHostMemory(HostUpdateConfig) error
	GetHostPublicKey(string) ([]byte, error)
	RegisterHost([]byte) error
	RegisterRemoteHost(*host.Host, []byte, bool) error
	WriteDelegateKey(string, []byte) error
	AuthenticateHost(string) (string, int64, error)
	ResetHostKey(string) ([]byte, error)
	GetHostWithAuthInfo(string) (*AuthHost, error)
	GetHostsWithAuthInfo() ([]AuthHost, error)

	// Pools
	GetResourcePools() ([]pool.ResourcePool, error)
	GetResourcePool(string) (*pool.ResourcePool, error)
	AddResourcePool(PoolConfig) (*pool.ResourcePool, error)
	RemoveResourcePool(string) error
	UpdateResourcePool(pool pool.ResourcePool) error
	GetPoolIPs(string) (*pool.PoolIPs, error)
	AddVirtualIP(pool.VirtualIP) error
	RemoveVirtualIP(pool.VirtualIP) error

	// Services
	GetAllServiceDetails() ([]service.ServiceDetails, error)
	GetServiceDetails(serviceID string) (*service.ServiceDetails, error)
	GetServiceStatus(string) (map[string]map[string]interface{}, error)
	GetService(string) (*service.Service, error)
	AddService(ServiceConfig) (*service.ServiceDetails, error)
	CloneService(string, string) (*service.ServiceDetails, error)
	RemoveService(string) error
	UpdateService(io.Reader) (*service.ServiceDetails, error)
	StartService(SchedulerConfig) (int, error)
	RestartService(SchedulerConfig) (int, error)
	StopService(SchedulerConfig) (int, error)
	AssignIP(IPConfig) error
	GetEndpoints(serviceID string, reportImports, reportExports, validate bool) ([]applicationendpoint.EndpointReport, error)
	ResolveServicePath(path string) ([]service.ServiceDetails, error)
	ClearEmergency(serviceID string) (int, error)

	// Shell
	StartShell(ShellConfig) error
	RunShell(ShellConfig, chan struct{}) (int, error)

	// Snapshots
	GetSnapshots() ([]dao.SnapshotInfo, error)
	GetSnapshotsByServiceID(string) ([]dao.SnapshotInfo, error)
	GetSnapshotByServiceIDAndTag(string, string) (string, error)
	AddSnapshot(SnapshotConfig) (string, error)
	RemoveSnapshot(string) error
	Rollback(string, bool) error
	TagSnapshot(string, string) error
	RemoveSnapshotTag(string, string) (string, error)

	// Templates
	GetServiceTemplates() ([]template.ServiceTemplate, error)
	GetServiceTemplate(string) (*template.ServiceTemplate, error)
	AddServiceTemplate(io.Reader) (*template.ServiceTemplate, error)
	RemoveServiceTemplate(string) error
	CompileServiceTemplate(CompileTemplateConfig) (*template.ServiceTemplate, error)
	DeployServiceTemplate(DeployTemplateConfig) ([]service.ServiceDetails, error)

	// Backup & Restore
	Backup(string, []string) (string, error)
	Restore(string) error

	// Docker
	ResetRegistry() error
	RegistrySync() error
	UpgradeRegistry(endpoint string, override bool) error
	DockerOverride(newImage string, oldImage string) error

	// Logs
	ExportLogs(config ExportLogsConfig) error

	// Metric
	PostMetric(metricName string, metricValue string) (string, error)

	// Scripts
	ScriptRun(fileName string, config *script.Config, stopChan chan struct{}) error
	ScriptParse(fileName string, config *script.Config) error

	// Volumes
	GetVolumeStatus() (*volume.Statuses, error)

	// Public endpoints
	AddPublicEndpointPort(serviceid, endpointName, portAddr string, usetls bool, protocol string, isEnabled, restart bool) (*servicedefinition.Port, error)
	RemovePublicEndpointPort(serviceid, endpointName, portAddr string) error
	EnablePublicEndpointPort(serviceid, endpointName, portAddr string, isEnabled bool) error
	AddPublicEndpointVHost(serviceid, endpointName, vhost string, isEnabled, restart bool) (*servicedefinition.VHost, error)
	RemovePublicEndpointVHost(serviceid, endpointName, vhost string) error
	EnablePublicEndpointVHost(serviceid, endpointName, vhost string, isEnabled bool) error
	GetAllPublicEndpoints() ([]service.PublicEndpoint, error)

	// Service Instances
	GetServiceInstances(serviceID string) ([]service.Instance, error)
	StopServiceInstance(serviceID string, instanceID int) error
	AttachServiceInstance(serviceID string, instanceID int, command string, args []string) error
	LogsForServiceInstance(serviceID string, instanceID int, command string, args []string) error
	SendDockerAction(serviceID string, instanceID int, action string, args []string) error

	// Debug Management
	DebugEnableMetrics() (string, error)
	DebugDisableMetrics() (string, error)
}
