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

// +build integration

package facade

import (
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/zenoss/glog"
	. "gopkg.in/check.v1"
)

func (ft *FacadeIntegrationTest) TestFacadeServiceTemplate(c *C) {
	glog.V(0).Infof("TestFacadeServiceTemplate started")
	defer glog.V(0).Infof("TestFacadeServiceTemplate finished")

	var (
		e             error
		templateId    string
		newTemplateId string
		templates     map[string]servicetemplate.ServiceTemplate
	)

	// Clean up old templates...
	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	c.Assert(e, IsNil)
	for id, _ := range templates {
		e := ft.Facade.RemoveServiceTemplate(ft.CTX, id)
		c.Assert(e, IsNil)
	}

	template := servicetemplate.ServiceTemplate{
		ID:          "",
		Name:        "test_template",
		Description: "test template",
	}

	newTemplateId, e = ft.Facade.AddServiceTemplate(ft.CTX, template, false)
	c.Assert(e, IsNil)
	templateId = newTemplateId

	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	c.Assert(e, IsNil)
	c.Assert(len(templates), Equals, 1)

	_, ok := templates[templateId]
	c.Assert(ok, Equals, true)
	c.Assert(templates[templateId].Name, Equals, "test_template")

	template.ID = templateId
	template.Description = "test_template_modified"
	e = ft.Facade.UpdateServiceTemplate(ft.CTX, template, false)
	c.Assert(e, IsNil)

	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	c.Assert(e, IsNil)
	c.Assert(len(templates), Equals, 1)

	_, ok = templates[templateId]
	c.Assert(ok, Equals, true)
	c.Assert(templates[templateId].Name, Equals, "test_template")
	c.Assert(templates[templateId].Description, Equals, "test_template_modified")

	e = ft.Facade.RemoveServiceTemplate(ft.CTX, templateId)
	c.Assert(e, IsNil)

	time.Sleep(1 * time.Second) // race condition. :(
	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	c.Assert(e, IsNil)
	c.Assert(len(templates), Equals, 0)

	e = ft.Facade.UpdateServiceTemplate(ft.CTX, template, false)
	c.Assert(e, IsNil)
	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	c.Assert(e, IsNil)
	c.Assert(len(templates), Equals, 1)

	_, ok = templates[templateId]
	c.Assert(ok, Equals, true)
	c.Assert(templates[templateId].Name, Equals, "test_template")
}

func (ft *FacadeIntegrationTest) TestFacadeValidServiceForStart(c *C) {
	testService := service.Service{
		ID:     "TestFacadeValidServiceForStart_ServiceID",
		PoolID: "default",
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Name:        "TestFacadeValidServiceForStart_EndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
				}),
		},
	}

	// add the resource pool (no permissions required)
	rp := pool.ResourcePool{ID: "default"}
	err := ft.Facade.AddResourcePool(ft.CTX, &rp)
	c.Assert(err, IsNil)

	err = ft.Facade.validateServiceStart(datastore.Get(), &testService)
	c.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacadeInvalidServiceForStart(c *C) {
	testService := service.Service{
		ID:     "TestFacadeInvalidServiceForStart_ServiceID",
		PoolID: "default",
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Name:        "TestFacadeInvalidServiceForStart_EndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
					AddressConfig: servicedefinition.AddressResourceConfig{
						Port:     8081,
						Protocol: commons.TCP,
					},
				}),
		},
	}

	// add the resource pool (no permissions required)
	rp := pool.ResourcePool{ID: "default"}
	err := ft.Facade.AddResourcePool(ft.CTX, &rp)
	c.Assert(err, IsNil)

	err = ft.Facade.validateServiceStart(datastore.Get(), &testService)
	c.Assert(err, NotNil)
}

func (ft *FacadeIntegrationTest) TestFacadeServiceTemplate_WithLogFilters(c *C) {
	var (
		err        error
		ok         bool
		templateId string
		logFilter  *logfilter.LogFilter
	)

	template := servicetemplate.ServiceTemplate{
		ID:          "",
		Name:        "template1",
		Description: "test template1",
		Version:     "1.0",
		Services: []servicedefinition.ServiceDefinition{
			servicedefinition.ServiceDefinition{
				Name:   "service1",
				Launch: "manual",
				LogFilters: map[string]string{
					"filter1": "original filter",
				},
			},
		},
	}

	// Phase 1 - add a template and verify its filter is added
	templateId, err = ft.Facade.AddServiceTemplate(ft.CTX, template, false)
	c.Assert(err, IsNil)

	logFilter, err = ft.Facade.logFilterStore.Get(ft.CTX, "filter1", template.Version)
	c.Assert(err, IsNil)
	c.Assert(logFilter, NotNil)
	c.Assert(logFilter.Name, Equals, "filter1")
	c.Assert(logFilter.Version, Equals, template.Version)
	c.Assert(logFilter.Filter, Equals, "original filter")

	// Phase 2 - update the template and verify its filters is added/updated
	templates, e := ft.Facade.GetServiceTemplates(ft.CTX)
	c.Assert(e, IsNil)
	c.Assert(len(templates), Not(Equals), 0)

	template, ok = templates[templateId]
	c.Assert(ok, Equals, true)

	template.Services[0].LogFilters["filter1"] = "updated filter"
	template.Services[0].LogFilters["filter2"] = "second filter"
	e = ft.Facade.UpdateServiceTemplate(ft.CTX, template, false)
	c.Assert(e, IsNil)
	ft.verifyLogFilters(c, template.Version)

	// Phase 3 - add a new version of the template and verify the older filters are unchanged
	newTemplate := template
	newTemplate.Version = "2.0"
	template.Services[0].LogFilters["filter1"] = "filter1 v2"
	template.Services[0].LogFilters["filter2"] = "filter2 v2"
	_, err = ft.Facade.AddServiceTemplate(ft.CTX, newTemplate, false)
	c.Assert(err, IsNil)
	ft.verifyLogFilters(c, "1.0")
	ft.verifyLogFilters(c, "2.0")

	err = ft.Facade.RemoveServiceTemplate(ft.CTX, templateId)
	c.Assert(err, IsNil)
	ft.verifyLogFilters(c, "2.0")

	// Verify that the filters remain after the template is removed
	logFilter, err2 := ft.Facade.logFilterStore.Get(ft.CTX, "filter1", "1.0")
	c.Assert(err2, IsNil)
	c.Assert(logFilter, NotNil)

	logFilter, err2 = ft.Facade.logFilterStore.Get(ft.CTX, "filter2", "1.0")
	c.Assert(err2, IsNil)
	c.Assert(logFilter, NotNil)
}

func (ft *FacadeIntegrationTest) verifyLogFilters(c *C, version string) {

	name1 := "filter1"
	filter1 := "updated filter"
	name2 := "filter2"
	filter2 := "second filter"
	if version == "2.0" {
		filter1 = "filter1 v2"
		filter2 = "filter2 v2"
	}

	logFilter, err := ft.Facade.logFilterStore.Get(ft.CTX, name1, version)
	c.Assert(err, IsNil)
	c.Assert(logFilter, NotNil)
	c.Assert(logFilter.Name, Equals, name1)
	c.Assert(logFilter.Version, Equals, version)
	c.Assert(logFilter.Filter, Equals, filter1)

	logFilter, err = ft.Facade.logFilterStore.Get(ft.CTX, name2, version)
	c.Assert(err, IsNil)
	c.Assert(logFilter, NotNil)
	c.Assert(logFilter.Name, Equals, name2)
	c.Assert(logFilter.Version, Equals, version)
	c.Assert(logFilter.Filter, Equals, filter2)
}
