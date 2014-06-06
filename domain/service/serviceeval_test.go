// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package service

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"

	"fmt"
	"time"
	"strings"
)

var startup_testcases = []struct {
	service  Service
	expected string
}{
	{Service{
		Id:              "0",
		Name:            "Zenoss",
		Context:         "",
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       0,
		InstanceLimits:  domain.MinMax{0, 0},
		ImageID:         "",
		PoolID:          "default",
		DesiredState:    0,
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
				Filename: "{{.Name}}test.conf",
				Owner: "",
				Permissions: "0700",
				Content: "\n# SAMPLE config file for {{.Name}}\n\n",
			},
		},
		Snapshot: servicedefinition.SnapshotCommands{
			Pause:  "",
			Resume: "",
		},
		Actions: map[string]string{"debug": "{{.Name}} debug", "stats": "{{.Name}} stats"},
	}, ""},
	{Service{
		Id:              "1",
		Name:            "Collector",
		Context:         "{\"RemoteHost\":\"a_hostname\"}",
		Startup:         "",
		Description:     "",
		Instances:       0,
		InstanceLimits:  domain.MinMax{0, 0},
		ImageID:         "",
		PoolID:          "default",
		DesiredState:    0,
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
		Id:              "2",
		Name:            "pinger",
		Context:         "{\"Count\": 32}",
		Startup:         "/usr/bin/ping -c {{(context .).Count}} {{(context (parent .)).RemoteHost}}",
		Description:     "Ping a remote host a fixed number of times",
		Instances:       1,
		InstanceLimits:  domain.MinMax{1, 1},
		ImageID:         "test/pinger",
		PoolID:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []ServiceEndpoint{},
		ParentServiceID: "1",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs:      []servicedefinition.LogConfig{},
		Snapshot:        servicedefinition.SnapshotCommands{},
	}, "/usr/bin/ping -c 32 a_hostname"},
	{Service{
		Id:              "3",
		Name:            "/bin/sh",
		Context:         "",
		Startup:         "{{.Name}} ls -l .",
		Description:     "Execute ls -l on .",
		Instances:       1,
		InstanceLimits:  domain.MinMax{1, 1},
		ImageID:         "test/bin_sh",
		PoolID:          "default",
		DesiredState:    1,
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
		Id:              "100",
		Name:            "Zenoss",
		Context:         "{\"RemoteHost\":\"hostname\"}",
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       0,
		InstanceLimits:  domain.MinMax{0, 0},
		ImageID:         "",
		PoolID:          "default",
		DesiredState:    0,
		Launch:          "auto",
		Endpoints:       []ServiceEndpoint{},
		ParentServiceID: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, ""},
	{Service{
		Id:             "101",
		Name:           "Collector",
		Context:        "",
		Startup:        "",
		Description:    "",
		Instances:      0,
		InstanceLimits: domain.MinMax{0, 0},
		ImageID:        "",
		PoolID:         "default",
		DesiredState:   0,
		Launch:         "auto",
		Endpoints: []ServiceEndpoint{
			ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Purpose:     "something",
					Protocol:    "tcp",
					PortNumber:  1000,
					Application: "{{(context (parent .)).RemoteHost}}_collector",
				},
			},
		},
		ParentServiceID: "100",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, "hostname_collector"},
}

var addresses []string

func createSvcs(store *Store, ctx datastore.Context) error {
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

func (s *S) getSVC(svcID string) (Service, error) {
	svc, err := s.store.Get(s.ctx, svcID)
	return *svc, err
}

//TestEvaluateLogConfigTemplate makes sure that the log config templates can be
// parsed and evaluated correctly.
func (s *S) TestEvaluateLogConfigTemplate(t *C) {
	err := createSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)

	testcase := startup_testcases[0]
	testcase.service.EvaluateLogConfigTemplate(s.getSVC)
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

	testcase := startup_testcases[0]
	testcase.service.EvaluateConfigFilesTemplate(s.getSVC)

	if len(testcase.service.ConfigFiles) != 1 {
		t.Errorf("Was expecting 1 ConfigFile, found %d", len(testcase.service.ConfigFiles))
	}
	for key, configFile := range testcase.service.ConfigFiles {
		if configFile.Filename != key {
			t.Errorf("Was expecting configFile.Filename to be %s instead it was %s", key, configFile.Filename)
		}
		if !strings.Contains(configFile.Content, testcase.service.Name) {
			t.Errorf("Was expecting configFile.Content to include the service name instead it was %s", configFile.Content)
		}
	}
}

func (s *S) TestEvaluateStartupTemplate(t *C) {
	err := createSvcs(s.store, s.ctx)
	t.Assert(err, IsNil)

	for _, testcase := range startup_testcases {
		glog.Infof("Service.Startup before: %s", testcase.service.Startup)
		err = testcase.service.EvaluateStartupTemplate(s.getSVC)
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
		err = testcase.service.EvaluateActionsTemplate(s.getSVC)
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
			err = testcase.service.EvaluateEndpointTemplates(s.getSVC)
			glog.Infof("Service.Endpoint[0].Application: %s, error=%s", testcase.service.Endpoints[0].Application, err)

			result := testcase.service.Endpoints[0].Application
			if result != testcase.expected {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", testcase.expected, result)
			}
			if testcase.service.Endpoints[0].ApplicationTemplate != oldApp {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", oldApp, testcase.service.Endpoints[0].ApplicationTemplate)
			}

			glog.Infof("Evaluate ServiceEndpoints a second time")
			err = testcase.service.EvaluateEndpointTemplates(s.getSVC)
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
		Id:              "1000",
		Name:            "pinger",
		Context:         "{\"RemoteHost\": \"zenoss.com\"}",
		Startup:         "/usr/bin/ping -c {{(context .).Count}} {{(context .).RemoteHost}}",
		Description:     "Ping a remote host a fixed number of times",
		Instances:       1,
		InstanceLimits:  domain.MinMax{1, 1},
		ImageID:         "test/pinger",
		PoolID:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []ServiceEndpoint{},
		ParentServiceID: "0987654321",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc.EvaluateStartupTemplate(s.getSVC)
	if svc.Startup == "/usr/bin/ping -c 64 zenoss.com" {
		t.Errorf("Not expecting a match")
	}
}
