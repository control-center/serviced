package web

import (
	"github.com/zenoss/serviced/domain/pool"

	"testing"
)

func TestBuildPoolMonitoringProfile(t *testing.T) {
	pool := pool.ResourcePool{}
	err := buildPoolMonitoringProfile(&pool, []string{ "1", "2", "3"})

	if err != nil {
		t.Fatalf("Failed to build pool monitoring profile: err=%s", err)
	}

	if len(pool.MonitoringProfile.Metrics) <= 0 {
		t.Fatalf("Failed to build pool monitoring profile: pool=%+v", pool)
	}
}
