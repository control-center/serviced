package api

import (
	"encoding/json"
	"fmt"
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
	LocalPorts  *PortMap
	RemotePorts *PortMap
}

// IPConfig is the deserialized object from the command-line
type IPConfig struct {
	ServiceID string
	IPAddress string
}

// ListServices lists all of the available services
func (a *api) ListServices() ([]service.Service, error) {
	client, err := a.connect()
	if err != nil {
		return nil, err
	}

	var servicemap []*service.Service
	if err := client.GetServices(&empty, &servicemap); err != nil {
		return nil, fmt.Errorf("could not get services: %s", err)
	}

	services := make([]service.Service, len(servicemap))
	for i, s := range servicemap {
		services[i] = *s
	}

	return services, nil
}

// GetService gets the service definition identified by its service ID
func (a *api) GetService(id string) (*service.Service, error) {
	client, err := a.connect()
	if err != nil {
		return nil, err
	}

	var s service.Service
	if err := client.GetService(id, &s); err != nil {
		return nil, fmt.Errorf("could not get service definition: %s", err)
	}

	return &s, nil
}

// AddService adds a new service
func (a *api) AddService(config ServiceConfig) (*service.Service, error) {
	client, err := a.connect()
	if err != nil {
		return nil, err
	}

	endpoints := make([]service.ServiceEndpoint, len(*config.LocalPorts)+len(*config.RemotePorts))
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
		return nil, fmt.Errorf("could not add service: %s", err)
	}

	return a.GetService(id)
}

// RemoveService removes an existing service
func (a *api) RemoveService(id string) error {
	client, err := a.connect()
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
	client, err := a.connect()
	if err != nil {
		return nil, err
	}

	// Update the service
	if err := client.UpdateService(s, &unusedInt); err != nil {
		return nil, fmt.Errorf("could not update service definition: %s", err)
	}

	return a.GetService(s.Id)
}

// StartService starts a service
func (a *api) StartService(id string) (*host.Host, error) {
	client, err := a.connect()
	if err != nil {
		return nil, err
	}

	var hostID string
	if err := client.StartService(id, &hostID); err != nil {
		return nil, fmt.Errorf("could not start service: %s", err)
	}

	return a.GetHost(hostID)
}

// StopService stops a service
func (a *api) StopService(id string) error {
	client, err := a.connect()
	if err != nil {
		return err
	}

	if err := client.StopService(id, &unusedInt); err != nil {
		return fmt.Errorf("could not stop service: %s", err)
	}

	return nil
}

// AssignIP assigns an IP address to a service
func (a *api) AssignIP(config IPConfig) ([]service.AddressAssignment, error) {
	client, err := a.connect()
	if err != nil {
		return nil, err
	}

	req := service.AssignmentRequest{
		ServiceId:      config.ServiceID,
		IpAddress:      config.IPAddress,
		AutoAssignment: config.IPAddress == "",
	}

	if err := client.AssignIPs(req, nil); err != nil {
		return nil, fmt.Errorf("could not assign IP: %s", err)
	}

	var addresses []service.AddressAssignment
	if err := client.GetServiceAddressAssignments(config.ServiceID, &addresses); err != nil {
		return nil, fmt.Errorf("could not get address assignments: %s", err)
	}

	return addresses, nil
}
