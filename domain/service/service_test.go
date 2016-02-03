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
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"

	"fmt"
	"testing"
)

func (s *S) TestAddVirtualHost(t *C) {
	svc := Service{
		Endpoints: []ServiceEndpoint{
			BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Purpose:     "export",
					Application: "server",
					VHostList:   nil,
				}),
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

	if len(svc.Endpoints[0].VHostList) != 1 && (svc.Endpoints[0].VHostList)[0].Name != "name" {
		t.Errorf("Virtualhost incorrect, %+v should contain name", svc.Endpoints[0].VHostList)
	}

	if svc.Endpoints[0].VHostList[0].Enabled != true {
		t.Errorf("Virtualhost %s should be enabled", svc.Endpoints[0].VHostList[0].Name)
	}
}

func (s *S) TestRemoveVirtualHost(t *C) {
	svc := Service{
		Endpoints: []ServiceEndpoint{
			BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Purpose:     "export",
					Application: "server",
					VHostList:   []servicedefinition.VHost{servicedefinition.VHost{Name: "name0"}, servicedefinition.VHost{Name: "name1"}},
				}),
		},
	}

	var err error
	if err = svc.RemoveVirtualHost("server", "name0"); err != nil {
		t.Errorf("Unexpected error removing vhost: %v", err)
	}

	if len(svc.Endpoints[0].VHostList) != 1 && svc.Endpoints[0].VHostList[0].Name != "name1" {
		t.Errorf("Virtualhost incorrect, %+v should contain one host", svc.Endpoints[0].VHostList)
	}

	if err = svc.RemoveVirtualHost("server", "name0"); err == nil {
		t.Errorf("Expected error removing vhost")
	}
}

func (s *S) TestAddPort(t *C) {
	svc := Service{
		Endpoints: []ServiceEndpoint{
			BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Purpose:     "export",
					Application: "server",
					PortList:    nil,
				}),
		},
	}

	var err error
	if err = svc.AddPort("empty_server", ":1234"); err == nil {
		t.Errorf("Expected error adding port")
	}

	if err = svc.AddPort("server", ":1234"); err != nil {
		t.Errorf("Unexpected error adding port: %v", err)
	}

	//no duplicate ports can be added
	if err = svc.AddPort("server", "1234"); err != nil {
		t.Errorf("Unexpected error adding port: %v", err)
	}

	if len(svc.Endpoints[0].PortList) != 1 && (svc.Endpoints[0].PortList)[0].PortAddr != ":1234" {
		t.Errorf("Public port incorrect, %+v should contain port address", svc.Endpoints[0].PortList)
	}

	if svc.Endpoints[0].PortList[0].Enabled != true {
		t.Errorf("Port %s should be enabled", svc.Endpoints[0].PortList[0].PortAddr)
	}
}

func (s *S) TestRemovePort(t *C) {
	svc := Service{
		Endpoints: []ServiceEndpoint{
			BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Purpose:     "export",
					Application: "server",
					PortList:    []servicedefinition.Port{servicedefinition.Port{PortAddr: ":1234"}, servicedefinition.Port{PortAddr: "128.0.0.1:1234"}},
				}),
		},
	}

	var err error
	if err = svc.RemovePort("server", ":1234"); err != nil {
		t.Errorf("Unexpected error removing port: %v", err)
	}

	if len(svc.Endpoints[0].PortList) != 1 && svc.Endpoints[0].PortList[0].PortAddr != "128.0.0.1:1234" {
		t.Errorf("PortList incorrect, %+v should contain 128.0.0.1:1234", svc.Endpoints[0].PortList)
	}

	if err = svc.RemoveVirtualHost("server", ":1234"); err == nil {
		t.Errorf("Expected error removing port")
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

func TestScrubPortString(t *testing.T) {
	testStrings := map[string]string{
		"1234":                  ":1234",
		":1234":                 ":1234",
		"128.0.0.1:1234":        "128.0.0.1:1234",
		"http://128.0.0.1:1234": "128.0.0.1:1234",
	}

	for portString, expectedString := range testStrings {
		scrubbedString := ScrubPortString(portString)
		if scrubbedString != expectedString {
			t.Fail()
		}
	}
}
