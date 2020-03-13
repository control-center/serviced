// Copyright 2015 The Serviced Authors.
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
	"fmt"
	"time"
)

const (
	PoolDataAvailableName     = "thinpool-data"
	PoolMetadataAvailableName = "thinpool-metadata"
)

type MetricSeries struct {
	xcoords, ycoords []float64
}

func (s *MetricSeries) X() []float64 {
	return s.xcoords
}

func (s *MetricSeries) Y() []float64 {
	return s.ycoords
}

func DatapointsToSeries(dp []Datapoint) MetricSeries {
	n := len(dp)
	m := MetricSeries{
		xcoords: make([]float64, n),
		ycoords: make([]float64, n),
	}
	for i, d := range dp {
		m.xcoords[i] = float64(d.Timestamp)
		m.ycoords[i] = d.Value.Value
	}
	return m
}

type StorageMetrics struct {
	PoolDataAvailable     MetricSeries
	PoolMetadataAvailable MetricSeries
	Tenants               map[string]MetricSeries
}

func (c *clientImpl) GetAvailableStorage(window time.Duration, aggregator string, tenants ...string) (*StorageMetrics, error) {
	log.WithField("tenants", tenants).Debug("Requesting storage availability metrics")

	options := PerformanceOptions{
		Start:     time.Now().UTC().Add(-window).Format(timeFormat),
		End:       "now",
		Returnset: "exact",
	}
	metrics := []MetricOptions{
		{
			Metric:     "storage.pool.data.available",
			Name:       PoolDataAvailableName,
			Aggregator: aggregator,
		},
		{
			Metric:     "storage.pool.metadata.available",
			Name:       PoolMetadataAvailableName,
			Aggregator: aggregator,
		},
	}
	for _, tenantID := range tenants {
		metrics = append(metrics, MetricOptions{
			Metric:     fmt.Sprintf("storage.filesystem.available.%s", tenantID),
			Name:       tenantID,
			Aggregator: aggregator,
		})
	}
	options.Metrics = metrics
	data, err := c.performanceQuery(options)
	if err != nil {
		log.WithError(err).WithField("options", options).Debug("Storage availability metric query failed")
		return nil, err
	}
	storagemetrics := &StorageMetrics{
		Tenants: make(map[string]MetricSeries),
	}
	for _, result := range data.Results {
		switch result.Metric {
		case PoolDataAvailableName:
			storagemetrics.PoolDataAvailable = DatapointsToSeries(result.Datapoints)
		case PoolMetadataAvailableName:
			storagemetrics.PoolMetadataAvailable = DatapointsToSeries(result.Datapoints)
		default:
			storagemetrics.Tenants[result.Metric] = DatapointsToSeries(result.Datapoints)
		}
	}
	return storagemetrics, nil
}
