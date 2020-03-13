// Copyright 2017 The Serviced Authors.
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

// +build integration

package facade

import (
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	. "gopkg.in/check.v1"
)

// Add a service template with no log filters
//	run bootstrap
//	Expect false returned and no log filters exist
func (ft *IntegrationTest) TestFacade_LogFilterBootstrap_NoFilters(c *C) {
	var (
		result bool
		err    error
	)

	template := servicetemplate.ServiceTemplate{
		ID:          "",
		Name:        "bootstrap template1",
		Description: "test bootstrap template1",
		Version:     "1.0",
		Services: []servicedefinition.ServiceDefinition{
			servicedefinition.ServiceDefinition{
				Name:   "service1",
				Launch: "manual",
			},
		},
	}

	_, err = ft.Facade.AddServiceTemplate(ft.CTX, template, false)
	c.Assert(err, IsNil)

	result, err = ft.Facade.BootstrapLogFilters(ft.CTX)
	c.Assert(err, IsNil)
	c.Assert(result, Equals, false)

	_, err = ft.Facade.logfilterStore.Get(ft.CTX, "filter1", template.Version)
	c.Assert(err, ErrorMatches, "No such entity.*")
}

// Add a service template with log filters
//	run bootstrap
//	Expect false returned and log filters exist
func (ft *IntegrationTest) TestFacade_LogFilterBootstrap_ExistingFilters(c *C) {
	var (
		result bool
		err    error
	)
	template := servicetemplate.ServiceTemplate{
		ID:          "",
		Name:        "bootstrap template2",
		Description: "test bootstrap template2",
		Version:     "1.0",
		Services: []servicedefinition.ServiceDefinition{
			servicedefinition.ServiceDefinition{
				Name:   "service1",
				Launch: "manual",
				LogFilters: map[string]string{
					"filter1": "some filter",
				},
			},
		},
	}

	_, err = ft.Facade.AddServiceTemplate(ft.CTX, template, false)
	c.Assert(err, IsNil)

	result, err = ft.Facade.BootstrapLogFilters(ft.CTX)
	c.Assert(err, IsNil)
	c.Assert(result, Equals, false)

	_, err = ft.Facade.logfilterStore.Get(ft.CTX, "filter1", template.Version)
	c.Assert(err, IsNil)
}

// Add a service template with log filters
//	remove the filters to simulate older implementations
//	run bootstrap
//	Expect true returned and log filters exist
func (ft *IntegrationTest) TestFacade_LogFilterBootstrap_AddsFilters(c *C) {
	var (
		result bool
		err    error
	)
	template := servicetemplate.ServiceTemplate{
		ID:          "",
		Name:        "bootstrap template2",
		Description: "test bootstrap template2",
		Version:     "1.0",
		Services: []servicedefinition.ServiceDefinition{
			servicedefinition.ServiceDefinition{
				Name:   "service1",
				Launch: "manual",
				LogFilters: map[string]string{
					"filter1": "some filter",
				},
			},
		},
	}

	_, err = ft.Facade.AddServiceTemplate(ft.CTX, template, false)
	c.Assert(err, IsNil)

	err = ft.Facade.RemoveLogFilters(ft.CTX, &template)
	c.Assert(err, IsNil)

	// Sanity check to prove the filters were really removed so that we're sure that
	//   the following BootstrapLogFilters was responsible for adding the filters
	_, err = ft.Facade.logfilterStore.Get(ft.CTX, "filter1", template.Version)
	c.Assert(err, ErrorMatches, "No such entity.*")

	result, err = ft.Facade.BootstrapLogFilters(ft.CTX)
	c.Assert(err, IsNil)
	c.Assert(result, Equals, true)

	_, err = ft.Facade.logfilterStore.Get(ft.CTX, "filter1", template.Version)
	c.Assert(err, IsNil)
}
