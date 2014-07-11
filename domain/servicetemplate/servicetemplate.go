// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package servicetemplate

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
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

func (a *ServiceTemplate) Hash() (string, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash), nil
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
	Description     string // Description
	Data            string // JSON encoded template definition
	APIVersion      int    // Version of the ServiceTemplate API this expects
	TemplateVersion int    // Version of the template
}

func newWrapper(st ServiceTemplate) (*serviceTemplateWrapper, error) {
	data, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}

	var wrapper serviceTemplateWrapper
	wrapper.ID = st.ID
	wrapper.Name = st.Name
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
		Services: []servicedefinition.ServiceDefinition{*sd},
		Name:     sd.Name,
	}
	return &st, nil
}
