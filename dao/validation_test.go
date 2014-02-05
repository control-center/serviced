/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package dao

import (
	"github.com/zenoss/serviced/commons"

	"testing"
)

var validSvcDef *ServiceDefinition
var invalidSvcDef *ServiceDefinition

func init() {
	// should we make the service definition from the dao test package public and use here?
	validSvcDef = &ServiceDefinition{
		Name: "testsvc",
		Services: []ServiceDefinition{
			ServiceDefinition{
				Name: "s1",
				Endpoints: []ServiceEndpoint{
					ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "www",
						Purpose:     "export",
					},
					ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8081,
						Application: "websvc",
						Purpose:     "import",
						AddressConfig: AddressResourceConfig{
							Port:     8081,
							Protocol: commons.TCP,
						},
					},
				},
				LogConfigs: []LogConfig{
					LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
					},
				},
				Snapshot: SnapshotCommands{
					Pause:  "echo pause",
					Resume: "echo resume",
				},
			},
			ServiceDefinition{
				Name:    "s2",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageId: "ubuntu",
				Endpoints: []ServiceEndpoint{
					ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "websvc",
						Purpose:     "export",
						VHosts:      []string{"testhost"},
					},
				},
				LogConfigs: []LogConfig{
					LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
					},
				},
				Snapshot: SnapshotCommands{
					Pause:  "echo pause",
					Resume: "echo resume",
				},
			},
		},
	}
}

func TestServiceTemplateValidate(t *testing.T) {
	template := ServiceTemplate{}
	template.Services = []ServiceDefinition{*validSvcDef}
	err := template.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestServiceTemplateValidateNoServiceDefinitions(t *testing.T) {
	template := ServiceTemplate{}
	err := template.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestServiceTemplateValidateErrorVhost(t *testing.T) {
	template := ServiceTemplate{}
	template.Services = []ServiceDefinition{*validSvcDef, *validSvcDef}
	err := template.Validate()
	if err == nil {
		t.Error("Expected error")
	} else if err.Error() != "Service Definition s2: Duplicate Vhost found: testhost" {
		t.Errorf("Unexpected Error %v", err)
	}
}

func TestServiceDefinitionValidate(t *testing.T) {
	sd := *validSvcDef
	context := validationContext{make(map[string]ServiceEndpoint)}
	err := sd.validate(&context)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestServiceDefinition(t *testing.T) {
	sd := ServiceDefinition{}
	context := validationContext{make(map[string]ServiceEndpoint)}
	err := sd.validate(&context)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNormalizeLaunch(t *testing.T) {
	sd := ServiceDefinition{}
	//explicitly zeroing out for test
	sd.Launch = ""
	err := sd.normalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.AUTO {
		t.Errorf("Expected %v Launch field, found: %v", commons.AUTO, sd.Launch)
	}
	sd.Launch = commons.AUTO
	err = sd.normalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.AUTO {
		t.Errorf("Expected %v Launch field, found: %v", commons.AUTO, sd.Launch)
	}
	//Test case insensitive
	sd.Launch = "AutO"
	err = sd.normalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.AUTO {
		t.Errorf("Expected %v Launch field, found: %v", commons.AUTO, sd.Launch)
	}
	sd.Launch = commons.MANUAL
	err = sd.normalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.MANUAL {
		t.Errorf("Expected %v Launch field, found: %v", commons.MANUAL, sd.Launch)
	}
	//Test case insensitive
	sd.Launch = "ManUaL"
	err = sd.normalizeLaunch()
	if err != nil {
		t.Errorf("Unexpected error normalizing Launch field: %v", err)
	}
	if sd.Launch != commons.MANUAL {
		t.Errorf("Expected %v Launch field, found: %v", commons.MANUAL, sd.Launch)
	}

	sd.Launch = "unknown"
	err = sd.normalizeLaunch()
	if sd.Launch != "unknown" {
		t.Errorf("Expected  Launch field to be unmodified, found %v", sd.Launch)
	}
	if err == nil {
		t.Error("Expected error normalizing Launch field")
	}

}

func TestNormalizeAddressResourcConfigPorts(t *testing.T) {
	// test port validation
	arc := AddressResourceConfig{}
	err := arc.normalize()
	if err != nil {
		t.Errorf("Unexpected error normalizing AddressResourceConfig %v", err)
	}
	arc.Protocol = commons.TCP
	err = arc.normalize()
	if err == nil || err.Error() != "AddressResourceConfig: Invalid port number 0" {
		t.Error("Expected error for 0 port value")
	}

	arc.Port = -1
	err = arc.normalize()
	if err == nil || err.Error() != "AddressResourceConfig: Invalid port number -1" {
		t.Error("Expected error for -1 port value")
	}

	arc.Port = 65536
	err = arc.normalize()
	if err == nil || err.Error() != "AddressResourceConfig: Invalid port number 65536" {
		t.Error("Expected error for 65536 port value")
	}

	//test valid ports
	for i := 1; i <= 65535; i = i + 1 {
		arc.Port = i
		err = arc.normalize()
		if err != nil {
			t.Errorf("Unexpected error normalizing AddressResourceConfig %v", err)
		}
	}
}

func TestNormalizeAddressResourcConfigProtocol(t *testing.T) {
	// test port validation
	arc := AddressResourceConfig{}
	err := arc.normalize()
	if err != nil {
		t.Errorf("Unexpected error normalizing AddressResourceConfig %v", err)
	}
	//set valid port for the rest of this test
	arc.Port = 100

	arc.Protocol = "blam"
	err = arc.normalize()
	if err == nil || err.Error() != "AddressResourceConfig: Protocol must be one of tcp or udp; found blam" {
		t.Errorf("Expected error for invalid protocol %v", err)
	}

	validProtocols := []string{"tcp", "TCP", "tcP", "udp", "UDP", "uDp", commons.TCP, commons.UDP}
	for _, protocol := range validProtocols {
		arc.Protocol = protocol
		err = arc.normalize()
		if err != nil {
			t.Errorf("Unexpected error for protocol %v", err)
		}
	}

}

func TestMinMax(t *testing.T) {

	mm := MinMax{}
	//validate default
	err := mm.validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//same
	mm.Min, mm.Max = 1, 1
	err = mm.validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//0 to 100
	mm.Min, mm.Max = 0, 100
	err = mm.validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//min > 0
	mm.Min, mm.Max = 10, 0
	err = mm.validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//min > max
	mm.Min, mm.Max = 10, 5
	err = mm.validate()
	if err.Error() != "Minimum instances larger than maximum instances: Min=10; Max=5" {
		t.Errorf("Unexpected error: %v", err)
	}

	// negative min
	mm.Min, mm.Max = -1, 1
	err = mm.validate()
	if err.Error() != "Instances constraints must be positive: Min=-1; Max=1" {
		t.Errorf("Unexpected error: %v", err)
	}

	// negative max
	mm.Min, mm.Max = 1, -1
	err = mm.validate()
	if err.Error() != "Instances constraints must be positive: Min=1; Max=-1" {
		t.Errorf("Unexpected error: %v", err)
	}

	// negative min and max
	mm.Min, mm.Max = -10, -10
	err = mm.validate()
	if err.Error() != "Instances constraints must be positive: Min=-10; Max=-10" {
		t.Errorf("Unexpected error: %v", err)
	}

}
