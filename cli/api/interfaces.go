package api

import (
	"io"

	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/domain/service"
	template "github.com/zenoss/serviced/domain/servicetemplate"
	"github.com/zenoss/serviced/facade"
)

// API is the intermediary between the command-line interface and the dao layer
type API interface {

	// Server
	StartServer() error
	StartProxy(ControllerOptions) error

	// Hosts
	GetHosts() ([]*host.Host, error)
	GetHost(string) (*host.Host, error)
	AddHost(HostConfig) (*host.Host, error)
	RemoveHost(string) error

	// Pools
	GetResourcePools() ([]*pool.ResourcePool, error)
	GetResourcePool(string) (*pool.ResourcePool, error)
	AddResourcePool(PoolConfig) (*pool.ResourcePool, error)
	RemoveResourcePool(string) error
	GetPoolIPs(string) (*facade.PoolIPs, error)
	AddVirtualIP(pool.VirtualIP) error
	RemoveVirtualIP(pool.VirtualIP) error

	// Services
	GetServices() ([]*service.Service, error)
	GetService(string) (*service.Service, error)
	GetServicesByName(string) ([]*service.Service, error)
	AddService(ServiceConfig) (*service.Service, error)
	RemoveService(RemoveServiceConfig) error
	UpdateService(io.Reader) (*service.Service, error)
	StartService(string) error
	StopService(string) error
	AssignIP(IPConfig) error

	// RunningServices (ServiceStates)
	GetRunningServices() ([]*dao.RunningService, error)
	Attach(AttachConfig) error
	Action(AttachConfig) error

	// Shell
	StartShell(ShellConfig) error
	RunShell(ShellConfig) error

	// Snapshots
	GetSnapshots() ([]string, error)
	GetSnapshotsByServiceID(string) ([]string, error)
	AddSnapshot(string) (string, error)
	RemoveSnapshot(string) error
	Commit(string) (string, error)
	Rollback(string) error

	// Templates
	GetServiceTemplates() ([]*template.ServiceTemplate, error)
	GetServiceTemplate(string) (*template.ServiceTemplate, error)
	AddServiceTemplate(io.Reader) (*template.ServiceTemplate, error)
	RemoveServiceTemplate(string) error
	CompileServiceTemplate(CompileTemplateConfig) (*template.ServiceTemplate, error)
	DeployServiceTemplate(DeployTemplateConfig) (*service.Service, error)

	// Backup & Restore
	Backup(string) (string, error)
	Restore(string) error

	// Docker
	Squash(imageName, downToLayer, newName, tempDir string) (string, error)

	// Logs
	ExportLogs(serviceIds []string, to, from, outfile string) error
}
