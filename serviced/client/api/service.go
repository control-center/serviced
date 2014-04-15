package api

import (
	"io"

	host "github.com/zenoss/serviced/dao"
	service "github.com/zenoss/serviced/dao"
)

const ()

var ()

// ServiceConfig is the deserialized object from the command-line
type ServiceConfig struct {
	Name        string
	PoolID      string
	ImageID     string
	Command     string
	LocalPorts  PortMap
	RemotePorts PortMap
}

// IPConfig is the deserialized object from the command-line
type IPConfig struct {
	ServiceID  string
	IPAddress  string
	AutoAssign bool
}

// ListServices lists all of the available services
func (a *api) ListServices() ([]service.Service, error) {
	return nil, nil
}

// GetService gets the service definition identified by its service ID
func (a *api) GetService(id string) (*service.Service, error) {
	return nil, nil
}

// AddService adds a new service
func (a *api) AddService(config ServiceConfig) (*service.Service, error) {
	return nil, nil
}

// RemoveService removes an existing service
func (a *api) RemoveService(id string) error {
	return nil
}

// UpdateService updates an existing service
func (a *api) UpdateService(reader io.Reader) (*service.Service, error) {
	return nil, nil
}

// StartService starts a service
func (a *api) StartService(id string) (*host.Host, error) {
	return nil, nil
}

// StopService stops a service
func (a *api) StopService(id string) (*host.Host, error) {
	return nil, nil
}

// AssignIP assigns an IP address to a service
func (a *api) AssignIP(config IPConfig) (*host.HostIPResource, error) {
	return nil, nil
}
