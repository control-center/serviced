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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/zenoss/glog"
	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/health"
)

var startup_testcases = []struct {
	service  Service
	expected string
}{
	{Service{
		ID:              "0",
		Name:            "Zenoss",
		Context:         nil,
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       0,
		InstanceLimits:  domain.MinMax{0, 0, 0},
		ImageID:         "",
		PoolID:          "default",
		DesiredState:    int(SVCStop),
		Launch:          "auto",
		Endpoints:       []ServiceEndpoint{},
		ParentServiceID: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs: []servicedefinition.LogConfig{
			servicedefinition.LogConfig{
				Path: "{{.Description}}",
				Type: "test",
				LogTags: []servicedefinition.LogTag{
					servicedefinition.LogTag{
						Name:  "pepe",
						Value: "{{.Name}}",
					},
				},
			},
		},
		ConfigFiles: map[string]servicedefinition.ConfigFile{
			"Zenosstest.conf": servicedefinition.ConfigFile{
				Filename:    "{{.Name}}test.conf",
				Owner:       "",
				Permissions: "0700",
				Content:     "\n# SAMPLE config file for {{.Name}} {{.InstanceID}}\n\n",
			},
		},
		Snapshot: servicedefinition.SnapshotCommands{
			Pause:  "",
			Resume: "",
		},
		Actions: map[string]string{"debug": "{{.Name}} debug", "stats": "{{.Name}} stats"},
	}, ""},
	{Service{
		ID:              "1",
		Name:            "Collector",
		Context:         map[string]interface{}{"RemoteHost": "a_hostname"},
		Startup:         "",
		Description:     "",
		Instances:       0,
		InstanceLimits:  domain.MinMax{0, 0, 0},
		ImageID:         "",
		PoolID:          "default",
		DesiredState:    int(SVCStop),
		Launch:          "auto",
		Endpoints:       []ServiceEndpoint{},
		ParentServiceID: "0",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs:      []servicedefinition.LogConfig{},
		Snapshot:        servicedefinition.SnapshotCommands{},
		Actions:         map[string]string{},
	}, ""},
	{Service{
		ID:              "2",
		Name:            "pinger",
		Context:         map[string]interface{}{"Count": 32},
		Startup:         "/usr/bin/ping -c {{(context .).Count}} {{(context (parent .)).RemoteHost}}",
		Description:     "Ping a remote host a fixed number of times",
		Instances:       1,
		InstanceLimits:  domain.MinMax{1, 1, 1},
		ImageID:         "test/pinger",
		PoolID:          "default",
		DesiredState:    int(SVCRun),
		Launch:          "auto",
		Endpoints:       []ServiceEndpoint{},
		ParentServiceID: "1",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs:      []servicedefinition.LogConfig{},
		Snapshot:        servicedefinition.SnapshotCommands{},
	}, "/usr/bin/ping -c 32 a_hostname"},
	{Service{
		ID:              "3",
		Name:            "/bin/sh",
		Context:         nil,
		Startup:         "{{.Name}} ls -l .",
		Description:     "Execute ls -l on .",
		Instances:       1,
		InstanceLimits:  domain.MinMax{1, 1, 1},
		ImageID:         "test/bin_sh",
		PoolID:          "default",
		DesiredState:    int(SVCRun),
		Launch:          "auto",
		Endpoints:       []ServiceEndpoint{},
		ParentServiceID: "1",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs:      []servicedefinition.LogConfig{},
		Snapshot:        servicedefinition.SnapshotCommands{},
	}, "/bin/sh ls -l ."},
}

var endpoint_testcases = []struct {
	service  Service
	expected string
}{
	{Service{
		ID:              "100",
		Name:            "Zenoss",
		Context:         map[string]interface{}{"RemoteHost": "hostname"},
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       0,
		InstanceLimits:  domain.MinMax{0, 0, 0},
		ImageID:         "",
		PoolID:          "default",
		DesiredState:    int(SVCStop),
		Launch:          "auto",
		Endpoints:       []ServiceEndpoint{},
		ParentServiceID: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, ""},
	{Service{
		ID:             "101",
		Name:           "Collector",
		Context:        nil,
		Startup:        "",
		Description:    "",
		Instances:      0,
		InstanceLimits: domain.MinMax{0, 0, 0},
		ImageID:        "",
		PoolID:         "default",
		DesiredState:   int(SVCStop),
		Launch:         "auto",
		Endpoints: []ServiceEndpoint{
			BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Purpose:     "something",
					Protocol:    "tcp",
					PortNumber:  1000,
					Application: "{{(context (parent .)).RemoteHost}}_collector",
				}),
		},
		ParentServiceID: "100",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, "hostname_collector"},
}

