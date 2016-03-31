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

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	template "github.com/control-center/serviced/domain/servicetemplate"
)

// DeployTemplateConfig is the configuration object to deploy a template
type DeployTemplateConfig struct {
	ID              string
	PoolID          string
	DeploymentID    string
	ManualAssignIPs bool
}

// CompileTemplateConfig is the configuration object to conpile a template directory
type CompileTemplateConfig struct {
	Dir string
	Map ImageMap
}

// Gets all available service templates
func (a *api) GetServiceTemplates() ([]template.ServiceTemplate, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	templateMap, err := client.GetServiceTemplates()
	if err != nil {
			return nil, err
	}
	templates := make([]template.ServiceTemplate, len(templateMap))
	i := 0
	for id, t := range templateMap {
		t.ID = id
		templates[i] = t
		i++
	}

	return templates, nil
}

// Gets a particular serviced template by its template ID
func (a *api) GetServiceTemplate(id string) (*template.ServiceTemplate, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	templateMap, err := client.GetServiceTemplates()
	if err != nil {
		return nil, err
	}

	if _, ok := templateMap[id]; !ok {
		return nil, fmt.Errorf("unable to find template by id: %s", id)
	}
	t := templateMap[id]
	t.ID = id

	return &t, nil
}

// Adds a new service template
func (a *api) AddServiceTemplate(reader io.Reader) (*template.ServiceTemplate, error) {
	// Unmarshal JSON from the reader
	var t template.ServiceTemplate
	if err := json.NewDecoder(reader).Decode(&t); err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %s", err)
	}

	// Connect to the client
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	// Add the template
	if id, err := client.AddServiceTemplate(t); err != nil {
		return nil, err
	} else {
		return a.GetServiceTemplate(id)
	}

}

// RemoveTemplate removes an existing template by its template ID
func (a *api) RemoveServiceTemplate(id string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.RemoveServiceTemplate(id)
}

// CompileTemplate builds a template given a source path
func (a *api) CompileServiceTemplate(config CompileTemplateConfig) (*template.ServiceTemplate, error) {
	st, err := template.BuildFromPath(config.Dir)
	if err != nil {
		return nil, err
	}

	var mapImageNames func(*servicedefinition.ServiceDefinition)
	mapImageNames = func(svc *servicedefinition.ServiceDefinition) {
		if imageID, found := config.Map[svc.ImageID]; found {
			svc.ImageID = imageID
		}
		for i := range svc.Services {
			mapImageNames(&svc.Services[i])
		}
	}
	for idx := range st.Services {
		mapImageNames(&st.Services[idx])
	}
	return st, nil
}

// DeployTemplate deploys a template given its template ID
func (a *api) DeployServiceTemplate(config DeployTemplateConfig) ([]service.Service, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	req := template.ServiceTemplateDeploymentRequest{
		PoolID:       config.PoolID,
		TemplateID:   config.ID,
		DeploymentID: config.DeploymentID,
	}

	ids, err := client.DeployTemplate(req);
	if err != nil {
		return nil, err
	}

	svcs := make([]service.Service, len(ids))
	for i, id := range ids {
		s, err := a.GetService(id)
		if err != nil {
			return nil, err
		}

		if !config.ManualAssignIPs {
			a.AssignIP(IPConfig{id, ""})
		}
		svcs[i] = *s
	}

	return svcs, nil
}
