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

// +build unit

package domain

import (
	"net/http"
	"testing"
)

func TestNewBuilder(t *testing.T) {
	build, err := NewMetricConfigBuilder("/metrics", "POST")
	if err != nil || build == nil {
		t.Fatalf("Failed Creating metric builder: build=%+v, err=%+v", build, err)
	}

	build, err = NewMetricConfigBuilder("/metrics", "?")
	if err == nil || build != nil {
		t.Fatalf("Expected Error Creating metric builder: build=%+v, err=%+v", build, err)
	}
}

func TestBuilder(t *testing.T) {
	build, _ := NewMetricConfigBuilder("metrics/api/performance/query", "POST")
	metric := Metric{ID: "metric_0", Name: "metric_name_0"}
	build.Metric(metric).SetTag("tag", "value-0")
	config, err := build.Config("metric_group", "metric_group_name", "metric_group_description", "1h-ago")
	if err != nil {
		t.Fatalf("Error building config=%+v, err=%+v", config, err)
	}

	headers := make(http.Header)
	headers["Content-Type"] = []string{"application/json"}
	expected := MetricConfig{
		ID:          "metric_group",
		Name:        "metric_group_name",
		Description: "metric_group_description",
		Query: QueryConfig{
			RequestURI: "/metrics/api/performance/query",
			Method:     "POST",
			Headers:    headers,
			Data:       "{\"metrics\":[{\"metric\":\"metric_0\",\"tags\":{\"tag\":[\"value-0\"]}}],\"start\":\"1h-ago\"}",
		},
		Metrics: []Metric{Metric{
			ID:   "metric_0",
			Name: "metric_name_0",
		}}}

	if !expected.Equals(config) {
		t.Logf("Config does not match expected")
		t.Logf("expected=%+v", expected)
		t.Fatalf("acutal=%+v", config)
	}
}
