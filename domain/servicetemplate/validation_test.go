// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package servicetemplate

import (
	"github.com/zenoss/serviced/domain/servicedefinition"
	. "github.com/zenoss/serviced/domain/servicedefinition/testutils"

	"strings"
	"testing"
)

func TestServiceTemplateValidate(t *testing.T) {
	template := ServiceTemplate{}
	template.ID = "test_id"
	template.Services = []servicedefinition.ServiceDefinition{*ValidSvcDef}
	err := template.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestServiceTemplateValidateNoServiceDefinitions(t *testing.T) {
	template := ServiceTemplate{}
	template.ID = "test_id"
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
	} else if !strings.Contains(err.Error(), "duplicate vhost found: testhost; ServiceDefintion s2") {
		t.Errorf("Unexpected Error %v", err)
	}
}
