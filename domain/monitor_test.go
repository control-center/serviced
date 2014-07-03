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
	}

	if !newProfile.Equals(expectedProfile) {
		t.Logf("rebuilt profile != expected")
		t.Logf("newProfile: %+v", newProfile)
		t.Fatalf("expectedProfile: %+v", expectedProfile)
	}
}
