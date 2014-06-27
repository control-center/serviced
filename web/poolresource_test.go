package web

import (
	"github.com/zenoss/serviced/domain/pool"

	"testing"
)

func TestBuildPoolMonitoringProfile(t *testing.T) {
	pool := pool.ResourcePool{}
	err := buildPoolMonitoringProfile(&pool, []string{}, nil)

	if err != nil {
		t.Fatalf("Failed to build pool monitoring profile: err=%s", err)
	}

	if len(pool.MonitoringProfile.MetricConfigs) <= 0 {
		t.Fatalf("Failed to build pool monitoring profile (missing metric configs): pool=%+v", pool)
	}

	if len(pool.MonitoringProfile.GraphConfigs) <= 0 {
		t.Fatalf("Failed to build pool monitoring profile (missing graphs): pool=%+v", pool)
	}
}
