/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package svc

import (
	"github.com/zenoss/serviced"
	"testing"
)

func levelZeroServiceDefinitionZeroCommand(t serviced.ServiceTemplate) string {
	return t.Services[0].Command
}

func levelOneServiceDefinitionZeroCommand(t serviced.ServiceTemplate) string {
	return t.Services[0].Services[0].Command
}

var injectionTests = []struct {
	template serviced.ServiceTemplate
	expected string
	resfunc  func(serviced.ServiceTemplate) string
}{
	{serviced.ServiceTemplate{"Context Free",
		"A context free template",
		map[string]interface{}{},
		[]serviced.ServiceDefinition{{"ls",
			"ls",
			"Run ls",
			"test/ls",
			serviced.MinMax{1, 0},
			"auto",
			[]serviced.ServiceEndpoint{},
			[]serviced.ServiceDefinition{}}},
	}, "ls", levelZeroServiceDefinitionZeroCommand},
	{serviced.ServiceTemplate{"No Substitutions",
		"No substitutions",
		map[string]interface{}{"City": "Austin", "State": "Texas"},
		[]serviced.ServiceDefinition{{"bash",
			"/bin/bash",
			"Run bash",
			"test/bash",
			serviced.MinMax{1, 0},
			"auto",
			[]serviced.ServiceEndpoint{},
			[]serviced.ServiceDefinition{}}},
	}, "/bin/bash", levelZeroServiceDefinitionZeroCommand},
	{serviced.ServiceTemplate{"Single Substitution",
		"A single substitution template",
		map[string]interface{}{"Command": "/bin/sh"},
		[]serviced.ServiceDefinition{{"/bin/sh",
			"{{.Command}}",
			"Run /bin/sh",
			"test/single",
			serviced.MinMax{1, 0},
			"auto",
			[]serviced.ServiceEndpoint{},
			[]serviced.ServiceDefinition{}}},
	}, "/bin/sh", levelZeroServiceDefinitionZeroCommand},
	{serviced.ServiceTemplate{"Multiple Substitution",
		"A multi-substitution template",
		map[string]interface{}{"RemoteHost": "zenoss.com", "Count": 32},
		[]serviced.ServiceDefinition{{"pinger",
			"/usr/bin/ping -c {{.Count}} {{.RemoteHost}}",
			"Ping a remote host a fixed number of times",
			"test/pinger",
			serviced.MinMax{1, 0},
			"auto",
			[]serviced.ServiceEndpoint{},
			[]serviced.ServiceDefinition{}}},
	}, "/usr/bin/ping -c 32 zenoss.com", levelZeroServiceDefinitionZeroCommand},
	{serviced.ServiceTemplate{"Subservice Injection",
		"Inject context in a subservice",
		map[string]interface{}{"RemoteHost": "zenoss.com", "Count": 64},
		[]serviced.ServiceDefinition{{"shell",
			"/bin/bash",
			"Bash shell",
			"test/subservice",
			serviced.MinMax{1, 0},
			"auto",
			[]serviced.ServiceEndpoint{},
			[]serviced.ServiceDefinition{{"pinger",
				"/usr/bin/ping -c {{.Count}} {{.RemoteHost}}",
				"Ping a remote host a fixed number of times",
				"test/pinger",
				serviced.MinMax{1, 0},
				"auto",
				[]serviced.ServiceEndpoint{},
				[]serviced.ServiceDefinition{}}},
		}},
	}, "/usr/bin/ping -c 64 zenoss.com", levelOneServiceDefinitionZeroCommand},
	{serviced.ServiceTemplate{"Subservice Injection",
		"Inject context in a subservice",
		map[string]interface{}{"User": "scott", "Password": "tiger", "Database": "demo", "RemoteHost": "zenoss.com", "Count": 64},
		[]serviced.ServiceDefinition{{"database creation",
			"mysqladmin -u {{.User}} -P {{.Password}} create {{.Database}}",
			"create a database",
			"test/subservice",
			serviced.MinMax{1, 0},
			"auto",
			[]serviced.ServiceEndpoint{},
			[]serviced.ServiceDefinition{{"pinger",
				"/usr/bin/ping -c {{.Count}} {{.RemoteHost}}",
				"Ping a remote host a fixed number of times",
				"test/pinger",
				serviced.MinMax{1, 0},
				"auto",
				[]serviced.ServiceEndpoint{},
				[]serviced.ServiceDefinition{}}},
		}},
	}, "mysqladmin -u scott -P tiger create demo", levelZeroServiceDefinitionZeroCommand},
	{serviced.ServiceTemplate{"Subservice Injection",
		"Inject context in a subservice",
		map[string]interface{}{"User": "scott", "Password": "tiger", "Database": "demo", "RemoteHost": "zenoss.com", "Count": 64},
		[]serviced.ServiceDefinition{{"database creation",
			"mysqladmin -u {{.User}} -P {{.Password}} create {{.Database}}",
			"create a database",
			"test/subservice",
			serviced.MinMax{1, 0},
			"auto",
			[]serviced.ServiceEndpoint{},
			[]serviced.ServiceDefinition{{"pinger",
				"/usr/bin/ping -c {{.Count}} {{.RemoteHost}}",
				"Ping a remote host a fixed number of times",
				"test/pinger",
				serviced.MinMax{1, 0},
				"auto",
				[]serviced.ServiceEndpoint{},
				[]serviced.ServiceDefinition{}}},
		}},
	}, "/usr/bin/ping -c 64 zenoss.com", levelOneServiceDefinitionZeroCommand},
}

