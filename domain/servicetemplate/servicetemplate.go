// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
package servicetemplate

import (
	"reflect"

	"github.com/zenoss/serviced/domain/servicedefinition"
)

// ServiceTemplate type to hold service definitions
type ServiceTemplate struct {
	ID          string                                  // Unique ID of this service template
	Name        string                                  // Name of service template
	Description string                                  // Meaningful description of service
	Services    []servicedefinition.ServiceDefinition   // Child services
	ConfigFiles map[string]servicedefinition.ConfigFile // Config file templates
}

// Equals checks the equality of two service templates
func (a *ServiceTemplate) Equals(b *ServiceTemplate) bool {
	if a.ID != b.ID {
		return false
	}
	if a.Name != b.Name {
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

// ServiceTemplateWrapper type for storing ServiceTemplates  TODO: no need to be public when CRUD moves hers
type ServiceTemplateWrapper struct {
	ID              string // Primary-key - Should match ServiceTemplate.ID
	Name            string // Name of top level service
	Description     string // Description
	Data            string // JSON encoded template definition
	ApiVersion      int    // Version of the ServiceTemplate API this expects
	TemplateVersion int    // Version of the template
}
