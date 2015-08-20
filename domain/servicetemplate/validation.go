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
	"github.com/control-center/serviced/validation"

	//	"strings"
	"fmt"
	"github.com/control-center/serviced/domain/servicedefinition"
)

// ValidEntity ensure that a ServiceTemplate has valid values
func (st *ServiceTemplate) ValidEntity() error {
	//	trimmedID := strings.TrimSpace(st.ID)
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("ServiceTemplate.ID", st.ID))
	//	violations.Add(validation.StringsEqual(st.ID, trimmedID, "leading and trailing spaces not allowed for service template id"))

	//TODO: check name, description, config files.
	//TODO: do servicedefinition names need to be unique?

	//TODO: Is there any special validation if more than one top level service definition?
	for _, sd := range st.Services {
		if err := sd.ValidEntity(); err != nil {
			violations.Add(err)
		}
	}

	//keep track of seen vhosts
	vhosts := make(map[string]struct{})

	//grab the vhost from every endpoing
	visit := func(sd *servicedefinition.ServiceDefinition) error {
		for _, ep := range sd.Endpoints {
			for _, vhost := range ep.VHostList {
				if _, found := vhosts[vhost.Name]; found {
					return fmt.Errorf("duplicate vhost found: %s; ServiceDefintion %s", vhost.Name, sd)
				}
				vhosts[vhost.Name] = struct{}{}
			}
		}
		return nil
	}

	for _, sd := range st.Services {
		violations.Add(servicedefinition.Walk(&sd, visit))
	}

	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}

//ValidEntity makes sure all serviceTemplateWrapper have non-empty values
func (st *serviceTemplateWrapper) ValidEntity() error {

	v := validation.NewValidationError()
	v.Add(validation.NotEmpty("ID", st.ID))
	v.Add(validation.NotEmpty("Name", st.Name))
	v.Add(validation.NotEmpty("Data", st.Data))
	if v.HasError() {
		return v
	}
	return nil
}
