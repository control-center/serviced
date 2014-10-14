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
	"fmt"
	"time"
)

type MinMax struct {
	Min     int
	Max     int
	Default int
}

type HostIPAndPort struct {
	HostIP   string
	HostPort string
}

//Validate ensure that the values in min max are valid. Max >= Min >=0  returns error otherwise
func (minmax *MinMax) Validate() error {
	// Instances["min"] and Instances["max"] must be positive
	if minmax.Min < 0 || minmax.Max < 0 {
		return fmt.Errorf("Instances constraints must be positive: Min=%v; Max=%v", minmax.Min, minmax.Max)
	}

	// If "min" and "max" are both declared Instances["min"] < Instances["max"]
	if minmax.Max != 0 && minmax.Min > minmax.Max {
		return fmt.Errorf("Minimum instances larger than maximum instances: Min=%v; Max=%v", minmax.Min, minmax.Max)
	}

	// "Default" should be between min + max, inclusive if max is nonzero and default is set
	if minmax.Default != 0 {
		if minmax.Max != 0 {
			if minmax.Default < minmax.Min || minmax.Default > minmax.Max {
				return fmt.Errorf("Default instance spec must be between min and max, inclusive: Min=%v; Max=%v; Default=%v", minmax.Min, minmax.Max, minmax.Default)
			}
		} else {
			if minmax.Default < minmax.Min {
				return fmt.Errorf("Default instance spec cannot be less than the minimum: Min=%v; Max=%v; Default=%v", minmax.Min, minmax.Max, minmax.Default)
			}
		}
	}
	return nil
}

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

type HealthCheckResult struct {
	ServiceID  string
	InstanceID string
	Name       string
	Timestamp  string
	Passed     string
}

type Prereq struct {
	Name   string
	Script string
}

func (h *HealthCheckResult) ValidEntity() error {
	return nil
}
