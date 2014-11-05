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

// Package domain contains graph config data objects for all domain objects
package domain

import (
	"encoding/json"
	"reflect"
)

//DataPointRateOptions define the rate options for a data point
type DataPointRateOptions struct {
	Counter        bool  `json:"counter"`
	CounterMax     int64 `json:"counterMax"`
	ResetThreshold int64 `json:"resetThreshold"`
}

type jsonDataPointRateOptions struct {
	Counter        bool   `json:"counter"`
	CounterMax     *int64 `json:"counterMax,omitEmpty"`
	ResetThreshold *int64 `json:"resetThreshold,omitEmpty"`
}

func (d DataPointRateOptions) MarshalJSON() ([]byte, error) {
	var ctrMax *int64
	var rstThr *int64
	if d.CounterMax == 0 {
		ctrMax = nil
	} else {
		ctrMax = &d.CounterMax
	}
	if d.ResetThreshold == 0 {
		rstThr = nil
	} else {
		rstThr = &d.ResetThreshold
	}
	return json.Marshal(jsonDataPointRateOptions{
		Counter:        d.Counter,
		CounterMax:     ctrMax,
		ResetThreshold: rstThr,
	})
}

func (d *DataPointRateOptions) UnmarshalJSON(data []byte) error {
	var temp jsonDataPointRateOptions
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	d.Counter = temp.Counter
	if temp.CounterMax == nil {
		d.CounterMax = 0
	} else {
		d.CounterMax = *temp.CounterMax
	}
	if temp.ResetThreshold == nil {
		d.ResetThreshold = 0
	} else {
		d.ResetThreshold = *temp.ResetThreshold
	}
	return nil
}

// DataPoint defines a datum to be plotted within a graph
type DataPoint struct {
	Aggregator   string                `json:"aggregator"`
	Color        string                `json:"color"`
	Expression   string                `json:"expression"`
	Fill         bool                  `json:"fill"`
	Format       string                `json:"format"`
	Legend       string                `json:"legend"`
	Metric       string                `json:"metric"`       //the metric id inside the metric config (defined by metric source)
	MetricSource string                `json:"metricSource"` //the metric config id in the monitoring profile
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Rate         bool                  `json:"rate"`
	RateOptions  *DataPointRateOptions `json:"rateOptions"`
	Type         string                `json:"type"`
}

// Equals returns if point equals that point
func (point *DataPoint) Equals(that *DataPoint) bool {
	return reflect.DeepEqual(point, that)
}

//GraphConfigRange defines the X-Axis for a graph
type GraphConfigRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// GraphConfig defines a graph for display using central query's
type GraphConfig struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Footer      bool                `json:"footer"`
	Format      string              `json:"format"`
	ReturnSet   string              `json:"returnset"`
	Type        string              `json:"type"`
	Tags        map[string][]string `json:"tags"`
	MinY        *int                `json:"miny"`
	MaxY        *int                `json:"maxy"`
	YAxisLabel  string              `json:"yAxisLabel"`
	TimeZone    string              `json:"timezone,omitempty"`
	DownSample  string              `json:"downsample,omitempty"`
	Description string              `json:"description"`
	Range       *GraphConfigRange   `json:"range"`
	DataPoints  []DataPoint         `json:"datapoints"`
	BuiltIn     bool                `json:"builtin"`
	Units       string              `json:"units"`
	Base        int                 `json:"base"`
}

// Equals returns if graph equals that graph
func (graph *GraphConfig) Equals(that *GraphConfig) bool {
	return reflect.DeepEqual(graph, that)
}
