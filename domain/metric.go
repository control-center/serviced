// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package domain contains metric data objects for all domain objects

package domain

import "encoding/json"
import "github.com/zenoss/glog"
import "github.com/zenoss/serviced/utils"
import "net/url"
import "net/http"
import "strings"
import "errors"

// Metric defines the meta-data for a metric
type Metric struct {
	ID   string //id is a unique idenitifier for the metric
	Name string //name is a canonical name for the metric
}

// MetricRequest defines
type MetricRequest struct {
	Metric
	Tags map[string][]string //tags required in MetricRequest
}

// SetTag puts a tag into the metric request object
func (request *MetricRequest) SetTag(Name string, Values... string) *MetricRequest {
	request.Tags[Name] = Values
	return request
}

// QueryConfig defines the parameters to request a collection of metrics
type QueryConfig struct {
	URL     string      // the http URL to request the metrics
	Method  string      // the http method to retrieve metrics
	Headers http.Header // http headers required to make request
	Data    string      // the http request body to request metrics
}

// Equals compares two QueryConfig objects for equality
func (config *QueryConfig) Equals(that *QueryConfig) bool {
	if config.URL != that.URL {
		return false
	}

	if config.Method != that.Method {
		return false
	}

	if config.Data != that.Data {
		return false
	}

	if config.Headers == nil && that.Headers == nil {
		return true
	}

	if config.Headers != nil && that.Headers == nil {
		return false
	}

	if config.Headers == nil && that.Headers != nil {
		return false
	}

	if len(config.Headers) != len(that.Headers) {
		return false
	}

	for k, v := range config.Headers {
		tv, ok := that.Headers[k]
		if !ok {
			return false
		}
		if !utils.StringSliceEquals(v, tv) {
			return false
		}
	}

	return true
}

// MetricConfig defines a collection of metrics and the query to request said metrics
type MetricConfig struct {
	ID          string      // a unique identifier for the metrics
	Name        string      // a canonical name for the metrics
	Description string      // a description of the metrics
	Query       QueryConfig // the http query to request metrics
	Metrics     []Metric    // meta-data describing all metrics
}

// Equals equality test for MetricConfig
func (config *MetricConfig) Equals(that *MetricConfig) bool {
	if config.ID != that.ID {
		return false
	}

	if config.Name != that.Name {
		return false
	}

	if config.Description != that.Description {
		return false
	}

	if !config.Query.Equals(&that.Query) {
		return false
	}

	if config.Metrics != nil && that.Metrics != nil {
		if len(config.Metrics) == len(that.Metrics) {
			for i := range config.Metrics {
				if config.Metrics[i] != that.Metrics[i] {
					return false
				}
			}
			return true
		}
	}

	return false
}

// Builder aggregates a url, method, and metrics for building a MetricConfig
type Builder struct {
	url     *url.URL        //url to request metrics
	method  string          //method to retrieve metrics
	metrics []MetricRequest //metrics available in url
}

// Metric appends a metric configuration to the Builder
func (builder *Builder) Metric(ID string, Name string) *MetricRequest {
	request := MetricRequest{Metric{ID, Name}, make(map[string][]string)}
	builder.metrics = append(builder.metrics, request)
	return &builder.metrics[len(builder.metrics)-1]
}

// Config builds a MetricConfig using all defined MetricRequests and resets the metrics requets
func (builder *Builder) Config(ID, Name, Description, Start string) (*MetricConfig, error) {
	//config object to build
	headers := make(http.Header)
	headers["Content-Type"] = []string{"application/json"}
	config := &MetricConfig{
		ID:          ID,
		Name:        Name,
		Description: Description,
		Query: QueryConfig{
			URL:     builder.url.String(),
			Method:  builder.method,
			Headers: headers,
		},
		Metrics: make([]Metric, len(builder.metrics)),
	}

	//define a metric type to build json
	type metric struct {
		Metric string            `json:"metric"`
		Tags   map[string][]string `json:"tags"`
	}

	//aggregate request object
	type metrics struct {
		Metrics []metric `json:"metrics"`
		Start   string   `json:"start"`
	}

	// build an array of metric requests to central query and setup config metrics
	request := metrics{
		Metrics: make([]metric, len(builder.metrics)),
		Start:   Start,
	}

	for i := range builder.metrics {
		id := &builder.metrics[i].ID
		name := &builder.metrics[i].Name
		tags := &builder.metrics[i].Tags
		request.Metrics[i] = metric{*id, *tags}
		config.Metrics[i] = Metric{
			ID:   *id,
			Name: *name,
		}
	}

	//build the query body object
	bodyBytes, err := json.Marshal(request)
	if err != nil {
		glog.Errorf("Failed to marshal query body: %+v", err)
		return nil, err
	}

	builder.metrics = make([]MetricRequest, 0)
	config.Query.Data = string(bodyBytes)
	return config, nil
}

// NewMetricConfigBuilder creates a factory to create MetricConfig instances.
func NewMetricConfigBuilder(URL, Method string) (*Builder, error) {
	url, err := url.Parse(URL)
	if err != nil {
		glog.Errorf("Invalid url: URL=%s, method=%s, err=%+v", URL, Method, err)
		return nil, err
	}

	method := strings.ToUpper(Method)
	switch method {
	case "GET":
	case "PUT":
	case "POST":
	default:
		glog.Errorf("Invalid method: URL=%s, method=%s", URL, Method)
		return nil, errors.New("invalid method")
	}

	return &Builder{
		url:     url,
		method:  Method,
		metrics: make([]MetricRequest, 0),
	}, nil
}
