package api

import (
	"io"

	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
)

const ()

var ()

// ServiceConfig is the deserialized object from the command-line
type ServiceConfig struct {
	Name        string
	PoolID      string
	ImageID     string
	Command     string
	LocalPorts  PortOpts
	RemotePorts PortOpts
}

type PortOpts map[string]service.ServiceEndpoint

func (p *PortOpts) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return fmt.Errorf("malformed port specification (%s)", value)
	}
	protocol := parts[0]
	if protocol != "tcp" || protocol != "udp" {
		return fmt.Errorf("unsupported protocol for port specification: %s (%s)", protocol, value)
	}
	portNum, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid port number: %s (%s)", parts[1], value)
	}
	portName = parts[2]
	if portName == "" {
		return fmt.Errorf("endpoint name cannot be empty (%s)", value)
	}
	port := fmt.Sprintf("%s:%s", protocol, portNum)
	(*opts)[port] = dao.ServiceEndpoint{Protocol: protocol, PortNumber: portNum, Application: portName}
	return nil
}

func (p *PortOpts) String() string {
	return fmt.Sprintf("%s", *p)
}

func (p *PortOpts) Value() interface{} {
	return *p
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
