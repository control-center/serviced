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

package service

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"
)

// set up template function definitions
var funcs = template.FuncMap{
	"plus": func(a, b int) int { return a + b },
}

type TemplateError struct {
	Application string
	Field       string
	Message     string
}

func (err TemplateError) Error() string {
	return fmt.Sprintf("could not parse '%s' for %s: %s", err.Application, err.Field, err.Message)
}

// ImportBinding describes an import endpoint
type ImportBinding struct {
	Application    string
	Purpose        string // import or import_all
	PortNumber     uint16
	PortTemplate   string
	VirtualAddress string
}

// GetPortNumber retrieves a port number for a given instance ID
func (i ImportBinding) GetPortNumber(instanceID int) (uint16, error) {
	t := template.Must(template.New(i.Application).Funcs(funcs).Parse(i.PortTemplate))

	var buffer bytes.Buffer
	if err := t.Execute(&buffer, struct{ InstanceID int }{instanceID}); err != nil {
		return 0, &TemplateError{
			Application: i.Application,
			Field:       i.PortTemplate,
			Message:     "could not interpret value",
		}
	}

	if port := buffer.String(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil {
			return 0, &TemplateError{
				Application: i.Application,
				Field:       i.PortTemplate,
				Message:     "port value is not an integer",
			}
		} else if portNumber < 0 {
			return 0, &TemplateError{
				Application: i.Application,
				Field:       i.PortTemplate,
				Message:     "port value must be gte zero",
			}
		}
		return uint16(portNumber), nil
	}

	return i.PortNumber, nil
}

// GetVirtualAddress retrieves the virtual address for a given instance ID
func (i ImportBinding) GetVirtualAddress(instanceID int) (string, error) {
	t := template.Must(template.New(i.Application).Funcs(funcs).Parse(i.VirtualAddress))

	var buffer bytes.Buffer
	if err := t.Execute(&buffer, struct{ InstanceID int }{instanceID}); err != nil {
		return "", &TemplateError{
			Application: i.Application,
			Field:       i.VirtualAddress,
			Message:     "could not interpret value",
		}
	}

	return buffer.String(), nil
}

// ExportBinding describes an export endpoint
type ExportBinding struct {
	Application        string
	Protocol           string
	PortNumber         uint16
	AssignedPortNumber uint16
}
