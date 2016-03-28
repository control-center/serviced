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
	"github.com/control-center/serviced/domain/servicestate"
	template "github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/script"
	"github.com/control-center/serviced/volume"
)

// API is the intermediary between the command-line interface and the dao layer
type API interface {

	// Server
	StartServer() error
	ServicedHealthCheck(IServiceNames []string) ([]dao.IServiceHealthResult, error)

	// Hosts
	GetHosts() ([]host.Host, error)
	GetHost(string) (*host.Host, error)
	GetHostMap() (map[string]host.Host, error)
	AddHost(HostConfig) (*host.Host, error)
	RemoveHost(string) error
	GetHostMemory(string) (*metrics.MemoryUsageStats, error)
	SetHostMemory(HostUpdateConfig) error

	// Pools
	GetResourcePools() ([]pool.ResourcePool, error)
	GetResourcePool(string) (*pool.ResourcePool, error)
	AddResourcePool(PoolConfig) (*pool.ResourcePool, error)
	RemoveResourcePool(string) error
	GetPoolIPs(string) (*pool.PoolIPs, error)
	AddVirtualIP(pool.VirtualIP) error
	RemoveVirtualIP(pool.VirtualIP) error

	// Services
	GetServices() ([]service.Service, error)
	GetServiceStates(string) ([]servicestate.ServiceState, error)
	GetServiceStatus(string) (map[string]map[string]interface{}, error)
	GetService(string) (*service.Service, error)
	GetServicesByName(string) ([]service.Service, error)
	AddService(ServiceConfig) (*service.Service, error)
	CloneService(string, string) (*service.Service, error)
	RemoveService(string) error
	UpdateService(io.Reader) (*service.Service, error)
	StartService(SchedulerConfig) (int, error)
	RestartService(SchedulerConfig) (int, error)
	StopService(SchedulerConfig) (int, error)
	AssignIP(IPConfig) error
	GetEndpoints(serviceID string, reportImports, reportExports, validate bool) ([]applicationendpoint.EndpointReport, error)

	// RunningServices (ServiceStates)
	GetRunningServices() ([]dao.RunningService, error)
	StopRunningService(string, string) error
	Attach(AttachConfig) error
	Action(AttachConfig) error

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
	DeployServiceTemplate(DeployTemplateConfig) ([]service.Service, error)

	// Backup & Restore
	Backup(string) (string, error)
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
}
