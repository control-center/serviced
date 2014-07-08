// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package service

import (
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"

	"fmt"
	"testing"
)

func (s *S) TestAddVirtualHost(t *C) {
	svc := Service{
		Endpoints: []ServiceEndpoint{
			ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Purpose:     "export",
					Application: "server",
					VHosts:      nil,
				},
			},
		},
	}

	var err error
	if err = svc.AddVirtualHost("empty_server", "name"); err == nil {
		t.Errorf("Expected error adding vhost")
	}

	if err = svc.AddVirtualHost("server", "name"); err != nil {
		t.Errorf("Unexpected error adding vhost: %v", err)
	}

	//no duplicate hosts can be added... hostnames are case-insensitive
	if err = svc.AddVirtualHost("server", "NAME"); err != nil {
		t.Errorf("Unexpected error adding vhost: %v", err)
	}

	if len(svc.Endpoints[0].VHosts) != 1 && (svc.Endpoints[0].VHosts)[0] != "name" {
		t.Errorf("Virtualhost incorrect, %+v should contain name", svc.Endpoints[0].VHosts)
	}
}

func (s *S) TestRemoveVirtualHost(t *C) {
	svc := Service{
		Endpoints: []ServiceEndpoint{
			ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Purpose:     "export",
					Application: "server",
					VHosts:      []string{"name0", "name1"},
				},
			},
		},
	}

	var err error
	if err = svc.RemoveVirtualHost("server", "name0"); err != nil {
		t.Errorf("Unexpected error adding vhost: %v", err)
	}

	if len(svc.Endpoints[0].VHosts) != 1 && svc.Endpoints[0].VHosts[0] != "name1" {
		t.Errorf("Virtualhost incorrect, %+v should contain one host", svc.Endpoints[0].VHosts)
	}

	if err = svc.RemoveVirtualHost("server", "name0"); err == nil {
		t.Errorf("Expected error removing vhost")
	}
}

func TestBuildServiceBuildsMetricConfigs(t *testing.T) {

	sd := servicedefinition.ServiceDefinition{
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "jvm.memory",
					Name:        "JVM Memory",
					Description: "JVM heap vs. non-heap memory usage",
					Metrics: []domain.Metric{
						domain.Metric{ID: "jvm.memory.heap", Name: "JVM Heap Usage"},
						domain.Metric{ID: "jvm.memory.non_heap", Name: "JVM Non-Heap Usage"},
					},
				},
			},
		},
	}

	actual, err := BuildService(sd, "", "", 0, "")
	if err != nil {
		t.Errorf("BuildService Failed w/err=%s", err)
	}

	data_heap_request := fmt.Sprintf("{\"metric\":\"jvm.memory.heap\",\"tags\":{\"controlplane_service_id\":[\"%s\"]}}", actual.ID)
	data_non_heap_request := fmt.Sprintf("{\"metric\":\"jvm.memory.non_heap\",\"tags\":{\"controlplane_service_id\":[\"%s\"]}}", actual.ID)
	data := fmt.Sprintf("{\"metrics\":[%s,%s],\"start\":\"1h-ago\"}", data_heap_request, data_non_heap_request)
	expected := Service{
		ID:        actual.ID,
		CreatedAt: actual.CreatedAt,
		UpdatedAt: actual.UpdatedAt,
		Context:   actual.Context,
		MonitoringProfile: domain.MonitorProfile{
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
						domain.Metric{ID: "jvm.memory.non_heap", Name: "JVM Non-Heap Usage"},
					},
				},
			},
			GraphConfigs:     []domain.GraphConfig{},
			ThresholdConfigs: []domain.ThresholdConfig{},
		},
	}

	if !expected.Equals(actual) {
		t.Logf("expected: %+v", expected.MonitoringProfile)
		t.Logf("actual: %+v", actual.MonitoringProfile)
		t.Error("expected != actual")
	}
}
