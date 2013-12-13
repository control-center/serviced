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
	}, "/usr/bin/ping -c 32 zenoss.com --monitor monitor"},
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
        cp.RemoveService( testcase.service.Id, &unused)
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
		err = testcase.service.EvaluateContext(cp)
		glog.Infof("Service: %+v, %s", testcase.service, err)
	}
}
