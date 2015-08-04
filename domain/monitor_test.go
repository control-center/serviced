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

func TestReBuild(t *testing.T) {
	profile := MonitorProfile{
		MetricConfigs: []MetricConfig{
			MetricConfig{
				ID:   "memory",
				Name: "Memory Metrics",
				Metrics: []Metric{
					Metric{ID: "free", Name: "Free Memory"},
				},
			},
		},
		GraphConfigs: []GraphConfig{
			GraphConfig{
				ID: "free.memory",
			},
		},
	}

	tags := map[string][]string{
		"controlplane_host_id": []string{"1", "2"},
	}

	newProfile, err := profile.ReBuild("1h-ago", tags)
	if err != nil {
		t.Fatalf("Error re building profile=%v", err)
	}

	headers := make(http.Header)
	headers["Content-Type"] = []string{"application/json"}
	expectedProfile := &MonitorProfile{
		MetricConfigs: []MetricConfig{
			MetricConfig{
				ID:   "memory",
				Name: "Memory Metrics",
				Query: QueryConfig{
					RequestURI: "/metrics/api/performance/query",
					Method:     "POST",
					Headers:    headers,
					Data:       "{\"metrics\":[{\"metric\":\"free\",\"tags\":{\"controlplane_host_id\":[\"1\",\"2\"]}}],\"start\":\"1h-ago\"}",
				},
				Metrics: []Metric{
					Metric{ID: "free", Name: "Free Memory"},
				},
			},
		},
		GraphConfigs: []GraphConfig{
			GraphConfig{
				ID:   "free.memory",
				Tags: tags,
			},
		},
		ThresholdConfigs: []ThresholdConfig{},
	}

	if !newProfile.Equals(expectedProfile) {
		t.Logf("rebuilt profile != expected")
		t.Logf("newProfile: %+v", newProfile)
		t.Fatalf("expectedProfile: %+v", expectedProfile)
	}
}

func TestReBuildAssignsUnitsToGraphConfigOneDataPoint(t *testing.T) {
	profile := MonitorProfile{
		MetricConfigs: []MetricConfig{
			MetricConfig{
				ID:   "memory",
				Name: "Memory Metrics",
				Metrics: []Metric{
					Metric{ID: "free", Name: "Free Memory", Unit: "mb"},
				},
			},
		},
		GraphConfigs: []GraphConfig{
			GraphConfig{
				ID: "free.memory",
				DataPoints: []DataPoint{
					DataPoint{
						Metric:       "free",
						MetricSource: "memory",
					},
				},
			},
		},
	}

	tags := map[string][]string{
		"controlplane_host_id": []string{"1", "2"},
	}

	newProfile, err := profile.ReBuild("1h-ago", tags)
	if err != nil {
		t.Fatalf("Error re building profile=%v", err)
	}

	headers := make(http.Header)
	headers["Content-Type"] = []string{"application/json"}
	expectedProfile := &MonitorProfile{
		MetricConfigs: []MetricConfig{
			MetricConfig{
				ID:   "memory",
				Name: "Memory Metrics",
				Query: QueryConfig{
					RequestURI: "/metrics/api/performance/query",
					Method:     "POST",
					Headers:    headers,
					Data:       "{\"metrics\":[{\"metric\":\"free\",\"tags\":{\"controlplane_host_id\":[\"1\",\"2\"]}}],\"start\":\"1h-ago\"}",
				},
				Metrics: []Metric{
					Metric{ID: "free", Name: "Free Memory", Unit: "mb"},
				},
			},
		},
		GraphConfigs: []GraphConfig{
			GraphConfig{
				ID:    "free.memory",
				Tags:  tags,
				Units: "mb",
				DataPoints: []DataPoint{
					DataPoint{
						Metric:       "free",
						MetricSource: "memory",
					},
				},
			},
		},
		ThresholdConfigs: []ThresholdConfig{},
	}

	if !newProfile.Equals(expectedProfile) {
		t.Logf("rebuilt profile != expected")
		t.Logf("newProfile: %+v", newProfile)
		t.Fatalf("expectedProfile: %+v", expectedProfile)
	}
}

func TestReBuildAssignsUnitsToGraphConfig2DataPoints(t *testing.T) {
	profile := MonitorProfile{
		MetricConfigs: []MetricConfig{
			MetricConfig{
				ID:   "memory",
				Name: "Memory Metrics",
				Metrics: []Metric{
					Metric{ID: "free", Name: "Free Memory", Unit: ""},
					Metric{ID: "used", Name: "Used Memory", Unit: "mb"},
				},
			},
		},
		GraphConfigs: []GraphConfig{
			GraphConfig{
				ID: "free.memory",
				DataPoints: []DataPoint{
					DataPoint{
						Metric:       "free",
						MetricSource: "memory",
					},
					DataPoint{
						Metric:       "used",
						MetricSource: "memory",
					},
				},
			},
		},
	}

	tags := map[string][]string{
		"controlplane_host_id": []string{"1", "2"},
	}

	newProfile, err := profile.ReBuild("1h-ago", tags)
	if err != nil {
		t.Fatalf("Error re building profile=%v", err)
	}

	headers := make(http.Header)
	headers["Content-Type"] = []string{"application/json"}
	expectedProfile := &MonitorProfile{
		MetricConfigs: []MetricConfig{
			MetricConfig{
				ID:   "memory",
				Name: "Memory Metrics",
				Query: QueryConfig{
					RequestURI: "/metrics/api/performance/query",
					Method:     "POST",
					Headers:    headers,
					Data:       "{\"metrics\":[{\"metric\":\"free\",\"tags\":{\"controlplane_host_id\":[\"1\",\"2\"]}},{\"metric\":\"used\",\"tags\":{\"controlplane_host_id\":[\"1\",\"2\"]}}],\"start\":\"1h-ago\"}",
				},
				Metrics: []Metric{
					Metric{ID: "free", Name: "Free Memory", Unit: ""},
					Metric{ID: "used", Name: "Used Memory", Unit: "mb"},
				},
			},
		},
		GraphConfigs: []GraphConfig{
			GraphConfig{
				ID:    "free.memory",
				Tags:  tags,
				Units: "mb",
				DataPoints: []DataPoint{
					DataPoint{
						Metric:       "free",
						MetricSource: "memory",
					},
					DataPoint{
						Metric:       "used",
						MetricSource: "memory",
					},
				},
			},
		},
		ThresholdConfigs: []ThresholdConfig{},
	}

	if !newProfile.Equals(expectedProfile) {
		t.Logf("rebuilt profile != expected")
		t.Logf("newProfile: %+v", newProfile)
		t.Fatalf("expectedProfile: %+v", expectedProfile)
	}
}
