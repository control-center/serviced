// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
