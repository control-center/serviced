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

package servicedefinition_test

import (
	"github.com/control-center/serviced/commons"
	. "github.com/control-center/serviced/domain/servicedefinition"
	. "github.com/control-center/serviced/domain/servicedefinition/testutils"

	"strings"
	"testing"
)

func TestServiceDefinitionValidate(t *testing.T) {
	sd := *ValidSvcDef
	err := sd.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
func TestNormalizeLaunch(t *testing.T) {
	sd := ServiceDefinition{}
	//explicitly zeroing out for test
	sd.Launch = ""
	sd.NormalizeLaunch()
	if sd.Launch != commons.AUTO {
		t.Errorf("Expected %v Launch field, found: %v", commons.AUTO, sd.Launch)
	}
	sd.Launch = commons.AUTO
	sd.NormalizeLaunch()
	if sd.Launch != commons.AUTO {
		t.Errorf("Expected %v Launch field, found: %v", commons.AUTO, sd.Launch)
	}
	//Test case insensitive
	sd.Launch = "AutO"
	sd.NormalizeLaunch()
	if sd.Launch != commons.AUTO {
		t.Errorf("Expected %v Launch field, found: %v", commons.AUTO, sd.Launch)
	}
	sd.Launch = commons.MANUAL
	if sd.Launch != commons.MANUAL {
		t.Errorf("Expected %v Launch field, found: %v", commons.MANUAL, sd.Launch)
	}
	//Test case insensitive
	sd.Launch = "ManUaL"
	sd.NormalizeLaunch()
	if sd.Launch != commons.MANUAL {
		t.Errorf("Expected %v Launch field, found: %v", commons.MANUAL, sd.Launch)
	}

	sd.Launch = "unknown"
	sd.NormalizeLaunch()
	if sd.Launch != "unknown" {
		t.Errorf("Expected  Launch field to be unmodified, found %v", sd.Launch)
	}

}

func TestValidateAddressResourcConfigPorts(t *testing.T) {
	// test port validation
	arc := AddressResourceConfig{}
	err := arc.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected error normalizing AddressResourceConfig %v", err)
	}
	arc.Protocol = commons.TCP
	err = arc.ValidEntity()
	if err == nil || !strings.Contains(err.Error(), "AddressResourceConfig: not in valid port range: 0") {
		t.Errorf("Expected error for 0 port value, got %v", err)
	}

	//explicit set to 0
	arc.Port = 0
	err = arc.ValidEntity()
	if err == nil || !strings.Contains(err.Error(), "AddressResourceConfig: not in valid port range: 0") {
		t.Error("Expected error for 0 port value")
	}

	//	//test valid ports 1-65535
	//	for i, _ := range make([]interface{}, 65535) {
	//		arc.Port = uint16(i + 1) // index is 0 based
	//		err = arc.ValidEntity()
	//		if err != nil {
	//			t.Errorf("Unexpected error normalizing AddressResourceConfig %v", err)
	//		}
	//	}
}

func TestNormalizeAddressResourcConfigProtocol(t *testing.T) {
	// test port validation
	arc := AddressResourceConfig{}
	err := arc.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected error normalizing AddressResourceConfig %v", err)
	}
	//set valid port for the rest of this test
	arc.Port = 100

	arc.Protocol = "blam"
	err = arc.ValidEntity()
	if err == nil || !strings.Contains(err.Error(), "string blam not in [tcp udp]") {
		t.Errorf("Expected error for invalid protocol %v", err)
	}

	validProtocols := []string{"tcp", "TCP", "tcP", "udp", "UDP", "uDp", commons.TCP, commons.UDP}
	for _, protocol := range validProtocols {
		arc.Protocol = protocol
		arc.Normalize()
		if err = arc.ValidEntity(); err != nil {
			t.Errorf("Unexpected error for protocol %v", err)
		}
	}

}

func TestServiceDefinitionEmptyEndpointName(t *testing.T) {
	sd := CreateValidServiceDefinition()
	sd.Services[0].Endpoints[0].Name = ""

	err := sd.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "endpoint must have a name") {
		t.Errorf("Unexpected Error %v", err)
	}
}
