// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package domain defines the monitoring profiles for control center domain objects
package domain

import "reflect"

//MonitorProfile describes metrics, thresholds and graphs to monitor an object's performance
type MonitorProfile struct {
	MetricConfigs []MetricConfig //metrics for domain object
	GraphConfigs  []GraphConfig  //graphs for a domain object
	//TODO Thresholds
}

//Equals equality test for MonitorProfile
func (profile *MonitorProfile) Equals(that *MonitorProfile) bool {
	return reflect.DeepEqual(profile, that)
}

//ReBuild metrics, graphs and thresholds with the new parameters
func (profile *MonitorProfile) ReBuild(timeSpan string, tags map[string][]string) (*MonitorProfile, error) {
	newProfile := MonitorProfile{
		MetricConfigs: make([]MetricConfig, len(profile.MetricConfigs)),
		GraphConfigs:  make([]GraphConfig, len(profile.GraphConfigs)),
	}

	build, err := NewMetricConfigBuilder("/metrics/api/performance/query", "POST")
	if err != nil {
		return nil, err
	}

	//rebuild metrics
	for i := range profile.MetricConfigs {
		metricGroup := &profile.MetricConfigs[i]
		for j := range metricGroup.Metrics {
			metric := metricGroup.Metrics[j]
			build.Metric(metric).SetTags(tags)
		}

		config, err := build.Config(metricGroup.ID, metricGroup.Name, metricGroup.Description, timeSpan)
		if err != nil {
			return nil, err
		}

		newProfile.MetricConfigs[i] = *config
	}

	//rebuild graphs
	for i := range profile.GraphConfigs {
		newProfile.GraphConfigs[i] = profile.GraphConfigs[i]
		newProfile.GraphConfigs[i].Tags = tags
	}

	return &newProfile, nil
}
