package api

import (
	"io"

	host "github.com/zenoss/serviced/dao"
	pool "github.com/zenoss/serviced/dao"
	service "github.com/zenoss/serviced/dao"
	template "github.com/zenoss/serviced/dao"
)

// API is the intermediary between the command-line interface and the dao layer
type API interface {

	// Server
	StartServer()
	StartProxy(ProxyConfig) error

	// Hosts
	ListHosts() ([]host.Host, error)
	GetHost(string) (*host.Host, error)
	AddHost(HostConfig) (*host.Host, error)
	RemoveHost(string) error

	// Pools
	ListPools() ([]pool.ResourcePool, error)
	GetPool(string) (*pool.ResourcePool, error)
	AddPool(PoolConfig) (*pool.ResourcePool, error)
	RemovePool(string) error
	ListPoolIPs(string) ([]host.HostIPResource, error)

	// Services
	ListServices() ([]service.Service, error)
	GetService(string) (*service.Service, error)
	AddService(ServiceConfig) (*service.Service, error)
	RemoveService(string) error
	UpdateService(io.Reader) (*service.Service, error)
	StartService(string) (*host.Host, error)
	StopService(string) (*host.Host, error)
	AssignIP(IPConfig) (*host.HostIPResource, error)

	// Shell
	ListCommands(string) ([]string, error)
	StartShell(ShellConfig) error

	// Snapshots
	ListSnapshots() ([]string, error)
	ListSnapshotsByServiceID(string) ([]string, error)
	AddSnapshot(string) (string, error)
	RemoveSnapshot(string) error
	Commit(string) (string, error)
	Rollback(string) error

	// Templates
	ListTemplates() ([]template.ServiceTemplate, error)
	GetTemplate(string) (*template.ServiceTemplate, error)
	AddTemplate(io.Reader) (*template.ServiceTemplate, error)
	RemoveTemplate(string) error
	CompileTemplate(CompileTemplateConfig) (*template.ServiceTemplate, error)
	DeployTemplate(DeployTemplateConfig) (*service.Service, error)
}