package zzk

import (
	"fmt"
	"testing"

	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain/servicestate"
)

func TestSssToRs(t *testing.T) {
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
	svcstate := servicestate.ServiceState{}
	rs, err := sssToRs(svc, &svcstate)
	if err != nil {
		t.Error("%v", err)
	}

	if fmt.Sprintf("%+v", svc.MonitoringProfile.MetricConfigs) != fmt.Sprintf("%+v", rs.MonitoringProfile.MetricConfigs) {
		t.Logf("expected: %+v", svc.MonitoringProfile.MetricConfigs)
		t.Logf("actual: %+v", rs.MonitoringProfile.MetricConfigs)
		t.Error("expected != actual")
	}

}
