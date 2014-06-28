package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain/servicestate"
)

func TestNewRunningService(t *testing.T) {
	sd := servicedefinition.ServiceDefinition{
		Metrics: []servicedefinition.MetricGroup{
			servicedefinition.MetricGroup{
				ID:          "jvm.memory",
				Name:        "JVM Memory",
				Description: "JVM heap vs. non-heap memory usage",
				Metrics: []servicedefinition.Metric{
					servicedefinition.Metric{ID: "jvm.memory.heap", Name: "JVM Heap Usage"},
					servicedefinition.Metric{ID: "jvm.memory.non_heap", Name: "JVM Non-Heap Usage"},
				},
			},
		},
	}
	svc, err := service.BuildService(sd, "", "", 0, "")
	if err != nil {
		t.Errorf("BuildService Failed w/err=%s", err)
	}
	data_heap_request := fmt.Sprintf("{\"metric\":\"jvm.memory.heap\",\"tags\":{\"controlplane_service_id\":[\"%s\"]}}", svc.Id)
	data_non_heap_request := fmt.Sprintf("{\"metric\":\"jvm.memory.non_heap\",\"tags\":{\"controlplane_service_id\":[\"%s\"]}}", svc.Id)
	data := fmt.Sprintf("{\"metrics\":[%s,%s],\"start\":\"1h-ago\"}", data_heap_request, data_non_heap_request)
	svc.MonitoringProfile = domain.MonitorProfile{
		MetricConfigs: []domain.MetricConfig{
			domain.MetricConfig{
				ID:          "jvm.memory",
				Name:        "JVM Memory",
				Description: "JVM heap vs. non-heap memory usage",
				Query: domain.QueryConfig{
					RequestURI: "/metrics/api/performance/query",
					Method:     "POST",
					Headers: map[string][]string{
						"Content-Type": []string{"application/json"},
					},
					Data: data,
				},
				Metrics: []domain.Metric{
					domain.Metric{ID: "jvm.memory.heap", Name: "JVM Heap Usage"},
				},
			},
		},
	}

	svcstate, err := servicestate.BuildFromService(svc, "fakehostid")
	if err != nil {
		t.Error("%v", err)
	}

	rs, err := NewRunningService(svc, svcstate)
	if err != nil {
		t.Error("%v", err)
	}

	var query interface{}
	json.Unmarshal([]byte(rs.MonitoringProfile.MetricConfigs[0].Query.Data), &query)

	metrics := query.(map[string]interface{})["metrics"].([]interface{})[0].(map[string]interface{})

	tags := metrics["tags"].(map[string]interface{})

	controlplane_instance_id := tags["controlplane_instance_id"].([]interface{})[0]
	if controlplane_instance_id != "0" {
		t.Errorf("Expected %+v, got %+v", "0", controlplane_instance_id)
	}

	controlplane_service_id := tags["controlplane_service_id"].([]interface{})[0]
	if controlplane_service_id != svc.Id {
		t.Errorf("Expected %+v, got %+v", svc.Id, controlplane_service_id)
	}
}