var context_testcases = []Service{
	{
		ID:      "200",
		Name:    "200",
		PoolID:  "default",
		Launch:  "manual",
		Context: map[string]interface{}{"A": "a_200", "B": "b_200", "g.w": "W"},
	},
	{
		ID:              "201",
		Name:            "201",
		PoolID:          "default",
		Launch:          "manual",
		Context:         map[string]interface{}{"A": "a_201", "C": "c_201"},
		ParentServiceID: "200",
		ConfigFiles: map[string]servicedefinition.ConfigFile{
			"inherited": servicedefinition.ConfigFile{
				// We store the expected value in the Owner field
				// Note that B comes from the parent context
				Owner:   `a_201, b_200, c_201`,
				Content: `{{(context .).A}}, {{(context .).B}}, {{(context .).C}}`,
			},
		},
	},
	{
		ID:              "202",
		Name:            "202",
		PoolID:          "default",
		Launch:          "manual",
		Context:         map[string]interface{}{"g.y": "Y", "g.x": "X", "g.z": "Z", "foo": "bar"},
		ParentServiceID: "200",
		ConfigFiles: map[string]servicedefinition.ConfigFile{
			"range": servicedefinition.ConfigFile{
				// We store the expected value in the Owner field
				// Note the keys are filtered, trimmed and sorted
				Owner:   `w:W, x:X, y:Y, z:Z, `,
				Content: `{{range $key,$val:=contextFilter . "g."}}{{$key}}:{{$val}}, {{end}}`,
			},
		},
	},
	{
		ID:              "203",
		Name:            "203",
		PoolID:          "default",
		Launch:          "manual",
		Context:         map[string]interface{}{"foo.bar-baz": "qux"},
		ParentServiceID: "200",
		ConfigFiles: map[string]servicedefinition.ConfigFile{
			"range": servicedefinition.ConfigFile{
				// We store the expected value in the Owner field
				Owner:   `qux!`,
				Content: `{{(contextFilter . "foo.bar-").baz}}!`,
			},
		},
	},
	{
		ID:              "204",
		Name:            "204",
		PoolID:          "default",
		Launch:          "manual",
		Context:         map[string]interface{}{},
		ParentServiceID: "200",
		ConfigFiles: map[string]servicedefinition.ConfigFile{
			"range": servicedefinition.ConfigFile{
				// We store the expected value in the Owner field
				Owner:   `Woot!`,
				Content: `{{(getContext . "g.w")}}oot!`,
			},
		},
	},
}

var addresses []string

func createSvcs(store Store, ctx datastore.Context) error {
	for _, testcase := range startup_testcases {
		if err := store.Put(ctx, &testcase.service); err != nil {
			return err
		}
	}
	for _, testcase := range endpoint_testcases {
		if err := store.Put(ctx, &testcase.service); err != nil {
			return err
		}
	}
	return nil
}

func createContextSvcs(store Store, ctx datastore.Context) error {
	for _, testcase := range context_testcases {
		if err := store.Put(ctx, &testcase); err != nil {
			return err
		}
	}
	return nil
}

func (s *S) getSVC(svcID string) (Service, error) {
	svc, err := s.store.Get(s.ctx, svcID)
	return *svc, err
}

func (s *S) findChild(svcID, childName string) (Service, error) {
	var svc Service
	svcs, err := s.store.GetChildServices(s.ctx, svcID)
	if err != nil {
		return svc, err
	}
	for _, x := range svcs {
		if x.Name == childName {
			return x, nil
		}
	}
	return svc, nil
}

