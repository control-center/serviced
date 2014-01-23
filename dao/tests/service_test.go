package tests

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/dao/elasticsearch"
	"github.com/zenoss/serviced/isvcs"

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
		LogConfigs:      []dao.LogConfig{},
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
	isvcs.Mgr.Wipe()
	if cp, err = elasticsearch.NewControlSvc("localhost", 9200, addresses); err != nil {
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
		Id:              "0",
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
