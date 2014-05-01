/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package servicetemplate

import (
	"github.com/zenoss/serviced/domain/servicedefinition"
	. "github.com/zenoss/serviced/domain/servicedefinition/testutils"

	"strings"
	"testing"
)

func TestServiceTemplateValidate(t *testing.T) {
	template := ServiceTemplate{}
	template.Services = []servicedefinition.ServiceDefinition{*ValidSvcDef}
	err := template.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestServiceTemplateValidateNoServiceDefinitions(t *testing.T) {
	template := ServiceTemplate{}
	err := template.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestServiceTemplateValidateErrorVhost(t *testing.T) {
	template := ServiceTemplate{}
	template.Services = []servicedefinition.ServiceDefinition{*ValidSvcDef, *ValidSvcDef}
	err := template.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "ServiceDefintion s2, duplicate vhost found: testhost") {
		t.Errorf("Unexpected Error %v", err)
	}
}

func TestServiceDefinitionEmptyEndpointName(t *testing.T) {
	sd := CreateValidServiceDefinition()
	sd.Services[0].Endpoints[0].Name = ""
	template := ServiceTemplate{}
	template.Services = []servicedefinition.ServiceDefinition{*sd}

	err := template.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "Endpoint must have a name") {
		t.Errorf("Unexpected Error %v", err)
	}
}
