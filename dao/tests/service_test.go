// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package tests

import (
	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	coordzk "github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/dao/elasticsearch"
	"github.com/zenoss/serviced/isvcs"

	"fmt"
	"testing"
	"time"
)

var startup_testcases = []struct {
	service  dao.Service
	expected string
}{
	{dao.Service{
		Id:              "0",
		Name:            "Zenoss",
		Context:         "",
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       0,
		ImageId:         "",
		PoolId:          "",
		DesiredState:    0,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs: []dao.LogConfig{
			dao.LogConfig{
				Path: "{{.Description}}",
				Type: "test",
				LogTags: []dao.LogTag{
					dao.LogTag{
						Name:  "pepe",
						Value: "{{.Name}}",
					},
				},
			},
		},
		Snapshot: dao.SnapshotCommands{
			Pause:  "",
			Resume: "",
		},
		Actions: map[string]string{"debug": "{{.Name}} debug", "stats": "{{.Name}} stats"},
	}, ""},
	{dao.Service{
		Id:              "1",
		Name:            "Collector",
		Context:         "{\"RemoteHost\":\"a_hostname\"}",
		Startup:         "",
		Description:     "",
		Instances:       0,
		ImageId:         "",
		PoolId:          "",
		DesiredState:    0,
		Launch:          "",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "0",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs:      []dao.LogConfig{},
		Snapshot:        dao.SnapshotCommands{},
		Actions:         map[string]string{},
	}, ""},
	{dao.Service{
		Id:              "2",
		Name:            "pinger",
		Context:         "{\"Count\": 32}",
		Startup:         "/usr/bin/ping -c {{(context .).Count}} {{(context (parent .)).RemoteHost}}",
		Description:     "Ping a remote host a fixed number of times",
		Instances:       1,
		ImageId:         "test/pinger",
		PoolId:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "1",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs:      []dao.LogConfig{},
		Snapshot:        dao.SnapshotCommands{},
	}, "/usr/bin/ping -c 32 a_hostname"},
	{dao.Service{
		Id:              "3",
		Name:            "/bin/sh",
		Context:         "",
		Startup:         "{{.Name}} ls -l .",
		Description:     "Execute ls -l on .",
		Instances:       1,
		ImageId:         "test/bin_sh",
		PoolId:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "1",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs:      []dao.LogConfig{},
		Snapshot:        dao.SnapshotCommands{},
	}, "/bin/sh ls -l ."},
}

