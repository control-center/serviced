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

package web

import (
	"github.com/control-center/serviced/domain/pool"

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
