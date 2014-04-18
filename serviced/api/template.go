package api

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/zenoss/glog"
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

// ListTemplates lists all available service templates
func (a *api) ListTemplates() ([]template.ServiceTemplate, error) {
	client, err := connect()
	if err != nil {
		return nil, err
	}

	templatemap := make(map[string]*template.ServiceTemplate)
	if err := client.GetServiceTemplates(unusedInt, &templatemap); err != nil {
		return nil, fmt.Errorf("could not get service templates: %s", err)
	}
	templates := make([]template.ServiceTemplate, len(templatemap))
	i := 0
	for id, t := range templatemap {
		(*t).Id = id
		templates[i] = *t
		i++
	}

	return templates, nil
}

// GetTemplate gets a particular serviced template by id
func (a *api) GetTemplate(id string) (*template.ServiceTemplate, error) {
	client, err := connect()
	if err != nil {
		return nil, err
	}

	templatemap := make(map[string]*template.ServiceTemplate)
	if err := client.GetServiceTemplates(unusedInt, &templatemap); err != nil {
		return nil, fmt.Errorf("could not get service templates: %s", err)
	}

	t := templatemap[id]
	(*t).Id = id

	return t, nil
}

// AddTemplate adds a new template
func (a *api) AddTemplate(reader io.Reader) (*template.ServiceTemplate, error) {
	// Unmarshal JSON from the reader
	var t template.ServiceTemplate
	if err := json.NewDecoder(reader).Decode(&t); err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %s", err)
	}

	// Connect to the client
	client, err := connect()
	if err != nil {
		return nil, err
	}

	// Add the template
	var id string
	if err := client.AddServiceTemplate(t, &id); err != nil {
		return nil, fmt.Errorf("could not add service template: %s", err)
	}

	// Get the service template that was added
	templatemap := make(map[string]*template.ServiceTemplate)
	if err := client.GetServiceTemplates(unusedInt, &templatemap); err != nil {
		return nil, fmt.Errorf("could not get service templates: %s", err)
	}
	t = *templatemap[id]
	t.Id = id

	return &t, nil
}

// RemoveTemplate removes an existing template by its template ID
func (a *api) RemoveTemplate(id string) error {
	client, err := connect()
	if err != nil {
		return err
	}

	if err := client.RemoveServiceTemplate(id, &unusedInt); err != nil {
		return fmt.Errorf("could not remove service template: %s", err)
	}

	return nil
}

// CompileTemplate builds a template given a source path
func (a *api) CompileTemplate(config CompileTemplateConfig) (*template.ServiceTemplate, error) {
	sd, err := service.ServiceDefinitionFromPath(config.Dir)
	if err != nil {
		return nil, fmt.Errorf("could not get the service definition: %s", err)
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
func (a *api) DeployTemplate(config DeployTemplateConfig) (*service.Service, error) {
	client, err := connect()
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
		return nil, fmt.Errorf("could not deploy template: %s", err)
	}

	var s service.Service
	if err := client.GetService(id, &s); err != nil {
		return nil, fmt.Errorf("could not get service definition: %s", err)
	}

	if !config.ManualAssignIPs {
		glog.V(0).Infof("Assigning IP addresses automatically to services requiring them.")
		if err := client.AssignIPs(service.AssignmentRequest{id, "", true}, nil); err != nil {
			return &s, fmt.Errorf("could not automatically assign IPs: %s", err)
		}
	} else {
		glog.V(0).Infof("Not assigning IP address to services requiring them.  You need to do this manually.")
	}

	return &s, nil
}
