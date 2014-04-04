package api

import (
	"io"

	"github.com/zenoss/serviced/domain/template"
)

const ()

var ()

// ListTemplates lists all available service templates
func (a *api) ListTemplates() ([]template.Template, error) {
	return nil, nil
}

// AddTemplate adds a new template
func (a *api) AddTemplate(reader io.Reader) (*template.Template, error) {
	return nil, nil
}

// RemoveTemplate removes an existing template by its template ID
func (a *api) RemoveTemplate(id string) error {
	return nil
}

// CompileTemplate builds a template given a source path
func (a *api) CompileTemplate(path string) (io.Reader, error) {
	return nil, nil
}

// DeployTemplate deploys a template given its template ID
func (a *api) DeployTemplate(id string) error {
	return nil
}
