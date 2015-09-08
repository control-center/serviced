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

// Package domain defines the threshold configurations in a monitoring profile
package domain

import (
	"encoding/json"
	"reflect"
	"time"
)

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
	Min string //min threshold value
	Max string //max threshold value
}

// DurationThreshold tiggers events when a percentage of min/max thresholds are breached in a given time perion
type DurationThreshold struct {
	Min        *int64        //min threshold value, null for no min
	Max        *int64        //max threshold value, null for no max
	TimePeriod time.Duration //a timePeriod (window) that triggers the threshold
	Percentage int           //Percentage of violations to trigger an event: a number from 0 (any violation triggers an event) to 100 (all values must violate the threshold)
}

type jsonDurationThreshold struct {
	Min        *int64  //min threshold value, null for no min
	Max        *int64  //max threshold value, null for no max
	TimePeriod float64 //a timePeriod (window) that triggers the threshold
	Percentage int     //Percentage of violations to trigger an event: a number from 0 (any violation triggers an event) to 100 (all values must violate the threshold)
}

func (t DurationThreshold) MarshalJSON() ([]byte, error) {
	// in json, the TimePeriod is represented in seconds
	timePeriod := float64(t.TimePeriod) / 1000000000.0
	return json.Marshal(jsonDurationThreshold{
		Min:        t.Min,
		Max:        t.Max,
		TimePeriod: timePeriod,
		Percentage: t.Percentage,
	})
}

func (t *DurationThreshold) UnmarshalJSON(data []byte) error {
	var temp jsonDurationThreshold
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	// interval in js is in seconds, convert to nanoseconds, then duration
	timePeriod := time.Duration(temp.TimePeriod * 1000000000.0)
	t.Min = temp.Min
	t.Max = temp.Max
	t.TimePeriod = timePeriod
	t.Percentage = temp.Percentage
	return nil
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
