package api

import (
	"io"

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
	return nil, nil
}

// GetTemplate gets a particular serviced template by id
func (a *api) GetTemplate(id string) (*template.ServiceTemplate, error) {
	return nil, nil
}

// AddTemplate adds a new template
func (a *api) AddTemplate(reader io.Reader) (*template.ServiceTemplate, error) {
	return nil, nil
}

// RemoveTemplate removes an existing template by its template ID
func (a *api) RemoveTemplate(id string) error {
	return nil
}

// CompileTemplate builds a template given a source path
func (a *api) CompileTemplate(config CompileTemplateConfig) (io.Reader, error) {
	return nil, nil
}

// DeployTemplate deploys a template given its template ID
func (a *api) DeployTemplate(config DeployTemplateConfig) error {
	return nil
}
