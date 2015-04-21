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

package domain

import (
	"encoding/json"
	"time"
)

// HealthCheck is a health check object
type HealthCheck struct {
	Script   string        // A script to execute to verify the health of a service.
	Interval time.Duration // The interval at which to execute the script.
	Timeout  time.Duration // A timeout in which to complete the health check.
}

type jsonHealthCheck struct {
	Script   string
	Interval float64 // the serialzed version will be in seconds
	Timeout  float64
}

func (hc HealthCheck) MarshalJSON() ([]byte, error) {
	// in json, the interval is represented in seconds
	interval := float64(hc.Interval) / 1000000000.0
	timeout := float64(hc.Timeout) / 1000000000.0
	return json.Marshal(jsonHealthCheck{
		Script:   hc.Script,
		Interval: interval,
		Timeout:  timeout,
	})
}

func (hc *HealthCheck) UnmarshalJSON(data []byte) error {
	var tempHc jsonHealthCheck
	if err := json.Unmarshal(data, &tempHc); err != nil {
		return err
	}
	hc.Script = tempHc.Script
	// interval in js is in seconds, convert to nanoseconds, then duration
	hc.Interval = time.Duration(tempHc.Interval * 1000000000.0)
	hc.Timeout = time.Duration(tempHc.Timeout * 1000000000.0)
	return nil
}

// HealthCheckResult is used internally to record the results for checking the
// the health of a single, regular service.
type HealthCheckResult struct {
	ServiceID  string
	InstanceID string
	Name       string
	Timestamp  string
	Passed     string
}

// HealthCheckStatus is the external facing information about the results of
// running a health check. This type is shared by both internal servcies and
// regular services.
type HealthCheckStatus struct {
	Name      string  // the name of the healthcheck
	Status    string  // "passed", "failed",  "stopped", etc
	Timestamp int64   // The last time the healthcheck was performed.
	Interval  float64 // The interval at which the healthcheck was run
	StartedAt int64   // The time when the service was started.
	Failure   string  // Contains details of the failure in cases of Status="failed"
}

func (h *HealthCheckResult) ValidEntity() error {
	return nil
}
