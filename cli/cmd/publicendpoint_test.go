// Copyright 2016 The Serviced Authors.
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

package cmd

import (
	"fmt"
	"testing"

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
)

var DefaultTestPublicEndpointPorts = []servicedefinition.Port{
	servicedefinition.Port{PortAddr: ":22222", Enabled: true, UseTLS: true, Protocol: "https"},
	servicedefinition.Port{PortAddr: ":22223", Enabled: true, UseTLS: false, Protocol: "http"},
	servicedefinition.Port{PortAddr: ":22224", Enabled: true, UseTLS: true, Protocol: ""},
	servicedefinition.Port{PortAddr: ":22225", Enabled: false, UseTLS: false, Protocol: ""},
}

type PublicEndpointTest struct {
	api.API
	fail  bool
	ports []servicedefinition.Port
}

func (t ServiceAPITest) GetAllPublicEndpoints() ([]service.PublicEndpoint, error) {
	if t.errs["GetAllPubliceEndpoints"] != nil {
		return nil, t.errs["GetAllPubliceEndpoints"]
	}

	peps := []service.PublicEndpoint{
		service.PublicEndpoint{Application: "zproxy", Enabled: true, PortAddress: ":22222", ServiceID: "test-service-1", ServiceName: "Zenoss", Protocol: "https"},
		service.PublicEndpoint{Application: "zproxy", Enabled: true, PortAddress: ":22223", ServiceID: "test-service-1", ServiceName: "Zenoss", Protocol: "http"},
		service.PublicEndpoint{Application: "zproxy", Enabled: true, PortAddress: ":22224", ServiceID: "test-service-1", ServiceName: "Zenoss", Protocol: "other-tls"},
		service.PublicEndpoint{Application: "zproxy", Enabled: false, PortAddress: ":22225", ServiceID: "test-service-1", ServiceName: "Zenoss", Protocol: "other"},
	}

	return peps, nil
}

func (t ServiceAPITest) AddPublicEndpointPort(serviceID, endpointName, portAddr string,
	usetls bool, protocol string, isEnabled, restart bool) (*servicedefinition.Port, error) {
	if t.errs["AddPublicEndpointPort"] != nil {
		return nil, t.errs["AddPublicEndpointPort"]
	}
	return &servicedefinition.Port{PortAddr: portAddr, Enabled: isEnabled, UseTLS: usetls, Protocol: protocol}, nil
}

func (t ServiceAPITest) RemovePublicEndpointPort(serviceID, endpointName, portAddr string) error {
	if t.errs["RemovePublicEndpointPort"] != nil {
		return t.errs["RemovePublicEndpointPort"]
	}
	return nil
}

func (t ServiceAPITest) EnablePublicEndpointPort(serviceID, endpointName, portAddr string, isEnabled bool) error {
	if t.errs["EnablePublicEndpointPort"] != nil {
		return t.errs["EnablePublicEndpointPort"]
	}
	return nil
}

func (t ServiceAPITest) AddPublicEndpointVHost(serviceid, endpointName, vhost string, isEnabled, restart bool) (*servicedefinition.VHost, error) {
	if t.errs["AddPublicEndpointVHost"] != nil {
		return nil, t.errs["AddPublicEndpointVHost"]
	}
	return &servicedefinition.VHost{Name: vhost, Enabled: isEnabled}, nil
}

func (t ServiceAPITest) RemovePublicEndpointVHost(serviceID, endpointName, vhost string) error {
	if t.errs["RemovePublicEndpointVHost"] != nil {
		return t.errs["RemovePublicEndpointVHost"]
	}
	return nil
}

func (t ServiceAPITest) EnablePublicEndpointVHost(serviceID, endpointName, vhost string, isEnabled bool) error {
	if t.errs["EnablePublicEndpointVHost"] != nil {
		return t.errs["EnablePublicEndpointVHost"]
	}
	return nil
}

func InitPublicEndpointPortTest(args ...string) {
	c := New(DefaultServiceAPITest, utils.TestConfigReader(make(map[string]string)), MockLogControl{})
	c.exitDisabled = true
	c.Run(args)
}

func ExampleServicedCLI_CmdPublicEndpointsList_usage(t *testing.T) {
	output := captureStdout(func() { InitSnapshotAPITest("serviced", "service", "public-endpoints", "list", "-h") })
	expected :=
		"NAME:\n" +
			"   list - Lists public endpoints for a service\n" +
			"\n" +
			"USAGE:\n" +
			"   command list [command options] [arguments...]\n" +
			"\n" +
			"DESCRIPTION:\n" +
			"   serviced service public-endpoints list [SERVICEID] [ENDPOINTNAME]\n" +
			"\n" +
			"OPTIONS:\n" +
			"   --ascii, -a                                                                  use ascii characters for service tree (env SERVICED_TREE_ASCII=1 will default to ascii)\n" +
			"   --ports                                                                      Show port public endpoints\n" +
			"   --vhosts                                                                     Show vhost public endpoints\n" +
			"   --show-fields 'Service,ServiceID,Endpoint,Type,Protocol,Name,Enabled'        Comma-delimited list describing which fields to display\n" +
			"   --verbose, -v                                                                Show JSON format"

	outStr := TrimLines(fmt.Sprintf("%s", output))
	expected = TrimLines(expected)

	if expected != outStr {
		t.Fatalf("\ngot:\n%s\nwant:\n%s", outStr, expected)
	}
}

