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

package service

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
)

// Desired states of services.
type DesiredState int

func (state DesiredState) String() string {
	switch state {
	case SVCRestart:
		return "restart"
	case SVCStop:
		return "stop"
	case SVCRun:
		return "go"
	case SVCPause:
		return "pause"
	default:
		return "unknown"
	}
}

const (
	SVCRestart = DesiredState(-1)
	SVCStop    = DesiredState(0)
	SVCRun     = DesiredState(1)
	SVCPause   = DesiredState(2)
)

// Service A Service that can run in serviced.
type Service struct {
	ID                string
	Name              string
	Title             string // Title is a label used when describing this service in the context of a service tree
	Version           string
	Context           map[string]interface{}
	Startup           string
	Description       string
	Tags              []string
	OriginalConfigs   map[string]servicedefinition.ConfigFile
	ConfigFiles       map[string]servicedefinition.ConfigFile
	Instances         int
	InstanceLimits    domain.MinMax
	ChangeOptions     []string
	ImageID           string
	PoolID            string
	DesiredState      int
	HostPolicy        servicedefinition.HostPolicy
	Hostname          string
	Privileged        bool
	Launch            string
	Endpoints         []ServiceEndpoint
	Tasks             []servicedefinition.Task
	ParentServiceID   string
	Volumes           []servicedefinition.Volume
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeploymentID      string
	DisableImage      bool
	LogConfigs        []servicedefinition.LogConfig
	Snapshot          servicedefinition.SnapshotCommands
	Runs              map[string]string
	RAMCommitment     utils.EngNotation
	CPUCommitment     uint64
	Actions           map[string]string
	HealthChecks      map[string]domain.HealthCheck // A health check for the service.
	Prereqs           []domain.Prereq               // Optional list of scripts that must be successfully run before kicking off the service command.
	MonitoringProfile domain.MonitorProfile
	MemoryLimit       float64
	CPUShares         int64
	PIDFile           string
	datastore.VersionedEntity
}

//ServiceEndpoint endpoint exported or imported by a service
type ServiceEndpoint struct {
	servicedefinition.EndpointDefinition
	AddressAssignment addressassignment.AddressAssignment
}

// IsConfigurable returns true if the endpoint is configurable
func (endpoint ServiceEndpoint) IsConfigurable() bool {
	return endpoint.AddressConfig.Port > 0 && endpoint.AddressConfig.Protocol != ""
}

// NewService Create a new Service.
func NewService() (s *Service, err error) {
	s = &Service{}
	s.ID, err = utils.NewUUID36()
	return s, err
}

// HasEndpointsFor determines if the service has any imports
// for the specified purpose, eg import
func (s *Service) HasEndpointsFor(purpose string) bool {
	if s.Endpoints == nil {
		return false
	}

	for _, ep := range s.Endpoints {
		if ep.Purpose == purpose {
			return true
		}
	}
	return false
}

//BuildServiceEndpoint build a ServiceEndpoint from a EndpointDefinition
func BuildServiceEndpoint(epd servicedefinition.EndpointDefinition) ServiceEndpoint {
	return ServiceEndpoint{EndpointDefinition: epd}
}

//BuildService build a service from a ServiceDefinition.
func BuildService(sd servicedefinition.ServiceDefinition, parentServiceID string, poolID string, desiredState int, deploymentID string) (*Service, error) {
	svcuuid, err := utils.NewUUID36()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	svc := Service{}
	svc.ID = svcuuid
	svc.Name = sd.Name
	svc.Title = sd.Title
	svc.Version = sd.Version
	svc.Context = sd.Context
	svc.Startup = sd.Command
	svc.Description = sd.Description
	svc.Tags = sd.Tags
	if sd.Instances.Default != 0 {
		svc.Instances = sd.Instances.Default
	} else {
		svc.Instances = sd.Instances.Min
	}
	svc.InstanceLimits = sd.Instances
	svc.ChangeOptions = sd.ChangeOptions
	svc.ImageID = sd.ImageID
	svc.PoolID = poolID
	svc.DesiredState = desiredState
	svc.Launch = sd.Launch
	svc.HostPolicy = sd.HostPolicy
	svc.Hostname = sd.Hostname
	svc.Privileged = sd.Privileged
	svc.OriginalConfigs = sd.ConfigFiles
	svc.ConfigFiles = sd.ConfigFiles
	svc.Tasks = sd.Tasks
	svc.ParentServiceID = parentServiceID
	svc.CreatedAt = now
	svc.UpdatedAt = now
	svc.Volumes = sd.Volumes
	svc.DeploymentID = deploymentID
	svc.LogConfigs = sd.LogConfigs
	svc.Snapshot = sd.Snapshot
	svc.RAMCommitment = sd.RAMCommitment
	svc.Runs = sd.Runs
	svc.Actions = sd.Actions
	svc.HealthChecks = sd.HealthChecks
	svc.Prereqs = sd.Prereqs
	svc.PIDFile = sd.PIDFile

	svc.Endpoints = make([]ServiceEndpoint, 0)
	for _, ep := range sd.Endpoints {
		svc.Endpoints = append(svc.Endpoints, ServiceEndpoint{EndpointDefinition: ep})
	}

	tags := map[string][]string{
		"controlplane_service_id": []string{svc.ID},
	}
	profile, err := sd.MonitoringProfile.ReBuild("1h-ago", tags)
	if err != nil {
		return nil, err
	}
	svc.MonitoringProfile = *profile
	svc.MemoryLimit = sd.MemoryLimit
	svc.CPUShares = sd.CPUShares

	return &svc, nil
}

