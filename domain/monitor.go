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

// Package domain defines the monitoring profiles for control center domain objects
package domain

import "reflect"

//MonitorProfile describes metrics, thresholds and graphs to monitor an object's performance
type MonitorProfile struct {
	MetricConfigs    []MetricConfig    //metrics for domain object
	GraphConfigs     []GraphConfig     //graphs for a domain object
	ThresholdConfigs []ThresholdConfig //thresholds for a domain object
}

//Equals equality test for MonitorProfile
func (profile *MonitorProfile) Equals(that *MonitorProfile) bool {
	return reflect.DeepEqual(profile, that)
}

//ReBuild metrics, graphs and thresholds with the new tags, also set graphs units based on datapoints
func (profile *MonitorProfile) ReBuild(timeSpan string, tags map[string][]string) (*MonitorProfile, error) {
	newProfile := MonitorProfile{
		MetricConfigs:    make([]MetricConfig, len(profile.MetricConfigs)),
		GraphConfigs:     make([]GraphConfig, len(profile.GraphConfigs)),
		ThresholdConfigs: make([]ThresholdConfig, len(profile.ThresholdConfigs)),
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

	//rebuild thresholds
	for i := range profile.ThresholdConfigs {
		newProfile.ThresholdConfigs[i] = profile.ThresholdConfigs[i]
	}

	//build a map of metric -> metricSource -> units
	unitMap := map[string]map[string]string{}
	for i := range newProfile.MetricConfigs {
		metricGroup := &newProfile.MetricConfigs[i]
		for j := range metricGroup.Metrics {
			metric := &metricGroup.Metrics[j]
			if metric.Unit != "" {
				if _, ok := unitMap[metricGroup.ID]; !ok {
					unitMap[metricGroup.ID] = map[string]string{}
				}
				unitMap[metricGroup.ID][metric.ID] = metric.Unit
			}
		}
	}

	//set graph units
	for i := range newProfile.GraphConfigs {
		graph := &newProfile.GraphConfigs[i]
		if graph.Units == "" {
			for j := range graph.DataPoints {
				dataPoint := &graph.DataPoints[j]
				if _, ok := unitMap[dataPoint.MetricSource]; !ok {
					continue
				}
				if _, ok := unitMap[dataPoint.MetricSource][dataPoint.Metric]; !ok {
					continue
				}
				graph.Units = unitMap[dataPoint.MetricSource][dataPoint.Metric]
				break
			}
		}
	}

	return &newProfile, nil
}