//TestEvaluateLogConfigTemplate makes sure that the log config templates can be
// parsed and evaluated correctly.
func (s *S) TestEvaluateLogConfigTemplate(t *C) {
	err := createSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)

	testcase := startup_testcases[0]
	testcase.service.EvaluateLogConfigTemplate(s.getSVC, s.findChild, 0)
	// check the tag
	result := testcase.service.LogConfigs[0].LogTags[0].Value
	if result != testcase.service.Name {
		t.Errorf("Was expecting the log tag pepe to be the service name instead it was %s", result)
	}

	// check the path
	result = testcase.service.LogConfigs[0].Path
	if result != testcase.service.Description {
		t.Errorf("Was expecting the log path to be the service description instead it was %s", result)
	}
}

func (s *S) TestEvaluateConfigFilesTemplate(t *C) {
	err := createSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)
	var instanceID = 5

	testcase := startup_testcases[0]
	testcase.service.EvaluateConfigFilesTemplate(s.getSVC, s.findChild, instanceID)

	if len(testcase.service.ConfigFiles) != 1 {
		t.Errorf("Was expecting 1 ConfigFile, found %d", len(testcase.service.ConfigFiles))
	}
	for key, configFile := range testcase.service.ConfigFiles {
		if configFile.Filename != key {
			t.Errorf("Was expecting configFile.Filename to be %s instead it was %s", key, configFile.Filename)
		}
		if !strings.Contains(configFile.Content, fmt.Sprintf("%s %d", testcase.service.Name, instanceID)) {
			t.Errorf("Was expecting configFile.Content to include the service name instead it was %s", configFile.Content)
		}
	}
}

func (s *S) TestEvaluateConfigFilesTemplateContext(t *C) {
	err := createContextSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)
	var instanceID = 5

	for _, testcase := range context_testcases {
		err := testcase.EvaluateConfigFilesTemplate(s.getSVC, s.findChild, instanceID)
		if err != nil {
			t.Errorf("Failed to eval template %s", err)
		}
		for name, cf := range testcase.ConfigFiles {
			// We store the expected value in the Owner field
			expected := cf.Owner
			if cf.Content != expected {
				t.Errorf(`Config file "%s" contents were "%s" expecting "%s"`, name, cf.Content, expected)
			}
		}
	}
}

func (s *S) TestEvaluateStartupTemplate(t *C) {
	err := createSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)

	for _, testcase := range startup_testcases {
		glog.Infof("Service.Startup before: %s", testcase.service.Startup)
		err = testcase.service.EvaluateStartupTemplate(s.getSVC, s.findChild, 0)
		t.Assert(err, IsNil)
		glog.Infof("Service.Startup after: %s, error=%s", testcase.service.Startup, err)
		result := testcase.service.Startup
		if result != testcase.expected {
			t.Errorf("Expecting \"%s\" got \"%s\"\n", testcase.expected, result)
		}
	}
}

// TestEvaluateActionsTemplate makes sure that the Actions templates can be
// parsed and evaluated correctly.
func (s *S) TestEvaluateActionsTemplate(t *C) {
	err := createSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)
	for _, testcase := range startup_testcases {
		glog.Infof("Service.Actions before: %s", testcase.service.Actions)
		err = testcase.service.EvaluateActionsTemplate(s.getSVC, s.findChild, 0)
		glog.Infof("Service.Actions after: %v, error=%v", testcase.service.Actions, err)
		for key, result := range testcase.service.Actions {
			expected := fmt.Sprintf("%s %s", testcase.service.Name, key)
			if result != expected {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", expected, result)
			}
			glog.Infof("Expecting \"%s\" got \"%s\"\n", expected, result)
		}
	}
}

func (s *S) TestEvaluateEndpointTemplate(t *C) {
	err := createSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)

	for _, testcase := range endpoint_testcases {
		if len(testcase.service.Endpoints) > 0 {
			glog.Infof("Service.Endpoint[0].Application: %s", testcase.service.Endpoints[0].Application)
			oldApp := testcase.service.Endpoints[0].Application
			err = testcase.service.EvaluateEndpointTemplates(s.getSVC, s.findChild)
			glog.Infof("Service.Endpoint[0].Application: %s, error=%s", testcase.service.Endpoints[0].Application, err)

			result := testcase.service.Endpoints[0].Application
			if result != testcase.expected {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", testcase.expected, result)
			}
			if testcase.service.Endpoints[0].ApplicationTemplate != oldApp {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", oldApp, testcase.service.Endpoints[0].ApplicationTemplate)
			}

			glog.Infof("Evaluate ServiceEndpoints a second time")
			err = testcase.service.EvaluateEndpointTemplates(s.getSVC, s.findChild)
			result = testcase.service.Endpoints[0].Application
			if result != testcase.expected {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", testcase.expected, result)
			}
			if testcase.service.Endpoints[0].ApplicationTemplate != oldApp {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", oldApp, testcase.service.Endpoints[0].ApplicationTemplate)
			}
		}
	}
}

