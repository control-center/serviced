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

func (ft *FacadeIntegrationTest) TestFacadeServiceTemplate(t *C) {
	glog.V(0).Infof("TestFacadeServiceTemplate started")
	defer glog.V(0).Infof("TestFacadeServiceTemplate finished")

	var (
		templateId string
		templates  map[string]servicetemplate.ServiceTemplate
	)

	// Clean up old templates...
	var e error
	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	if e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	for id, _ := range templates {
		if e := ft.Facade.RemoveServiceTemplate(ft.CTX, id); e != nil {
			t.Fatalf("Failure removing service template %s with error: %s", id, e)
		}
	}

	template := servicetemplate.ServiceTemplate{
		ID:          "",
		Name:        "test_template",
		Description: "test template",
	}

	if newTemplateId, e := ft.Facade.AddServiceTemplate(ft.CTX, template); e != nil {
		t.Fatalf("Failure adding service template %+v with error: %s", template, e)
	} else {
		templateId = newTemplateId
	}

	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	if e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template. Found %d", len(templates))
	}
	if _, ok := templates[templateId]; !ok {
		t.Fatalf("Expected to find template that was added (%s), but did not.", templateId)
	}
	if templates[templateId].Name != "test_template" {
		t.Fatalf("Expected to find test_template. Found %s", templates[templateId].Name)
	}
	template.ID = templateId
	template.Description = "test_template_modified"
	if e := ft.Facade.UpdateServiceTemplate(ft.CTX, template); e != nil {
		t.Fatalf("Failure updating service template %+v with error: %s", template, e)
	}
	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	if e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template. Found %d", len(templates))
	}
	if _, ok := templates[templateId]; !ok {
		t.Fatalf("Expected to find template that was updated (%s), but did not.", templateId)
	}
	if templates[templateId].Name != "test_template" {
		t.Fatalf("Expected to find test_template. Found %s", templates[templateId].Name)
	}
	if templates[templateId].Description != "test_template_modified" {
		t.Fatalf("Expected template to be modified. It hasn't changed!")
	}
	if e := ft.Facade.RemoveServiceTemplate(ft.CTX, templateId); e != nil {
		t.Fatalf("Failure removing service template with error: %s", e)
	}
	time.Sleep(1 * time.Second) // race condition. :(
	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	if e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 0 {
		t.Fatalf("Expected zero templates. Found %d", len(templates))
	}
	if e := ft.Facade.UpdateServiceTemplate(ft.CTX, template); e != nil {
		t.Fatalf("Failure updating service template %+v with error: %s", template, e)
	}
	templates, e = ft.Facade.GetServiceTemplates(ft.CTX)
	if e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template. Found %d", len(templates))
	}
	if _, ok := templates[templateId]; !ok {
		t.Fatalf("Expected to find template that was updated (%s), but did not.", templateId)
	}
	if templates[templateId].Name != "test_template" {
		t.Fatalf("Expected to find test_template. Found %s", templates[templateId].Name)
	}
}

func (ft *FacadeIntegrationTest) TestFacadeValidServiceForStart(t *C) {
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
	if err != nil {
		t.Error("Services failed validation for starting: ", err)
	}
}

func (ft *FacadeIntegrationTest) TestFacadeInvalidServiceForStart(t *C) {
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
	if err == nil {
		t.Error("Services should have failed validation for starting...")
	}
}
