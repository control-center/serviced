package api

import (
	"io"

	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/template"
)

// API is the intermediary between the command-line interface and the dao layer
type API interface {

	// Server
	StartServer(Options)
	StartProxy(ProxyOptions)

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
	ListTemplates() ([]template.Template, error)
	AddTemplate(io.Reader) (*template.Template, error)
	RemoveTemplate(string) error
	CompileTemplate(string) (io.Reader, error)
	DeployTemplate(string) error
}