package api

import (
	"encoding/json"
	"fmt"
	"io"

	service "github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/servicedefinition"
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

// IPConfig is the deserialized object from the command-line
type IPConfig struct {
	ServiceID string
	IPAddress string
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
		if s.Name == name {
			services = append(services, allServices[i])
		}
	}

	return services, nil
}

// Gets the service states for a service identified by its service ID
func (a *api) GetServiceStatesByServiceID(id string) ([]*service.ServiceState, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var states []*service.ServiceState
	if err := client.GetServiceStates(id, &states); err != nil {
		return nil, err
	}

	return states, nil
}

// Gets the service states for a service identified by the docker ID
func (a *api) GetServiceStatesByDockerID(id string) (*service.ServiceState, error) {
	services, err := a.GetServices()
	if err != nil {
		return nil, err
	}

	for _, s := range services {
		states, err := a.GetServiceStatesByServiceID(s.Id)
		if err != nil {
			return nil, err
		}
		for i, ss := range states {
			if ss.DockerId == id {
				return states[i], nil
			}
		}
	}

	return nil, nil
}

// Adds a new service
func (a *api) AddService(config ServiceConfig) (*service.Service, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	endpoints := make([]servicedefinition.ServiceEndpoint, len(*config.LocalPorts)+len(*config.RemotePorts))
	i := 0
	for _, e := range *config.LocalPorts {
		e.Purpose = "local"
		endpoints[i] = e
		i++
	}
	for _, e := range *config.RemotePorts {
		e.Purpose = "remote"
		endpoints[i] = e
		i++
	}

	s := service.Service{
		Name:      config.Name,
		PoolId:    config.PoolID,
		ImageId:   config.ImageID,
		Endpoints: endpoints,
		Startup:   config.Command,
		Instances: 1,
	}

	var id string
	if err := client.AddService(s, &id); err != nil {
		return nil, err
	}

	return a.GetService(id)
}

// RemoveService removes an existing service
func (a *api) RemoveService(id string) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	if err := client.DeleteSnapshots(id, &unusedInt); err != nil {
		return fmt.Errorf("could not clean up service history", err)
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
func (a *api) AssignIP(config IPConfig) ([]servicedefinition.AddressAssignment, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	req := service.AssignmentRequest{
		ServiceId:      config.ServiceID,
		IpAddress:      config.IPAddress,
		AutoAssignment: config.IPAddress == "",
	}

	if err := client.AssignIPs(req, nil); err != nil {
		return nil, err
	}

	var addresses []servicedefinition.AddressAssignment
	if err := client.GetServiceAddressAssignments(config.ServiceID, &addresses); err != nil {
		return nil, err
	}

	return addresses, nil
}
