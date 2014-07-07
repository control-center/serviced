// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package domain defines the threshold configurations in a monitoring profile
package domain

import "reflect"

// ThresholdConfig defines all meta-data for a threshold on a metric with in a monitoring profile
type ThresholdConfig struct {
	ID          string //a unique id for thresholds
	Name        string //canonical name of threshold
	Type        string //type of threshold (MinMax, Duration, ValueChange, or HoltWinters)
	Description string //description of threshold
	AppliedTo   int    //how should this threshold be applied 0=everything, 1=services only, 2=running services only

	MetricSource string                 //id of the MetricConfig this is applied to
	DataPoints   []string               //List of metrics within a MetricConfig this is applied to
	Threshold    interface{}            // threshold data (either MinMaxThreshold, DurationThreshold, HoltWintersThreshold)
	EventTags    map[string]interface{} //all relevant event data
}

// MinMaxThreshold triggers events when a metric breaches either min or max threshold value
type MinMaxThreshold struct {
	Min *int64 //min threshold value
	Max *int64 //max threshold value
}

// DurationThreshold tiggers events when a percentage of min/max thresholds are breached in a given time perion
type DurationThreshold struct {
	Min        *int64 //min threshold value, null for no min
	Max        *int64 //max threshold value, null for no max
	TimePeriod string //provide a time period using time operators like day, hours, minutes, or just the number of seconds. An example period: 4 hours 5 minutes
	Percentage int    //Percentage of violations to trigger an event: a number from 0 (any violation triggers an event) to 100 (all values must violate the threshold)
}

//HoltWintersThreshold adds the ability to fire threshold events when a device exceeds cyclical predicted values
type HoltWintersThreshold struct {
	Alpha  float64 //A number from 0 to 1 that controls how quickly the model adapts to unexpected values
	Beta   float64 //A number from 0 to 1 that controls how quicly the model adapts to changes in unexpected rates changes.
	Rows   int64   //The number of points to use for predictive purposes
	Season int64   //The number of primary data points in a season.  Note that Rows must be at least as large as Season
}

//Equals compares two threshold configs for equality
func (config *ThresholdConfig) Equals(that *ThresholdConfig) bool {
	return reflect.DeepEqual(config, that)
}
