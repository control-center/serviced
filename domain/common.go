// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// MinMax describes a min and max value pair
type MinMax struct {
	Min int
	Max int
}

// HostIPAndPort describes a host and IP port pair
type HostIPAndPort struct {
	HostIP   string
	HostPort string
}

//Validate ensure that the values in min max are valid. Max >= Min >=0  returns error otherwise
func (minmax *MinMax) Validate() error {
	// Instances["min"] and Instances["max"] must be positive
	if minmax.Min < 0 || minmax.Max < 0 {
		return fmt.Errorf("instances constraints must be positive: Min=%v; Max=%v", minmax.Min, minmax.Max)
	}

	// If "min" and "max" are both declared Instances["min"] < Instances["max"]
	if minmax.Max != 0 && minmax.Min > minmax.Max {
		return fmt.Errorf("minimum instances larger than maximum instances: Min=%v; Max=%v", minmax.Min, minmax.Max)
	}
	return nil
}

// StatusCheck describes a periodic process that reports the status of a service
type StatusCheck struct {
	Type            string        // health or metric are supported
	Name            string        // name of status check
	Script          string        // A script to execute to verify the health of a service.
	Interval        time.Duration // The interval at which to execute the script, 0 means script does not exit
	FailureSeverity uint8         // 0 Clear, 1 debug, 2 info, 3 warn, 4 error, 5 critical
}

type jsonStatusCheck struct {
	Type            string
	Name            string
	Script          string
	Interval        float64 // the serialzed version will be in seconds
	FailureSeverity uint8   // 0 Clear, 1 debug, 2 info, 3 warn, 4 error, 5 critical
}

// MarshalJSON implements the json marshal interface. Duration is serialized to seconds (float64).
func (hc StatusCheck) MarshalJSON() ([]byte, error) {
	// in json, the interval is represented in seconds
	interval := float64(hc.Interval) / 1000000000.0
	return json.Marshal(jsonStatusCheck{
		Type:            hc.Type,
		Name:            hc.Name,
		Script:          hc.Script,
		Interval:        interval,
		FailureSeverity: hc.FailureSeverity,
	})
}

// UnmarshalJSON implements the json unmarshal interface. Duration is serialized to seconds (float64).
func (hc *StatusCheck) UnmarshalJSON(data []byte) error {
	var tempHc jsonStatusCheck
	if err := json.Unmarshal(data, &tempHc); err != nil {
		return err
	}
	hc.Type = tempHc.Type
	hc.Name = tempHc.Name
	hc.Script = tempHc.Script
	hc.FailureSeverity = tempHc.FailureSeverity
	// interval in js is in seconds, convert to nanoseconds, then duration
	hc.Interval = time.Duration(tempHc.Interval * 1000000000.0)
	return nil
}

// Prereq is a special script that must execute sucessfully before the service command is executed.
type Prereq struct {
	Name   string
	Script string
}
