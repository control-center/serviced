// Copyright 2016 The Serviced Authors.
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

// Package stats collects serviced metrics and posts them to the TSDB.
package stats

import (
	"strconv"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/rcrowley/go-metrics"

	"time"
)

// StorageStatsReporter collects and posts storage stats to the TSDB.
type StorageStatsReporter struct {
	statsReporter
	hostID          string
	storageRegistry metrics.Registry
}

// NewStorageStatsReporter creates a new NewStorageStatsReporter and kicks off the reporting goroutine.
func NewStorageStatsReporter(destination string, interval time.Duration) (*StorageStatsReporter, error) {
	hostID, err := utils.HostID()
	if err != nil {
		plog.WithError(err).Errorf("Could not determine host ID.")
		return nil, err
	}

	sr := StorageStatsReporter{
		statsReporter: statsReporter{
			destination:  destination,
			closeChannel: make(chan struct{}),
		},
		hostID: hostID,
	}

	sr.storageRegistry = metrics.NewRegistry()
	sr.statsReporter.updateStatsFunc = sr.updateStats
	sr.statsReporter.gatherStatsFunc = sr.gatherStats
	go sr.report(interval)
	return &sr, nil
}

// Fills out the metric consumer format.
func (sr *StorageStatsReporter) gatherStats(t time.Time) []Sample {
	stats := []Sample{}
	// Handle the storage metrics.
	reg, _ := sr.storageRegistry.(*metrics.StandardRegistry)
	tagmap := map[string]string{"controlplane_host_id": sr.hostID}
	reg.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Gauge:
			stats = append(stats, Sample{name, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
		case metrics.GaugeFloat64:
			stats = append(stats, Sample{name, strconv.FormatFloat(metric.Value(), 'f', -1, 32), t.Unix(), tagmap})
		}
	})
	return stats
}

func (sr StorageStatsReporter) updateStats() {
	volumeStatuses := volume.GetStatus()
	if volumeStatuses == nil || len(volumeStatuses.GetAllStatuses()) == 0 {
		plog.Errorf("Unexpected error getting volume status")
		return
	}
	for _, volumeStatus := range volumeStatuses.GetAllStatuses() {
		for _, volumeUsage := range volumeStatus.GetUsageData() {
			metricName := volumeUsage.GetMetricName()

			valUint64, err := volumeUsage.GetValueUInt64()
			if err == volume.ErrWrongDataType {
				valFloat64, err := volumeUsage.GetValueFloat64()
				if err != nil {
					plog.WithError(err).Errorf("Error parsing volume usage %s", volumeUsage.GetType())
				} else {
					if metricName != "" {
						metrics.GetOrRegisterGaugeFloat64(metricName, sr.storageRegistry).Update(valFloat64)
					}
				}
			} else if err != nil {
				plog.WithError(err).Errorf("Error parsing volume usage %s", volumeUsage.GetType())
			} else {
				if metricName != "" {
					metrics.GetOrRegisterGauge(metricName, sr.storageRegistry).Update(int64(valUint64))
				}
			}
		}
	}
}
