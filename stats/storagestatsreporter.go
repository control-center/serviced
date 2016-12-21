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
	"github.com/zenoss/glog"

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
		glog.Errorf("Could not determine host ID.")
		return nil, err
	}

	ssr := StorageStatsReporter{
		statsReporter: statsReporter{
			destination:  destination,
			closeChannel: make(chan struct{}),
		},
		hostID: hostID,
	}

	ssr.storageRegistry = metrics.NewRegistry()
	ssr.statsReporter.updateStatsFunc = ssr.updateStats
	ssr.statsReporter.gatherStatsFunc = ssr.gatherStats
	go ssr.report(interval)
	return &ssr, nil
}

// Fills out the metric consumer format.
func (sr *StorageStatsReporter) gatherStats(t time.Time) []Sample {
	stats := []Sample{}
	// Handle the host metrics.
	reg, _ := sr.storageRegistry.(*metrics.StandardRegistry)
	reg.Each(func(name string, i interface{}) {
		if metric, ok := i.(metrics.Gauge); ok {
			tagmap := make(map[string]string)
			tagmap["controlplane_host_id"] = sr.hostID
			stats = append(stats, Sample{name, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
		}
		if metricf64, ok := i.(metrics.GaugeFloat64); ok {
			tagmap := make(map[string]string)
			tagmap["controlplane_host_id"] = sr.hostID
			stats = append(stats, Sample{name, strconv.FormatFloat(metricf64.Value(), 'f', -1, 32), t.Unix(), tagmap})
		}
	})
	return stats
}

func (ssr StorageStatsReporter) updateStats() {
	volumeStatuses := volume.GetStatus()
	if volumeStatuses == nil || len(volumeStatuses.GetAllStatuses()) == 0 {
		glog.Errorf("Unexpected error getting volume status")
		return
	}
	for _, volumeStatus := range volumeStatuses.GetAllStatuses() {
		for _, volumeUsage := range volumeStatus.GetUsageData() {
			metricName := volumeUsage.GetMetricName()

			valUint64, err := volumeUsage.GetValueUInt64()
			if err == volume.ErrWrongDataType {
				valFloat64, err := volumeUsage.GetValueFloat64()
				if err != nil {
					glog.Errorf("Error parsing volume usage %s", volumeUsage.GetType())
				} else {
					if metricName != "" {
						metrics.GetOrRegisterGaugeFloat64(metricName, ssr.storageRegistry).Update(valFloat64)
					}
				}
			} else if err != nil {
				glog.Errorf("Error parsing volume usage %s", volumeUsage.GetType())
			} else {
				if metricName != "" {
					metrics.GetOrRegisterGauge(metricName, ssr.storageRegistry).Update(int64(valUint64))
				}
			}
		}
	}
}
