// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package service

import (
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/utils"

	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// A Service that can run in serviced.
type Service struct {
	Id              string
	Name            string
	Context         string
	Startup         string
	Description     string
	Tags            []string
	ConfigFiles     map[string]servicedefinition.ConfigFile
	Instances       int
	ImageId         string
	PoolId          string
	DesiredState    int
	HostPolicy      servicedefinition.HostPolicy
	Hostname        string
	Privileged      bool
	Launch          string
	Endpoints       []ServiceEndpoint
	Tasks           []servicedefinition.Task
	ParentServiceId string
	Volumes         []servicedefinition.Volume
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeploymentId    string
	DisableImage    bool
	LogConfigs      []servicedefinition.LogConfig
	Snapshot        servicedefinition.SnapshotCommands
	Runs            map[string]string
	RAMCommitment   uint64
	Actions         map[string]string
}

type ServiceEndpoint struct {
	servicedefinition.EndpointDefinition
	AddressAssignment AddressAssignment
}

//AddressAssignment is used to track Ports that have been assigned to a Service.
type AddressAssignment struct {
	ID             string //Generated id
	AssignmentType string //Static or Virtual
	HostID         string //Host id if type is Static
	PoolID         string //Pool id if type is Virtual
	IPAddr         string //Used to associate to resource in Pool or Host
	Port           uint16 //Actual assigned port
	ServiceID      string //Service using this assignment
	EndpointName   string //Endpoint in the service using the assignment
}

// Create a new Service.
func NewService() (s *Service, err error) {
	s = &Service{}
	s.Id, err = utils.NewUUID()
	return s, err
}

// Does the service have endpoint imports
func (s *Service) HasImports() bool {
	if s.Endpoints == nil {
		return false
	}

	for _, ep := range s.Endpoints {
		if ep.Purpose == "import" {
			return true
		}
	}
	return false
}

func BuildServiceEndpoint(epd servicedefinition.EndpointDefinition) ServiceEndpoint {
	return ServiceEndpoint{EndpointDefinition: epd}
}

func BuildService(sd servicedefinition.ServiceDefinition, parentServiceId string, poolID string, desiredState int, deploymentId string) (*Service, error) {
	svcuuid, err := utils.NewUUID()
	if err != nil {
		return nil, err
	}

	ctx, err := json.Marshal(sd.Context)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	svc := Service{}
	svc.Id = svcuuid
	svc.Name = sd.Name
	svc.Context = string(ctx)
	svc.Startup = sd.Command
	svc.Description = sd.Description
	svc.Tags = sd.Tags
	svc.Instances = sd.Instances.Min
	svc.ImageId = sd.ImageID
	svc.PoolId = poolID
	svc.DesiredState = desiredState
	svc.Launch = sd.Launch
	svc.HostPolicy = sd.HostPolicy
	svc.Hostname = sd.Hostname
	svc.Privileged = sd.Privileged
	svc.ConfigFiles = sd.ConfigFiles
	svc.Tasks = sd.Tasks
	svc.ParentServiceId = parentServiceId
	svc.CreatedAt = now
	svc.UpdatedAt = now
	svc.Volumes = sd.Volumes
	svc.DeploymentId = deploymentId
	svc.LogConfigs = sd.LogConfigs
	svc.Snapshot = sd.Snapshot
	svc.RAMCommitment = sd.RAMCommitment
	svc.Runs = sd.Runs
	svc.Actions = sd.Actions

	svc.Endpoints = make([]ServiceEndpoint, 0)
	for _, ep := range sd.Endpoints {
		svc.Endpoints = append(svc.Endpoints, ServiceEndpoint{EndpointDefinition: ep})
	}

	return &svc, nil
}

// GetServiceImports retrieves service endpoints whose purpose is "import"
func (s *Service) GetServiceImports() []ServiceEndpoint {
	result := []ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if ep.Purpose == "import" {
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

// Add a virtual host for given service, this method avoids duplicates vhosts
func (s *Service) AddVirtualHost(application, vhostName string) error {
	if s.Endpoints != nil {

		//find the matching endpoint
		for i := 0; i < len(s.Endpoints); i += 1 {
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

	return fmt.Errorf("Unable to find application %s in service: %s", application, s.Name)
}

// Remove a virtual host for given service
func (s *Service) RemoveVirtualHost(application, vhostName string) error {
	if s.Endpoints != nil {

		//find the matching endpoint
		for i := 0; i < len(s.Endpoints); i += 1 {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				if len(ep.VHosts) == 0 {
					break
				}

				_vhostName := strings.ToLower(vhostName)
				if len(ep.VHosts) == 1 && ep.VHosts[0] == _vhostName {
					return fmt.Errorf("Cannot delete last vhost: %s", _vhostName)
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

	return fmt.Errorf("Unable to find application %s in service: %s", application, s.Name)
}

func (se *ServiceEndpoint) SetAssignment(aa *AddressAssignment) error {
	if se.AddressConfig.Port == 0 {
		return errors.New("Cannot assign address to endpoint without AddressResourceConfig")
	}
	se.AddressAssignment = *aa
	return nil
}

//GetAssignment Returns nil if no assignment set
func (se *ServiceEndpoint) GetAssignment() *AddressAssignment {
	if se.AddressAssignment.ID == "" {
		return nil
	}
	//return reference to copy
	result := se.AddressAssignment
	return &result
}

func (a *Service) Equals(b *Service) bool {
	if a.Id != b.Id {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.Context != b.Context {
		return false
	}
	if a.Startup != b.Startup {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.Instances != b.Instances {
		return false
	}
	if a.ImageId != b.ImageId {
		return false
	}
	if a.PoolId != b.PoolId {
		return false
	}
	if a.DesiredState != b.DesiredState {
		return false
	}
	if a.Launch != b.Launch {
		return false
	}
	if a.Hostname != b.Hostname {
		return false
	}
	if a.Privileged != b.Privileged {
		return false
	}
	if a.HostPolicy != b.HostPolicy {
		return false
	}
	if a.ParentServiceId != b.ParentServiceId {
		return false
	}
	if a.CreatedAt != b.CreatedAt {
		return false
	}
	if a.UpdatedAt != b.CreatedAt {
		return false
	}
	return true
}

//func parent(cp ControlPlane) func(s Service) (value Service, err error) {
//	return func(s Service) (value Service, err error) {
//		err = cp.GetService(s.ParentServiceId, &value)
//		return
//	}
//}
//
//func context(cp ControlPlane) func(s Service) (ctx map[string]interface{}, err error) {
//	return func(s Service) (ctx map[string]interface{}, err error) {
//		err = json.Unmarshal([]byte(s.Context), &ctx)
//		if err != nil {
//			glog.Errorf("Error unmarshal service context Id=%s: %s -> %s", s.Id, s.Context, err)
//		}
//		return
//	}
//}
//
//// EvaluateActionsTemplate parses and evaluates the Actions string of a service.
//func (service *Service) EvaluateActionsTemplate(cp ControlPlane) (err error) {
//	for key, value := range service.Actions {
//		result := service.evaluateTemplate(cp, value)
//		if result != "" {
//			service.Actions[key] = result
//		}
//	}
//	return
//}
//
//// EvaluateStartupTemplate parses and evaluates the StartUp string of a service.
//func (service *Service) EvaluateStartupTemplate(cp ControlPlane) (err error) {
//
//	result := service.evaluateTemplate(cp, service.Startup)
//	if result != "" {
//		service.Startup = result
//	}
//
//	return
//}
//
//// EvaluateRunsTemplate parses and evaluates the Runs string of a service.
//func (service *Service) EvaluateRunsTemplate(cp ControlPlane) (err error) {
//	for key, value := range service.Runs {
//		result := service.evaluateTemplate(cp, value)
//		if result != "" {
//			service.Runs[key] = result
//		}
//	}
//	return
//}
//
//// evaluateTemplate takes a control plane client and template string and evaluates
//// the template using the service as the context. If the template is invalid or there is an error
//// then an empty string is returned.
//func (service *Service) evaluateTemplate(cp ControlPlane, serviceTemplate string) string {
//	functions := template.FuncMap{
//		"parent":  parent(cp),
//		"context": context(cp),
//	}
//	// parse the template
//	t := template.Must(template.New("ServiceDefinitionTemplate").Funcs(functions).Parse(serviceTemplate))
//
//	// evaluate it
//	var buffer bytes.Buffer
//	err := t.Execute(&buffer, service)
//	if err == nil {
//		return buffer.String()
//	}
//
//	// something went wrong, warn them
//	glog.Warning("Evaluating template %s produced the following error %s ", serviceTemplate, err)
//	return ""
//}
//
//// EvaluateLogConfigTemplate parses and evals the Path, Type and all the values for the tags of the log
//// configs. This happens for each LogConfig on the service.
//func (service *Service) EvaluateLogConfigTemplate(cp ControlPlane) (err error) {
//	// evaluate the template for the LogConfig as well as the tags
//
//	for i, logConfig := range service.LogConfigs {
//		// Path
//		result := service.evaluateTemplate(cp, logConfig.Path)
//		if result != "" {
//			service.LogConfigs[i].Path = result
//		}
//		// Type
//		result = service.evaluateTemplate(cp, logConfig.Type)
//		if result != "" {
//			service.LogConfigs[i].Type = result
//		}
//
//		// Tags
//		for j, tag := range logConfig.LogTags {
//			result = service.evaluateTemplate(cp, tag.Value)
//			if result != "" {
//				service.LogConfigs[i].LogTags[j].Value = result
//			}
//		}
//	}
//	return
//}
//
//// EvaluateEndpointTemplates parses and evaluates the "ApplicationTemplate" property
//// of each of the service endpoints for this service.
//func (service *Service) EvaluateEndpointTemplates(cp ControlPlane) (err error) {
//	functions := template.FuncMap{
//		"parent":  parent(cp),
//		"context": context(cp),
//	}
//
//	for i, ep := range service.Endpoints {
//		if ep.Application != "" && ep.ApplicationTemplate == "" {
//			ep.ApplicationTemplate = ep.Application
//			service.Endpoints[i].ApplicationTemplate = ep.Application
//		}
//		if ep.ApplicationTemplate != "" {
//			t := template.Must(template.New(service.Name).Funcs(functions).Parse(ep.ApplicationTemplate))
//			var buffer bytes.Buffer
//			if err = t.Execute(&buffer, service); err == nil {
//				service.Endpoints[i].Application = buffer.String()
//			} else {
//				return
//			}
//		}
//	}
//	return
//}
