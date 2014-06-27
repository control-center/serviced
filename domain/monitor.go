// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package domain defines the monitoring profiles for control center domain objects
package domain

import "reflect"

//MonitorProfile describes metrics, thresholds and graphs to monitor an entity's performance
type MonitorProfile struct {
	MetricConfigs []MetricConfig //metrics for domain object
	GraphConfigs  []GraphConfig  //graphs for a domain object
	//TODO Thresholds
}

//Equals equality test for MonitorProfile
func (profile *MonitorProfile) Equals(that *MonitorProfile) bool {
	return reflect.DeepEqual(profile, that)
}
