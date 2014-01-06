package tests

import "time"
import "testing"
import "github.com/zenoss/glog"
import "github.com/zenoss/serviced/dao"
import "github.com/zenoss/serviced/isvcs"
import "github.com/zenoss/serviced/dao/elasticsearch"

var testcases = []struct {
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

var addresses []string
var cp, err = elasticsearch.NewControlSvc("localhost", 9200, addresses)

func init() {
	var unused int
	if err == nil {
		err = isvcs.ElasticSearchContainer.Run()
		if err == nil {
			for _, testcase := range testcases {
				var id string
				cp.RemoveService(testcase.service.Id, &unused)
				if err = cp.AddService(testcase.service, &id); err != nil {
					glog.Fatalf("Failed Loading Service: %+v, %s", testcase.service, err)
				}
			}
		} else {
			glog.Fatalf("Could not start es container: %s", err)
		}
	}
}

func TestEvaluateContext(t *testing.T) {
	for _, testcase := range testcases {
		glog.Infof("Service.Startup before: %s, %s", testcase.service.Startup)
		err = testcase.service.EvaluateContext(cp)
		glog.Infof("Service.Startup after: %s, %s", testcase.service.Startup, err)

		result := testcase.service.Startup
		if result != testcase.expected {
			t.Errorf("Expecting \"%s\" got \"%s\"\n", testcase.expected, result)
		}
	}
}

func TestIncompleteInjection(t *testing.T) {
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

	service.EvaluateContext(cp)
	if service.Startup == "/usr/bin/ping -c 64 zenoss.com" {
		t.Errorf("Not expecting a match")
	}
}