func ExampleServicedCLI_CmdPublicEndpointsList_InvalidService() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "invalidservice")
	})

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdPublicEndpointsList() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list")

	// Output:
	// Service       ServiceID           Endpoint      Type      Protocol       Name        Enabled
	//   Zenoss      test-service-1      zproxy        port      https          :22222      true
	//   Zenoss      test-service-1      zproxy        port      http           :22223      true
	//   Zenoss      test-service-1      zproxy        port      other-tls      :22224      true
	//   Zenoss      test-service-1      zproxy        port      other          :22225      false
}

func ExampleServicedCLI_CmdPublicEndpointsList_endpointsByService() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "Zenoss")

	// Output:
	// Service       ServiceID           Endpoint      Type      Protocol       Name        Enabled
	//   Zenoss      test-service-1      zproxy        port      https          :22222      true
	//   Zenoss      test-service-1      zproxy        port      http           :22223      true
	//   Zenoss      test-service-1      zproxy        port      other-tls      :22224      true
	//   Zenoss      test-service-1      zproxy        port      other          :22225      false
}

func ExampleServicedCLI_CmdPublicEndpointsList_endpointsByServiceID() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "test-service-1")

	// Output:
	// Service       ServiceID           Endpoint      Type      Protocol       Name        Enabled
	//   Zenoss      test-service-1      zproxy        port      https          :22222      true
	//   Zenoss      test-service-1      zproxy        port      http           :22223      true
	//   Zenoss      test-service-1      zproxy        port      other-tls      :22224      true
	//   Zenoss      test-service-1      zproxy        port      other          :22225      false
}

func ExampleServicedCLI_CmdPublicEndpointsList_endpointByEndpointName() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "Zenoss", "zproxy")

	// Output:
	// Service       ServiceID           Endpoint      Type      Protocol       Name        Enabled
	//   Zenoss      test-service-1      zproxy        port      https          :22222      true
	//   Zenoss      test-service-1      zproxy        port      http           :22223      true
	//   Zenoss      test-service-1      zproxy        port      other-tls      :22224      true
	//   Zenoss      test-service-1      zproxy        port      other          :22225      false
}

func ExampleServicedCLI_CmdPublicEndpointsList_endpointTypePort() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "Zenoss", "zproxy", "--ports")

	// Output:
	// Service       ServiceID           Endpoint      Type      Protocol       Name        Enabled
	//   Zenoss      test-service-1      zproxy        port      https          :22222      true
	//   Zenoss      test-service-1      zproxy        port      http           :22223      true
	//   Zenoss      test-service-1      zproxy        port      other-tls      :22224      true
	//   Zenoss      test-service-1      zproxy        port      other          :22225      false
}

func ExampleServicedCLI_CmdPublicEndpointsList_endpointTypeVHost() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "Zenoss", "zproxy", "--vhosts")
	})

	// Output:
	// No public endpoints found
}

func ExampleServicedCLI_CmdPublicEndpointsList_endpointNoneFound() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "Zenoss", "zope")
	})

	// Output:
	// No public endpoints found
}

func ExampleServicedCLI_CmdPublicEndpointsList_endpointInvalid() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "Zenoss", "invalid")
	})

	// Output:
	// Endpoint 'invalid' not found
}

func ExampleServicedCLI_CmdPublicEndpointsList_endpoint_service1_fields() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "--show-fields", "Service,Name,Enabled", "Zenoss", "zproxy")

	// Output:
	// Service       Name        Enabled
	//   Zenoss      :22222      true
	//   Zenoss      :22223      true
	//   Zenoss      :22224      true
	//   Zenoss      :22225      false
}

func ExampleServicedCLI_CmdPublicEndpointsList_endpoint_service1_verbose() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "list", "Zenoss", "zproxy", "--show-fields", "'Service,Name,Enabled'", "-v")

	// Output:
	// [
	//    {
	//      "Service": "Zenoss",
	//      "ServiceID": "test-service-1",
	//      "Application": "zproxy",
	//      "EpType": "port",
	//      "Protocol": "https",
	//      "Name": ":22222",
	//      "Enabled": true
	//    },
	//    {
	//      "Service": "Zenoss",
	//      "ServiceID": "test-service-1",
	//      "Application": "zproxy",
	//      "EpType": "port",
	//      "Protocol": "http",
	//      "Name": ":22223",
	//      "Enabled": true
	//    },
	//    {
	//      "Service": "Zenoss",
	//      "ServiceID": "test-service-1",
	//      "Application": "zproxy",
	//      "EpType": "port",
	//      "Protocol": "other-tls",
	//      "Name": ":22224",
	//      "Enabled": true
	//    },
	//    {
	//      "Service": "Zenoss",
	//      "ServiceID": "test-service-1",
	//      "Application": "zproxy",
	//      "EpType": "port",
	//      "Protocol": "other",
	//      "Name": ":22225",
	//      "Enabled": false
	//    }
	//  ]
}