var endpoint_testcases = []struct {
	service  dao.Service
	expected string
}{
	{dao.Service{
		Id:              "100",
		Name:            "Zenoss",
		Context:         "{\"RemoteHost\":\"hostname\"}",
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       0,
		ImageId:         "",
		PoolId:          "",
		DesiredState:    0,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, ""},
	{dao.Service{
		Id:           "101",
		Name:         "Collector",
		Context:      "",
		Startup:      "",
		Description:  "",
		Instances:    0,
		ImageId:      "",
		PoolId:       "",
		DesiredState: 0,
		Launch:       "",
		Endpoints: []dao.ServiceEndpoint{
			dao.ServiceEndpoint{
				Purpose:     "something",
				Protocol:    "tcp",
				PortNumber:  1000,
				Application: "{{(context (parent .)).RemoteHost}}_collector",
			},
		},
		ParentServiceId: "100",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, "hostname_collector"},
}

var addresses []string
var cp *elasticsearch.ControlPlaneDao

func init() {
	var unused int
	var err error
	isvcs.Init()
	isvcs.Mgr.SetVolumesDir("/tmp/serviced-test")
	err = isvcs.Mgr.Wipe()
	if err != nil {
		glog.Fatalf("could not wipe isvcs(): %s", err)
	}
	if err := isvcs.Mgr.Start(); err != nil {
		glog.Fatalf("Could not start es container: %s", err)
	}

	dsn := coordzk.NewDSN([]string{"127.0.0.1:2181"}, time.Second*15).String()
	glog.Infof("zookeeper dsn: %s", dsn)
	zclient, err := coordclient.New("zookeeper", dsn, "", nil)
	if err != nil {
		glog.Fatalf("Could not start es container: %s", err)
	}

	time.Sleep(time.Second * 5)
	if cp, err = elasticsearch.NewControlSvc("localhost", 9200, nil, zclient, "/tmp", "rsync"); err != nil {
		glog.Fatalf("could not start NewControlSvc(): %s", err)
	}

	if err == nil {
		for _, testcase := range startup_testcases {
			var id string
			cp.RemoveService(testcase.service.Id, &unused)
			if err = cp.AddService(testcase.service, &id); err != nil {
				glog.Fatalf("Failed Loading Service: %+v, %s", testcase.service, err)
			}
		}
		for _, testcase := range endpoint_testcases {
			var id string
			cp.RemoveService(testcase.service.Id, &unused)
			if err = cp.AddService(testcase.service, &id); err != nil {
				glog.Fatalf("Failed Loading Service: %+v, %s", testcase.service, err)
			}
		}
	}
}

//TestEvaluateLogConfigTemplate makes sure that the log config templates can be
// parsed and evaluated correctly.
func TestEvaluateLogConfigTemplate(t *testing.T) {
	testcase := startup_testcases[0]
	testcase.service.EvaluateLogConfigTemplate(cp)
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

func TestEvaluateStartupTemplate(t *testing.T) {
	var err error
	for _, testcase := range startup_testcases {
		glog.Infof("Service.Startup before: %s", testcase.service.Startup)
		err = testcase.service.EvaluateStartupTemplate(cp)
		glog.Infof("Service.Startup after: %s, error=%s", testcase.service.Startup, err)
		result := testcase.service.Startup
		if result != testcase.expected {
			t.Errorf("Expecting \"%s\" got \"%s\"\n", testcase.expected, result)
		}
	}
}

// TestEvaluateActionsTemplate makes sure that the Actions templates can be
// parsed and evaluated correctly.
func TestEvaluateActionsTemplate(t *testing.T) {
	var err error
	for _, testcase := range startup_testcases {
		glog.Infof("Service.Actions before: %s", testcase.service.Actions)
		err = testcase.service.EvaluateActionsTemplate(cp)
		glog.Infof("Service.Actions after: %s, error=%s", testcase.service.Actions, err)
		for key, result := range testcase.service.Actions {
			expected := fmt.Sprintf("%s %s", testcase.service.Name, key)
			if result != expected {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", expected, result)
			}
			glog.Infof("Expecting \"%s\" got \"%s\"\n", expected, result)
		}
	}
}

func TestEvaluateEndpointTemplate(t *testing.T) {
	var err error
	for _, testcase := range endpoint_testcases {
		if len(testcase.service.Endpoints) > 0 {
			glog.Infof("Service.Endpoint[0].Application: %s", testcase.service.Endpoints[0].Application)
			oldApp := testcase.service.Endpoints[0].Application
			err = testcase.service.EvaluateEndpointTemplates(cp)
			glog.Infof("Service.Endpoint[0].Application: %s, error=%s", testcase.service.Endpoints[0].Application, err)

			result := testcase.service.Endpoints[0].Application
			if result != testcase.expected {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", testcase.expected, result)
			}
			if testcase.service.Endpoints[0].ApplicationTemplate != oldApp {
				t.Errorf("Expecting \"%s\" got \"%s\"\n", oldApp, testcase.service.Endpoints[0].ApplicationTemplate)
			}

			glog.Infof("Evaluate ServiceEndpoints a second time")
			err = testcase.service.EvaluateEndpointTemplates(cp)
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

func TestIncompleteStartupInjection(t *testing.T) {
	service := dao.Service{
		Id:              "1000",
		Name:            "pinger",
		Context:         "{\"RemoteHost\": \"zenoss.com\"}",
		Startup:         "/usr/bin/ping -c {{(context .).Count}} {{(context .).RemoteHost}}",
		Description:     "Ping a remote host a fixed number of times",
		Instances:       1,
		ImageId:         "test/pinger",
		PoolId:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "0987654321",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	service.EvaluateStartupTemplate(cp)
	if service.Startup == "/usr/bin/ping -c 64 zenoss.com" {
		t.Errorf("Not expecting a match")
	}
}

func TestStoppingParentStopsChildren(t *testing.T) {
	service := dao.Service{
		Id:           "ParentServiceId",
		Name:         "ParentService",
		Startup:      "/usr/bin/ping -c localhost",
		Description:  "Ping a remote host a fixed number of times",
		Instances:    1,
		ImageId:      "test/pinger",
		PoolId:       "default",
		DesiredState: 1,
		Launch:       "auto",
		Endpoints:    []dao.ServiceEndpoint{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	childService1 := dao.Service{
		Id:              "childService1",
		Name:            "childservice1",
		Launch:          "auto",
		Startup:         "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceId: "ParentServiceId",
	}
	childService2 := dao.Service{
		Id:              "childService2",
		Name:            "childservice2",
		Launch:          "auto",
		Startup:         "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceId: "ParentServiceId",
	}
	// add a service with a subservice
	id := "ParentServiceId"
	var err error
	if err = cp.AddService(service, &id); err != nil {
		glog.Fatalf("Failed Loading Parent Service Service: %+v, %s", service, err)
	}

	childService1Id := "childService1"
	childService2Id := "childService2"
	if err = cp.AddService(childService1, &childService1Id); err != nil {
		glog.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}
	if err = cp.AddService(childService2, &childService2Id); err != nil {
		glog.Fatalf("Failed Loading Child Service 2: %+v, %s", childService2, err)
	}
	var unused int
	var stringUnused string
	// start the service
	if err = cp.StartService(id, &stringUnused); err != nil {
		glog.Fatalf("Unable to stop parent service: %+v, %s", service, err)
	}
	// stop the parent
	if err = cp.StopService(id, &unused); err != nil {
		glog.Fatalf("Unable to stop parent service: %+v, %s", service, err)
	}
	// verify the children have all stopped
	query := fmt.Sprintf("ParentServiceId:%s AND NOT Launch:manual", id)
	var services []*dao.Service
	err = cp.GetServices(query, &services)
	for _, subService := range services {
		if subService.DesiredState == 1 && subService.ParentServiceId == id {
			t.Errorf("Was expecting child services to be stopped %v", subService)
		}
	}

	defer cp.RemoveService(childService2Id, &unused)
	defer cp.RemoveService(childService1Id, &unused)
	defer cp.RemoveService(id, &unused)
}

func TestAddVirtualHost(t *testing.T) {
	service := dao.Service{
		Endpoints: []dao.ServiceEndpoint{
			dao.ServiceEndpoint{
				Purpose:     "export",
				Application: "server",
				VHosts:      nil,
			},
		},
	}

	var err error
	if err = service.AddVirtualHost("empty_server", "name"); err == nil {
		t.Errorf("Expected error adding vhost")
	}

	if err = service.AddVirtualHost("server", "name"); err != nil {
		t.Errorf("Unexpected error adding vhost: %v", err)
	}

	//no duplicate hosts can be added... hostnames are case-insensitive
	if err = service.AddVirtualHost("server", "NAME"); err != nil {
		t.Errorf("Unexpected error adding vhost: %v", err)
	}

	if len(service.Endpoints[0].VHosts) != 1 && (service.Endpoints[0].VHosts)[0] != "name" {
		t.Errorf("Virtualhost incorrect, %+v should contain name", service.Endpoints[0].VHosts)
	}
}

func TestRemoveVirtualHost(t *testing.T) {
	service := dao.Service{
		Endpoints: []dao.ServiceEndpoint{
			dao.ServiceEndpoint{
				Purpose:     "export",
				Application: "server",
				VHosts:      []string{"name0", "name1"},
			},
		},
	}

	var err error
	if err = service.RemoveVirtualHost("server", "name0"); err != nil {
		t.Errorf("Unexpected error adding vhost: %v", err)
	}

	if len(service.Endpoints[0].VHosts) != 1 && service.Endpoints[0].VHosts[0] != "name1" {
		t.Errorf("Virtualhost incorrect, %+v should contain one host", service.Endpoints[0].VHosts)
	}

	if err = service.RemoveVirtualHost("server", "name0"); err == nil {
		t.Errorf("Expected error removing vhost")
	}
}
