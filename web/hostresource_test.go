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
	"github.com/control-center/serviced/domain/host"

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