func (s *S) TestIncompleteStartupInjection(t *C) {
	err := createSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)

	svc := Service{
		ID:              "1000",
		Name:            "pinger",
		Context:         map[string]interface{}{"RemoteHost": "zenoss.com"},
		Startup:         "/usr/bin/ping -c {{(context .).Count}} {{(context .).RemoteHost}}",
		Description:     "Ping a remote host a fixed number of times",
		Instances:       1,
		InstanceLimits:  domain.MinMax{1, 1, 1},
		ImageID:         "test/pinger",
		PoolID:          "default",
		DesiredState:    int(SVCRun),
		Launch:          "auto",
		Endpoints:       []ServiceEndpoint{},
		ParentServiceID: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc.EvaluateStartupTemplate(s.getSVC, s.findChild, 0)
	if svc.Startup == "/usr/bin/ping -c 64 zenoss.com" {
		t.Errorf("Not expecting a match")
	}
}

func (s *S) TestIllegalTemplates(t *C) {
	err := createSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)

	illegal_services := []Service{
		//endpoint
		Service{
			Endpoints: []ServiceEndpoint{
				BuildServiceEndpoint(
					servicedefinition.EndpointDefinition{
						Application: "{{illegal_endpoint}}",
					}),
			},
		},
		//log config - path
		Service{
			LogConfigs: []servicedefinition.LogConfig{
				servicedefinition.LogConfig{
					Path: "{{illegal_logconfig_path}}",
				},
			},
		},
		//log config - type
		Service{
			LogConfigs: []servicedefinition.LogConfig{
				servicedefinition.LogConfig{
					Type: "{{illegal_logconfig_type}}",
				},
			},
		},
		//log config - value
		Service{
			LogConfigs: []servicedefinition.LogConfig{
				servicedefinition.LogConfig{
					LogTags: []servicedefinition.LogTag{
						servicedefinition.LogTag{
							Value: "{{illegal_logconfig_logtag_value}}",
						},
					},
				},
			},
		},
		//config files - filename
		Service{
			ConfigFiles: map[string]servicedefinition.ConfigFile{
				"Zenosstest.conf": servicedefinition.ConfigFile{
					Filename: "{{illegal_configfile_filename}}",
				},
			},
		},
		//config files - content
		Service{
			ConfigFiles: map[string]servicedefinition.ConfigFile{
				"Zenosstest.conf": servicedefinition.ConfigFile{
					Content: "{{illegal_configfile_content}}",
				},
			},
		},
		//startup
		Service{
			Startup: "{{illegal_startup}}",
		},
		//runs
		Service{
			Runs: map[string]string{
				"script": "{{illegal_runs}}",
			},
		},
		//actions
		Service{
			Actions: map[string]string{
				"action": "{{illegal_actions}}",
			},
		},
		//hostname
		Service{
			Hostname: "{{illegal_hostname}}",
		},
		//volumes
		Service{
			Volumes: []servicedefinition.Volume{
				servicedefinition.Volume{
					ResourcePath: "{{illegal_volume_resourcepath}}",
				},
			},
		},
		//prereqs
		Service{
			Prereqs: []domain.Prereq{
				domain.Prereq{
					Script: "{{illegal_prereq_script}}",
				},
			},
		},
		//health check
		Service{
			HealthChecks: map[string]health.HealthCheck{
				"check": health.HealthCheck{
					Script: "{{illegal_healthcheck_script}}",
				},
			},
		},
	}

	for _, svc := range illegal_services {
		err = svc.Evaluate(s.getSVC, s.findChild, 0)
		if err == nil {
			t.Errorf("Expecting error for invalid template: %+v", svc)
		}
	}
}
