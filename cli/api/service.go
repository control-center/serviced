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
	"encoding/json"
	"fmt"
	"io"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicestate"
)

const ()

var ()

// ServiceConfig is the deserialized object from the command-line
type ServiceConfig struct {
	Name            string
	ParentServiceID string
	ImageID         string
	Command         string
	LocalPorts      *PortMap
	RemotePorts     *PortMap
}

type SchedulerConfig struct {
	ServiceID  string
	AutoLaunch bool
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

// Type of method that controls the state of a service
type ServiceStateController func(SchedulerConfig) (int, error)

// Gets all of the available services
func (a *api) GetServices() ([]service.Service, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var services []service.Service
	var serviceRequest dao.ServiceRequest
	if err := client.GetServices(serviceRequest, &services); err != nil {
		return nil, err
	}

	return services, nil
}

// Gets all of the available services
func (a *api) GetServiceStates(serviceID string) ([]servicestate.ServiceState, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var states []servicestate.ServiceState
	if err := client.GetServiceStates(serviceID, &states); err != nil {
		return nil, err
	}

	return states, nil
}

func (a *api) GetServiceStatus(serviceID string) (map[string]dao.ServiceStatus, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var status map[string]dao.ServiceStatus
	if err := client.GetServiceStatus(serviceID, &status); err != nil {
		return nil, err
	}

	return status, nil
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
func (a *api) GetServicesByName(name string) ([]service.Service, error) {
	allServices, err := a.GetServices()
	if err != nil {
		return nil, err
	}

	var services []service.Service
	for i, s := range allServices {
		if s.Name == name || s.ID == name {
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

	endpoints := make([]servicedefinition.EndpointDefinition, len(*config.LocalPorts)+len(*config.RemotePorts))
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

	sd := &servicedefinition.ServiceDefinition{
		Name:      config.Name,
		Command:   config.Command,
		Instances: domain.MinMax{Min: 1, Max: 1, Default: 1},
		ImageID:   config.ImageID,
		Launch:    commons.AUTO,
		Endpoints: endpoints,
	}

	var serviceID string
	if err := client.DeployService(dao.ServiceDeploymentRequest{ParentID: config.ParentServiceID, Service: *sd}, &serviceID); err != nil {
		return nil, err
	}

	return a.GetService(serviceID)
}

// RemoveService removes an existing service
func (a *api) RemoveService(id string) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	if err := client.RemoveService(id, new(int)); err != nil {
		return fmt.Errorf("could not remove service %s: %s", id, err)
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

	return a.GetService(s.ID)
}

// StartService starts a service
func (a *api) StartService(config SchedulerConfig) (int, error) {
	client, err := a.connectDAO()
	if err != nil {
		return 0, err
	}

	var affected int
	err = client.StartService(dao.ScheduleServiceRequest{config.ServiceID, config.AutoLaunch}, &affected)
	return affected, err
}

// Restart
func (a *api) RestartService(config SchedulerConfig) (int, error) {
	client, err := a.connectDAO()
	if err != nil {
		return 0, err
	}

	var affected int
	err = client.RestartService(dao.ScheduleServiceRequest{config.ServiceID, config.AutoLaunch}, &affected)
	return affected, err
}

// StopService stops a service
func (a *api) StopService(config SchedulerConfig) (int, error) {
	client, err := a.connectDAO()
	if err != nil {
		return 0, err
	}

	var affected int
	err = client.StopService(dao.ScheduleServiceRequest{config.ServiceID, config.AutoLaunch}, &affected)
	return affected, err
}

// AssignIP assigns an IP address to a service
func (a *api) AssignIP(config IPConfig) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	req := dao.AssignmentRequest{
		ServiceID:      config.ServiceID,
		IPAddress:      config.IPAddress,
		AutoAssignment: config.IPAddress == "",
	}

	if err := client.AssignIPs(req, nil); err != nil {
		return err
	}

	return nil
}
