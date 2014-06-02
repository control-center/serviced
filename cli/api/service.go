package api

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/addressassignment"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
)

const ()

var ()

// ServiceConfig is the deserialized object from the command-line
type ServiceConfig struct {
	Name        string
	PoolID      string
	ImageID     string
	Command     string
	LocalPorts  *PortMap
	RemotePorts *PortMap
}

// RemoveServiceConfig is the deserialized object from the command-line
type RemoveServiceConfig struct {
	ServiceID       string
	RemoveSnapshots bool
}

// IPConfig is the deserialized object from the command-line
type IPConfig struct {
	ServiceID string
	IPAddress string
}

// RunningService contains the service for a state
type RunningService struct {
	Service *service.Service
	State   *servicestate.ServiceState
}

// Gets all of the available services
func (a *api) GetServices() ([]*service.Service, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var services []*service.Service
	if err := client.GetServices(&empty, &services); err != nil {
		return nil, err
	}

	return services, nil
}

// Gets the service definition identified by its service ID
func (a *api) GetService(id string) (*service.Service, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var s service.Service
	if err := client.GetService(id, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// Gets the service definition identified by its service Name
func (a *api) GetServicesByName(name string) ([]*service.Service, error) {
	allServices, err := a.GetServices()
	if err != nil {
		return nil, err
	}

	var services []*service.Service
	for i, s := range allServices {
		if s.Name == name || s.Id == name {
			services = append(services, allServices[i])
		}
	}

	return services, nil
}

// Adds a new service
func (a *api) AddService(config ServiceConfig) (*service.Service, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	endpoints := make([]service.ServiceEndpoint, len(*config.LocalPorts)+len(*config.RemotePorts))
	i := 0
	for _, e := range *config.LocalPorts {
		e.Purpose = "local"
		endpoints[i] = service.BuildServiceEndpoint(e)
		i++
	}
	for _, e := range *config.RemotePorts {
		e.Purpose = "remote"
		endpoints[i] = service.BuildServiceEndpoint(e)
		i++
	}

	s := service.Service{
		Name:           config.Name,
		PoolID:         config.PoolID,
		ImageID:        config.ImageID,
		Endpoints:      endpoints,
		Startup:        config.Command,
		Instances:      1,
		InstanceLimits: domain.MinMax{Min: 1, Max: 1},
	}

	var id string
	if err := client.AddService(s, &id); err != nil {
		return nil, err
	}

	return a.GetService(id)
}

// RemoveService removes an existing service
func (a *api) RemoveService(config RemoveServiceConfig) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	id := config.ServiceID

	if config.RemoveSnapshots {
		if err := client.DeleteSnapshots(id, &unusedInt); err != nil {
			return fmt.Errorf("could not clean up service history: %s", err)
		}
	}

	if err := client.RemoveService(id, &unusedInt); err != nil {
		return fmt.Errorf("could not remove service: %s", err)
	}

	return nil
}

// UpdateService updates an existing service
func (a *api) UpdateService(reader io.Reader) (*service.Service, error) {
	// Unmarshal JSON from the reader
	var s service.Service
	if err := json.NewDecoder(reader).Decode(&s); err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %s", err)
	}

	// Connect to the client
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	// Update the service
	if err := client.UpdateService(s, &unusedInt); err != nil {
		return nil, err
	}

	return a.GetService(s.Id)
}

// StartService starts a service
func (a *api) StartService(id string) (*host.Host, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var hostID string
	if err := client.StartService(id, &hostID); err != nil {
		return nil, err
	}

	return a.GetHost(hostID)
}

// StopService stops a service
func (a *api) StopService(id string) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	if err := client.StopService(id, &unusedInt); err != nil {
		return err
	}

	return nil
}

// AssignIP assigns an IP address to a service
func (a *api) AssignIP(config IPConfig) (string, error) {
	client, err := a.connectDAO()
	if err != nil {
		return "", err
	}

	req := dao.AssignmentRequest{
		ServiceID:      config.ServiceID,
		IPAddress:      config.IPAddress,
		AutoAssignment: config.IPAddress == "",
	}

	if err := client.AssignIPs(req, nil); err != nil {
		return "", err
	}

	var addresses []*addressassignment.AddressAssignment
	if err := client.GetServiceAddressAssignments(config.ServiceID, &addresses); err != nil {
		return "", err
	}

	if addresses == nil || len(addresses) == 0 {
		return "", nil
	}

	return addresses[0].IPAddr, nil
}