//CloneService copies a service and mutates id and names
func CloneService(fromSvc *Service, suffix string) (*Service, error) {
	svcuuid, err := utils.NewUUID36()
	if err != nil {
		return nil, err
	}

	svc := *fromSvc
	svc.ID = svcuuid
	svc.DesiredState = int(SVCStop)

	now := time.Now()
	svc.CreatedAt = now
	svc.UpdatedAt = now

	// add suffix to make certain things unique
	suffix = strings.TrimSpace(suffix)
	if len(suffix) == 0 {
		suffix = "-" + svc.ID[0:12]
	}
	svc.Name += suffix
	for idx := range svc.Endpoints {
		svc.Endpoints[idx].Name += suffix
		svc.Endpoints[idx].Application += suffix
		svc.Endpoints[idx].ApplicationTemplate += suffix
	}
	for idx := range svc.Volumes {
		svc.Volumes[idx].ResourcePath += suffix
	}

	return &svc, nil
}

// GetServiceImports retrieves service endpoints whose purpose is "import"
func (s *Service) GetServiceImports() []ServiceEndpoint {
	result := []ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if ep.Purpose == "import" || ep.Purpose == "import_all" {
				result = append(result, ep)
			}
		}
	}

	return result
}

// GetServiceExports retrieves service endpoints whose purpose is "export"
func (s *Service) GetServiceExports() []ServiceEndpoint {
	result := []ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if ep.Purpose == "export" {
				result = append(result, ep)
			}
		}
	}

	return result
}

// GetServiceVHosts retrieves service endpoints that specify a virtual HostPort
func (s *Service) GetServiceVHosts() []ServiceEndpoint {
	result := []ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if len(ep.VHosts) > 0 {
				result = append(result, ep)
			}
		}
	}

	return result
}

// AddVirtualHost Add a virtual host for given service, this method avoids duplicates vhosts
func (s *Service) AddVirtualHost(application, vhostName string) error {
	if s.Endpoints != nil {

		//find the matching endpoint
		for i := range s.Endpoints {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				_vhostName := strings.ToLower(vhostName)
				vhosts := make([]string, 0)
				for _, vhost := range ep.VHosts {
					if strings.ToLower(vhost) != _vhostName {
						vhosts = append(vhosts, vhost)
					}
				}
				ep.VHosts = append(vhosts, _vhostName)
				return nil
			}
		}
	}

	return fmt.Errorf("unable to find application %s in service: %s", application, s.Name)
}

// RemoveVirtualHost Remove a virtual host for given service
func (s *Service) RemoveVirtualHost(application, vhostName string) error {
	if s.Endpoints != nil {

		//find the matching endpoint
		for i := range s.Endpoints {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				if len(ep.VHosts) == 0 {
					break
				}

				_vhostName := strings.ToLower(vhostName)
				if len(ep.VHosts) == 1 && ep.VHosts[0] == _vhostName {
					return fmt.Errorf("cannot delete last vhost: %s", _vhostName)
				}

				found := false
				vhosts := make([]string, 0)
				for _, vhost := range ep.VHosts {
					if vhost != _vhostName {
						vhosts = append(vhosts, vhost)
					} else {
						found = true
					}
				}
				//error removing an unknown vhost
				if !found {
					break
				}

				ep.VHosts = vhosts
				return nil
			}
		}
	}

	return fmt.Errorf("unable to find application %s in service: %s", application, s.Name)
}

// GetPath uses the GetService function to determine the / delimited name path i.e. /test/app/sevicename
func (s Service) GetPath(gs GetService) (string, error) {
	var err error
	svc := s
	path := fmt.Sprintf("/%s", s.Name)
	for svc.ParentServiceID != "" {
		svc, err = gs(svc.ParentServiceID)
		if err != nil {
			return "", err
		}
		path = fmt.Sprintf("/%s%s", svc.Name, path)
	}
	return path, nil
}

//SetAssignment sets the AddressAssignment for the endpoint
func (se *ServiceEndpoint) SetAssignment(aa addressassignment.AddressAssignment) error {
	if se.AddressConfig.Port == 0 {
		return errors.New("cannot assign address to endpoint without AddressResourceConfig")
	}
	se.AddressAssignment = aa
	return nil
}

//RemoveAssignment resets a service endpoints to nothing
func (se *ServiceEndpoint) RemoveAssignment() error {
	se.AddressAssignment = addressassignment.AddressAssignment{}
	return nil
}

//GetAssignment Returns nil if no assignment set
func (se *ServiceEndpoint) GetAssignment() *addressassignment.AddressAssignment {
	if se.AddressAssignment.ID == "" {
		return nil
	}
	//return reference to copy
	result := se.AddressAssignment
	return &result
}

//Equals are they the same
func (s *Service) Equals(b *Service) bool {
	if s.ID != b.ID {
		return false
	}
	if s.Name != b.Name {
		return false
	}
	if s.Version != b.Version {
		return false
	}
	if !reflect.DeepEqual(s.Context, b.Context) {
		return false
	}
	if s.Startup != b.Startup {
		return false
	}
	if s.Description != b.Description {
		return false
	}
	if s.Instances != b.Instances {
		return false
	}
	if s.ImageID != b.ImageID {
		return false
	}
	if s.PoolID != b.PoolID {
		return false
	}
	if s.DesiredState != b.DesiredState {
		return false
	}
	if s.Launch != b.Launch {
		return false
	}
	if s.Hostname != b.Hostname {
		return false
	}
	if s.Privileged != b.Privileged {
		return false
	}
	if s.HostPolicy != b.HostPolicy {
		return false
	}
	if s.ParentServiceID != b.ParentServiceID {
		return false
	}
	if s.CreatedAt.Unix() != b.CreatedAt.Unix() {
		return false
	}
	if s.UpdatedAt.Unix() != b.CreatedAt.Unix() {
		return false
	}
	return true
}
