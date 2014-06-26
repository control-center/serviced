// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package domain contains graph config data objects for all domain objects
package domain

import "reflect"

//DataPointRateOptions define the rate options for a data point
type DataPointRateOptions struct {
	Counter        bool  `json:"counter"`
	CounterMax     int64 `json:"countermax"`
	ResetThreshold int64 `json:"resetthreshold"`
}

// DataPoint defines a datum to be plotted within a graph
type DataPoint struct {
	Aggregator  string                `json:"aggregator"`
	Color       string                `json:"color"`
	Expression  string                `json:"expression"`
	Fill        bool                  `json:"fill"`
	Format      string                `json:"format"`
	Legend      string                `json:"legend"`
	Metric      string                `json:"metric"`
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Rate        bool                  `json:"rate"`
	RateOptions *DataPointRateOptions `json:"rateOptions"`
	Type        string                `json:"type"`
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
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Footer     bool                `json:"footer"`
	Format     string              `json:"format"`
	ReturnSet  string              `json:"returnset"`
	Type       string              `json:"type"`
	Tags       map[string][]string `json:"tags"`
	MinY       *int                `json:"miny"`
	MaxY       *int                `json:"maxy"`
	YAxisLabel string              `json:"yAxisLabel"`
	TimeZone   string              `json:"timezone"`
	DownSample string              `json:"downsample"`
	Range      *GraphConfigRange   `json:"range"`
	DataPoints []DataPoint         `json:"datapoints"`
}

// Equals returns if graph equals that graph
func (graph *GraphConfig) Equals(that *GraphConfig) bool {
	return reflect.DeepEqual(graph, that)
}
