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

package servicetemplate

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/servicedefinition"
)


// A request to deploy a service template
type ServiceTemplateDeploymentRequest struct {
	PoolID       string // Pool Id to deploy service into
	TemplateID   string // Id of template to be deployed
	DeploymentID string // Unique id of the instance of this template
}

// ServiceTemplate type to hold service definitions
type ServiceTemplate struct {
	ID          string                                  // Unique ID of this service template
	Name        string                                  // Name of service template
	Version     string                                  // Version of the service
	Description string                                  // Meaningful description of service
	Services    []servicedefinition.ServiceDefinition   // Child services
	ConfigFiles map[string]servicedefinition.ConfigFile // Config file templates
	datastore.VersionedEntity
}

// Equals checks the equality of two service templates
func (a *ServiceTemplate) Equals(b *ServiceTemplate) bool {
	if a.ID != b.ID {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.Version != b.Version {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if !reflect.DeepEqual(a.Services, b.Services) {
		return false
	}
	if !reflect.DeepEqual(a.ConfigFiles, b.ConfigFiles) {
		return false
	}
	return true
}

func (a *ServiceTemplate) Hash() (string, error) {
	tpl := *a
	tpl.ID = ""
	tpl.DatabaseVersion = 0
	if data, err := json.Marshal(&tpl); err != nil {
		return "", err
	} else {
		return fmt.Sprintf("%x", md5.Sum(data)), nil
	}
}

//FromJSON creates a ServiceTemplate from the json string
func FromJSON(data string) (*ServiceTemplate, error) {
	var st ServiceTemplate
	err := json.Unmarshal([]byte(data), &st)
	return &st, err
}

// ServiceTemplateWrapper type for storing ServiceTemplates  TODO: no need to be public when CRUD moves hers
type serviceTemplateWrapper struct {
	ID              string // Primary-key - Should match ServiceTemplate.ID
	Name            string // Name of top level service
	Version         string // Version of the service
	Description     string // Description
	Data            string // JSON encoded template definition
	APIVersion      int    // Version of the ServiceTemplate API this expects
	TemplateVersion int    // Version of the template
	datastore.VersionedEntity
}

func newWrapper(st ServiceTemplate) (*serviceTemplateWrapper, error) {
	data, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}

	var wrapper serviceTemplateWrapper
	wrapper.ID = st.ID
	wrapper.Name = st.Name
	wrapper.Version = st.Version
	wrapper.Description = st.Description
	wrapper.Data = string(data)
	wrapper.APIVersion = 1
	wrapper.TemplateVersion = 1
	return &wrapper, nil

}

//BuildFromPath given a path will create a ServiceDefintion
func BuildFromPath(path string) (*ServiceTemplate, error) {
	sd, err := servicedefinition.BuildFromPath(path)
	if err != nil {
		return nil, err
	}
	st := ServiceTemplate{
		Services:    []servicedefinition.ServiceDefinition{*sd},
		Name:        sd.Name,
		Version:     sd.Version,
		Description: sd.Description,
	}
	return &st, nil
}
