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
	"strings"
)

var validSvcDef *ServiceDefinition

func init() {
	// should we make the service definition from the dao test package public and use here?
	validSvcDef = createValidServiceDefinition()
}

func createValidServiceDefinition() *ServiceDefinition {
	// should we make the service definition from the dao test package public and use here?
	return &ServiceDefinition{
		Name: "testsvc",
		Services: []ServiceDefinition{
			ServiceDefinition{
				Name: "s1",
				Endpoints: []ServiceEndpoint{
					ServiceEndpoint{
						Name:        "www",
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "www",
						Purpose:     "export",
					},
					ServiceEndpoint{
						Name:        "websvc",
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
						Name:        "websvc",
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

func TestServiceDefinitionDuplicateEndpoint(t *testing.T) {
	sd := createValidServiceDefinition()
	sd.Services[0].Endpoints[0].Name = "websvc"
	template := ServiceTemplate{}
	template.Services = []ServiceDefinition{*sd}

	err := template.Validate()
	if err == nil {
		t.Error("Expected error")
	} else if err.Error() != "Service Definition s1: Endpoint name websvc not unique in service definition" {
		t.Errorf("Unexpected Error %v", err)
	}
}

func TestServiceDefinitionEmpyEndpointName(t *testing.T) {
	sd := createValidServiceDefinition()
	sd.Services[0].Endpoints[0].Name = ""
	template := ServiceTemplate{}
	template.Services = []ServiceDefinition{*sd}

	err := template.Validate()
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(),"Endpoint must have a name") {
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

	//explicit set to 0
	arc.Port = 0
	err = arc.normalize()
	if err == nil || err.Error() != "AddressResourceConfig: Invalid port number 0" {
		t.Error("Expected error for 0 port value")
	}

	//test valid ports 1-65535
	for i, _ := range make([]interface{}, 65535) {
		arc.Port = uint16(i + 1) // index is 0 based
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

func TestAddressAssignmentValidate(t *testing.T) {
	aa := AddressAssignment{}
	err := aa.Validate()
	if err == nil {
		t.Error("Expected error")
	}

	// valid static assignment
	aa = AddressAssignment{"id", "static", "hostid", "", "ipaddr", 100, "serviceid", "endpointname"}
	err = aa.Validate()
	if err != nil {
		t.Errorf("Unexpected Error %v", err)
	}

	// valid Virtual assignment
	aa = AddressAssignment{"id", "virtual", "", "poolid", "ipaddr", 100, "serviceid", "endpointname"}
	err = aa.Validate()
	if err != nil {
		t.Errorf("Unexpected Error %v", err)
	}

	//Some error cases
	// no pool id when virtual
	aa = AddressAssignment{"id", "virtual", "hostid", "", "ipaddr", 100, "serviceid", "endpointname"}
	err = aa.Validate()
	if err == nil {
		t.Error("Expected error")
	}

	// no host id when static
	aa = AddressAssignment{"id", "static", "", "poolid", "ipaddr", 100, "serviceid", "endpointname"}
	err = aa.Validate()
	if err == nil {
		t.Error("Expected error")
	}

	// no type
	aa = AddressAssignment{"id", "", "hostid", "poolid", "ipaddr", 100, "serviceid", "endpointname"}
	err = aa.Validate()
	if err == nil {
		t.Error("Expected error")
	}

	// no ip
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "", 100, "serviceid", "endpointname"}
	err = aa.Validate()
	if err == nil {
		t.Error("Expected error")
	}

	// no port
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "ipaddr", 0, "serviceid", "endpointname"}
	err = aa.Validate()
	if err == nil {
		t.Error("Expected error")
	}

	// no serviceid
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "ipaddr", 80, "", "endpointname"}
	err = aa.Validate()
	if err == nil {
		t.Error("Expected error")
	}

	// no endpointname
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "ipaddr", 80, "svcid", ""}
	err = aa.Validate()
	if err == nil {
		t.Error("Expected error")
	}
}
