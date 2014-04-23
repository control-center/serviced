package api

import (
	"encoding/json"
	"fmt"
	"io"

	service "github.com/zenoss/serviced/dao"
	template "github.com/zenoss/serviced/dao"
)

const ()

var ()

type DeployTemplateConfig struct {
	ID              string
	PoolID          string
	DeploymentID    string
	ManualAssignIPs bool
}

type CompileTemplateConfig struct {
	Dir string
	Map *ImageMap
}

// Gets all available service templates
func (a *api) GetServiceTemplates() ([]*template.ServiceTemplate, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	templatemap := make(map[string]*template.ServiceTemplate)
	if err := client.GetServiceTemplates(unusedInt, &templatemap); err != nil {
		return nil, err
	}
	templates := make([]*template.ServiceTemplate, len(templatemap))
	i := 0
	for id, t := range templatemap {
		(*t).Id = id
		templates[i] = t
		i++
	}

	return templates, nil
}

// Gets a particular serviced template by its template ID
func (a *api) GetServiceTemplate(id string) (*template.ServiceTemplate, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	templatemap := make(map[string]*template.ServiceTemplate)
	if err := client.GetServiceTemplates(unusedInt, &templatemap); err != nil {
		return nil, err
	}

	t := templatemap[id]
	(*t).Id = id

	return t, nil
}

// Adds a new service template
func (a *api) AddServiceTemplate(reader io.Reader) (*template.ServiceTemplate, error) {
	// Unmarshal JSON from the reader
	var t template.ServiceTemplate
	if err := json.NewDecoder(reader).Decode(&t); err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %s", err)
	}

	// Connect to the client
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	// Add the template
	var id string
	if err := client.AddServiceTemplate(t, &id); err != nil {
		return nil, err
	}

	return a.GetServiceTemplate(id)
}

// RemoveTemplate removes an existing template by its template ID
func (a *api) RemoveServiceTemplate(id string) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	if err := client.RemoveServiceTemplate(id, &unusedInt); err != nil {
		return err
	}

	return nil
}

// CompileTemplate builds a template given a source path
func (a *api) CompileServiceTemplate(config CompileTemplateConfig) (*template.ServiceTemplate, error) {
	sd, err := service.ServiceDefinitionFromPath(config.Dir)
	if err != nil {
		return nil, err
	}

	var mapImageNames func(*service.ServiceDefinition)
	mapImageNames = func(svc *service.ServiceDefinition) {
		if imageID, found := (*config.Map)[svc.ImageId]; found {
			(*svc).ImageId = imageID
		}
		for i := range (*svc).Services {
			mapImageNames(&svc.Services[i])
		}
	}
	mapImageNames(sd)

	t := template.ServiceTemplate{
		Services: []service.ServiceDefinition{*sd},
		Name:     sd.Name,
	}

	return &t, nil
}

// DeployTemplate deploys a template given its template ID
func (a *api) DeployServiceTemplate(config DeployTemplateConfig) (*service.Service, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	req := template.ServiceTemplateDeploymentRequest{
		PoolId:       config.PoolID,
		TemplateId:   config.ID,
		DeploymentId: config.DeploymentID,
	}

	var id string
	if err := client.DeployTemplate(req, &id); err != nil {
		return nil, err
	}

	s, err := a.GetService(id)
	if err != nil {
		return nil, err
	}

	if !config.ManualAssignIPs {
		if _, err := a.AssignIP(IPConfig{id, ""}); err != nil {
			return s, err
		}
	}

	return s, nil
}
