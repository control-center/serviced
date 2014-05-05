/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package servicedefinition_test

import (
	"github.com/zenoss/serviced/commons"
	. "github.com/zenoss/serviced/domain/servicedefinition"
	. "github.com/zenoss/serviced/domain/servicedefinition/testutils"

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

func TestServiceDefinition(t *testing.T) {
	sd := ServiceDefinition{}
	err := sd.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNormalizeLaunch(t *testing.T) {
	sd := ServiceDefinition{}
	//explicitly zeroing out for test
	sd.Launch = ""
	err := sd.NormalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.AUTO {
		t.Errorf("Expected %v Launch field, found: %v", commons.AUTO, sd.Launch)
	}
	sd.Launch = commons.AUTO
	err = sd.NormalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.AUTO {
		t.Errorf("Expected %v Launch field, found: %v", commons.AUTO, sd.Launch)
	}
	//Test case insensitive
	sd.Launch = "AutO"
	err = sd.NormalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.AUTO {
		t.Errorf("Expected %v Launch field, found: %v", commons.AUTO, sd.Launch)
	}
	sd.Launch = commons.MANUAL
	err = sd.NormalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.MANUAL {
		t.Errorf("Expected %v Launch field, found: %v", commons.MANUAL, sd.Launch)
	}
	//Test case insensitive
	sd.Launch = "ManUaL"
	err = sd.NormalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.MANUAL {
		t.Errorf("Expected %v Launch field, found: %v", commons.MANUAL, sd.Launch)
	}

	sd.Launch = "unknown"
	err = sd.NormalizeLaunch()
	if sd.Launch != "unknown" {
		t.Errorf("Expected  Launch field to be unmodified, found %v", sd.Launch)
	}
	if err == nil {
		t.Error("Expected error normalizing Launch field")
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
