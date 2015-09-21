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

package metrics

import (
	"encoding/json"
	"strconv"
)

const timeFormat = "2006/01/02-15:04:00-MST"

type Float struct {
	Value float64
	IsNaN bool
}

func (f *Float) UnmarshalJSON(b []byte) error {
	var source string
	json.Unmarshal(b, &source)

	switch source {
	case "NaN":
		f.Value = 0
		f.IsNaN = true
	default:
		json.Unmarshal(b, &f.Value)
		f.IsNaN = false
	}

	return nil
}

func (f Float) MarshalJSON() ([]byte, error) {
	if f.IsNaN {
		return json.Marshal("NaN")
	}

	value := strconv.FormatFloat(f.Value, 'E', -1, 64)
	return json.Marshal(value)
}

// PerformanceOptions is the request object for doing a performance query.
type PerformanceOptions struct {
	Start                string              `json:"start,omitempty"`
	End                  string              `json:"end,omitempty"`
	Returnset            string              `json:"returnset,omitempty"`
	Downsample           string              `json:"downsample,omitempty"`
	DownsampleMultiplier string              `json:"downsampleMultiplier,omitempty"`
	Tags                 map[string][]string `json:"tags,omitempty"`
	Metrics              []MetricOptions     `json:"metrics,omitempty"`
}

// MetricOptions are the options for receiving metrics for a set of data.
type MetricOptions struct {
	Metric       string              `json:"metric,omitempty"`
	Name         string              `json:"name,omitempty"`
	ID           string              `json:"id,omitempty"`
	Aggregator   string              `json:"aggregator,omitempty"`
	Interpolator string              `json:"interpolator,omitempty"`
	Rate         bool                `json:"rate,omitempty"`
	RateOptions  RateOptions         `json:"rateOptions,omitempty"`
	Expression   string              `json:"expression,omitempty"`
	Tags         map[string][]string `json:"tags,omitempty"`
}

// RateOptions are the options for collecting performance data.
type RateOptions struct {
	Counter        bool `json:"counter,omitempty"`
	CounterMax     int  `json:"counterMax,omitempty"`
	ResetThreshold int  `json:"resetThreshold,omitempty"`
}

// PerformanceData is the resulting object from a performance query.
type PerformanceData struct {
	ClientID        string       `json:"clientId,omitempty"`
	Source          string       `json:"source,omitempty"`
	StartTime       string       `json:"startTime,omitempty"`
	StartTimeActual int64        `json:"startTimeActual"`
	EndTime         string       `json:"endTime,omitempty"`
	EndTimeActual   int64        `json:"endTimeActual"`
	ExactTimeWindow bool         `json:"exactTimeWindow,omitempty"`
	Results         []ResultData `json:"results,omitempty"`
}

// ResultData is actual resulting data from the query per metric and tag
type ResultData struct {
	Datapoints []Datapoint         `json:"datapoints,omitempty"`
	Metric     string              `json:"metric,omitempty"`
	Tags       map[string][]string `json:"tags,omitempty"`
}

// Datapoint is a single numerical datapoint.
type Datapoint struct {
	Timestamp int64 `json:"timestamp"`
	Value     Float `json:"value,omitempty"`
}

func (c *Client) performanceQuery(opts PerformanceOptions) (*PerformanceData, error) {
	path := "/api/performance/query"
	body, _, err := c.do("POST", path, opts)
	if err != nil {
		return nil, err
	}
	var perfdata PerformanceData
	if err = json.Unmarshal(body, &perfdata); err != nil {
		return nil, err
	}
	return &perfdata, nil
}
