// Copyright 2016 The Serviced Authors.
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

package master

import (
	"github.com/control-center/serviced/domain/servicetemplate"
)

// Add a new service template
func (s *Server) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, response *string) error  {
	templateID, err := s.f.AddServiceTemplate(s.context(), serviceTemplate)
	if err != nil {
		return err
	}
	*response = templateID
	return nil
}

// Get a list of service templates
func (s *Server) GetServiceTemplates(unused struct{}, response *map[string]servicetemplate.ServiceTemplate) error  {
	templates, err := s.f.GetServiceTemplates(s.context())
	if err != nil {
		return err
	}
	*response = templates
	return nil
}

// Remove a service template
func (s *Server) RemoveServiceTemplate(templateID string,  _ *struct{}) error  {
	return s.f.RemoveServiceTemplate(s.context(), templateID)
}

// Deploy a service template
func (s *Server) DeployTemplate(request servicetemplate.ServiceTemplateDeploymentRequest, response *[]string) error  {
	tenantIDs, err := s.f.DeployTemplate(s.context(), request.PoolID, request.TemplateID, request.DeploymentID)
	if err != nil {
		return err
	}
	*response = tenantIDs
	return nil
}
