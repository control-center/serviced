// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package domain

import (
	"fmt"
	"time"
)

type MinMax struct {
	Min int
	Max int
}

type HostIpAndPort struct {
	HostIp   string
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
	return nil
}

// HealthCheck is a health check object
type HealthCheck struct {
	Script   string        // A script to execute to verify the health of a service.
	Interval time.Duration // The interval at which to execute the script.
}