func ExampleServicedCLI_CmdPublicEndpointsPortAdd() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "add", "Zenoss", "zproxy", ":22222", "http", "true")

	// Output:
	// :22222
}

func ExampleServicedCLI_CmdPublicEndpointsPortAdd_InvalidEnable() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "add", "Zenoss", "zproxy", ":22222", "http", "invalid")
	})

	// Output:
	// The enabled flag must be true or false
}

func ExampleServicedCLI_CmdPublicEndpointsPortAdd_InvalidProtocol() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "add", "Zenoss", "zproxy", ":22222", "invalid", "true")
	})

	// Output:
	// The protocol must be one of: https, http, other-tls, other
}

func ExampleServicedCLI_CmdPublicEndpointsPortAdd_ValidProtocol() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "add", "Zenoss", "zproxy", ":22222", "http", "true")
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "add", "Zenoss", "zproxy", ":22222", "https", "true")
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "add", "Zenoss", "zproxy", ":22222", "other", "true")
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "add", "Zenoss", "zproxy", ":22222", "other-tls", "true")

	// Output:
	// :22222
	// :22222
	// :22222
	// :22222
}

func ExampleServicedCLI_CmdPublicEndpointsPortRemove() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "remove", "Zenoss", "zproxy", ":22222")

	// Output:
	// :22222
}

func ExampleServicedCLI_CmdPublicEndpointsPortEnable_InvalidArgCount() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "enable", "Zenoss", "zproxy", ":22222", "true", "invalid")
	})

	// Output:
	// NAME:
	//    enable - Enable/Disable a port public endpoint for a service
	//
	// USAGE:
	//    command enable [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service public-endpoints port enable <SERVICEID> <ENDPOINTNAME> <PORTADDR> true|false
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdPublicEndpointsPortEnable_InvalidService() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "enable", "invalid", "zproxy", ":22222", "true")
	})

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdPublicEndpointsPortEnable_InvalidEnableFlag() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "enable", "Zenoss", "zproxy", ":22222", "invalid")
	})

	// Output:
	// The enabled flag must be true or false
}

func ExampleServicedCLI_CmdPublicEndpointsPortEnable_ValidFlags() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "enable", "Zenoss", "zproxy", ":22222", "true")
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "enable", "Zenoss", "zproxy", ":22222", "false")

	// Output:
	// :22222
	// :22222
}

func ExampleServicedCLI_cmdPublicEndpointsVHostAdd() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "vhost", "add", "Zenoss", "zproxy", "zproxy2", "true")

	// Output:
	// zproxy2
}

func ExampleServicedCLI_cmdPublicEndpointsVHostAdd_InvalidArgCount() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "vhost", "add", "Zenoss", "zproxy", "zproxy2", "true", "invalid")

	// Output:
	// NAME:
	//    add - Add a vhost public endpoint to a service
	//
	// USAGE:
	//    command add [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service public-endpoints vhost add <SERVICEID> <ENDPOINTNAME> <VHOST> <ENABLED>
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_cmdPublicEndpointsVHostAdd_InvalidService() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "vhost", "add", "invalid", "zproxy", "zproxy2", "true")
	})

	// Output:
	// service not found
}

func ExampleServicedCLI_cmdPublicEndpointsVHostAdd_InvalidEnableFlag() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "vhost", "add", "Zenoss", "invalid", "zproxy2", "invalid")
	})

	// Output:
	// The enabled flag must be true or false
}

func ExampleServicedCLI_CmdPublicEndpointsVHostRemove() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "vhost", "remove", "Zenoss", "zproxy", "zproxy")

	// Output:
	// zproxy
}

func ExampleServicedCLI_CmdPublicEndpointsVHostEnable_InvalidArgCount() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "vhost", "enable", "Zenoss", "zproxy", "zproxy", "true", "invalid")
	})

	// Output:
	// NAME:
	//    enable - Enable/Disable a vhost public endpoint for a service
	//
	// USAGE:
	//    command enable [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service public-endpoints vhost enable <SERVICEID> <ENDPOINTNAME> <VHOST> true|false
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdPublicEndpointsVHostEnable_InvalidService() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "vhost", "enable", "invalid", "zproxy", "zproxy", "true")
	})

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdPublicEndpointsVHostEnable_InvalidEnableFlag() {
	pipeStderr(func() {
		InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "vhost", "enable", "Zenoss", "zproxy", "zproxy", "invalid")
	})

	// Output:
	// The enabled flag must be true or false
}

func ExampleServicedCLI_CmdPublicEndpointsVHostEnable_ValidFlags() {
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "enable", "Zenoss", "zproxy", "zproxy", "true")
	InitPublicEndpointPortTest("serviced", "service", "public-endpoints", "port", "enable", "Zenoss", "zproxy", "zproxy", "false")

	// Output:
	// zproxy
	// zproxy
}
