package web

import (
	"github.com/zenoss/serviced/domain/host"

	"testing"
)

func TestBuildHostMonitoringProfile(t *testing.T) {
	host := host.Host{}
	err := buildHostMonitoringProfile(&host)

	if err != nil {
		t.Fatalf("Failed to build host monitoring profile: err=%s", err)
	}

	if len(host.MonitoringProfile.MetricConfigs) <= 0 {
		t.Fatalf("Failed to build host monitoring profile (missing metrics): host=%+v", host)
	}

	if len(host.MonitoringProfile.GraphConfigs) <= 0 {
		t.Fatalf("Failed to build host monitoring profile (missing graphs): host=%+v", host)
	}
}
