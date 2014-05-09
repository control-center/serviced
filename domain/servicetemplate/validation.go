// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package servicetemplate

import (
	"github.com/zenoss/serviced/validation"

	//	"strings"
	"fmt"
	"github.com/zenoss/serviced/domain/servicedefinition"
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
			for _, vhost := range ep.VHosts {
				if _, found := vhosts[vhost]; found {
					return fmt.Errorf("duplicate vhost found: %s; ServiceDefintion %s", vhost, sd)
				}
				vhosts[vhost] = struct{}{}
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
