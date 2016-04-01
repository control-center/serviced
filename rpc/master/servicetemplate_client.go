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
func (c *Client) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate) (templateID string, err error) {
	response := ""
	if err := c.call("AddServiceTemplate", serviceTemplate, &response); err != nil {
		return "", err
	}
	return response, nil
}

// Get a list of service templates
func (c *Client) GetServiceTemplates() (map[string]servicetemplate.ServiceTemplate, error) {
	response := map[string]servicetemplate.ServiceTemplate{}
	if err := c.call("GetServiceTemplates", empty, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// Remove a service Template
func (c *Client) RemoveServiceTemplate(serviceTemplateID string) error {
	if err := c.call("RemoveServiceTemplate", serviceTemplateID, nil); err != nil {
		return err
	}
	return nil

}

// Deploy a service Template
func (c *Client) DeployTemplate(request servicetemplate.ServiceTemplateDeploymentRequest) (tenantIDs []string, err error){
	response := []string{}
	if err := c.call("DeployTemplate", request, &response); err != nil {
		return nil, err
	}
	return response, nil

}