func TestContextInjection(t *testing.T) {
	for _, it := range injectionTests {
		if err := InjectContext(&it.template); err != nil {
			t.Error(err)
		}

		result := it.resfunc(it.template)

		if result != it.expected {
			t.Errorf("Expecting %s got %s\n", result, it.expected)
		}
	}
}

func TestMultipleInjections(t *testing.T) {
	template := serviced.ServiceTemplate{"Subservice Injection",
		"Inject context in a subservice",
		map[string]interface{}{"User": "scott", "Password": "tiger", "Port": 3006, "Database": "demo", "RemoteHost": "zenoss.com", "Count": 64},
		[]serviced.ServiceDefinition{{"database creation",
			"mysqladmin -u {{.User}} -P {{.Password}} create {{.Database}}",
			"create a database",
			"test/subservice",
			serviced.MinMax{1, 0},
			"auto",
			[]serviced.ServiceEndpoint{},
			[]serviced.ServiceDefinition{{"telnet",
				"telnet {{.RemoteHost}} {{.Port}}",
				"telnet to a remote host on the given port",
				"test/telnet",
				serviced.MinMax{1, 0},
				"auto",
				[]serviced.ServiceEndpoint{},
				[]serviced.ServiceDefinition{}}, {"pinger",
				"/usr/bin/ping -c {{.Count}} {{.RemoteHost}}",
				"Ping a remote host a fixed number of times",
				"test/pinger",
				serviced.MinMax{1, 0},
				"auto",
				[]serviced.ServiceEndpoint{},
				[]serviced.ServiceDefinition{}}},
		}}}

	if err := InjectContext(&template); err != nil {
		t.Error(err)
	}

	result := template.Services[0].Command
	if result != "mysqladmin -u scott -P tiger create demo" {
		t.Errorf("Expecting %s got %s\n", "mysqladmin -u scott -P tiger create demo", result)
	}

	result = template.Services[0].Services[0].Command
	if result != "telnet zenoss.com 3006" {
		t.Errorf("Expecting %s got %s\n", "telnet zenoss.com 3006", result)
	}

	result = template.Services[0].Services[1].Command
	if result != "/usr/bin/ping -c 64 zenoss.com" {
		t.Errorf("Expecting %s got %s\n", "/usr/bin/ping -c 64 zenoss.com", result)
	}
}

func TestIncompleteInjection(t *testing.T) {
	template := serviced.ServiceTemplate{"Multiple Substitution",
		"A multi-substitution template",
		map[string]interface{}{"RemoteHost": "zenoss.com"},
		[]serviced.ServiceDefinition{{"pinger",
			"/usr/bin/ping -c {{.Count}} {{.RemoteHost}}",
			"Ping a remote host a fixed number of times",
			"test/pinger",
			serviced.MinMax{1, 0},
			"auto",
			[]serviced.ServiceEndpoint{},
			[]serviced.ServiceDefinition{}}},
	}

	if err := InjectContext(&template); err != nil {
		t.Error(err)
	}

	result := template.Services[0].Command

	if result == "/usr/bin/ping -c 64 zenoss.com" {
		t.Errorf("Not expecting a match")
	}
}
