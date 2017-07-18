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
		e error
		templateId string
		newTemplateId string
		templates  map[string]servicetemplate.ServiceTemplate
	)

	// Clean up old templates...
	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	c.Assert(e, IsNil)
	for id, _ := range templates {
		e := ft.Facade.RemoveServiceTemplate(ft.CTX, id);
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
		ID: "TestFacadeValidServiceForStart_ServiceID",
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
	err := ft.Facade.validateServiceStart(datastore.Get(), &testService)
	c.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacadeInvalidServiceForStart(c *C) {
	testService := service.Service{
		ID: "TestFacadeInvalidServiceForStart_ServiceID",
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
	err := ft.Facade.validateServiceStart(datastore.Get(), &testService)
	c.Assert(err, IsNil)
}
