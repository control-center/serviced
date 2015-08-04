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

// +build unit

package servicetemplate

import (
	"github.com/control-center/serviced/domain/servicedefinition"
	. "github.com/control-center/serviced/domain/servicedefinition/testutils"

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
