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

package web

import (
	"strings"

	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

// fillBuiltinMetrics adds internal metrics to the monitoring profile
func fillBuiltinMetrics(svc *service.Service) {
	if strings.HasPrefix(svc.ID, "isvc-") {
		return
	}

	if svc.MonitoringProfile.MetricConfigs == nil {
		builder, err := domain.NewMetricConfigBuilder("/metrics/api/performance/query", "POST")
		if err != nil {
			glog.Errorf("Could not create builder to add internal metrics: %s", err)
			return
		}
		config, err := builder.Config("metrics", "metrics", "metrics", "-1h")
		if err != nil {
			glog.Errorf("could not create metric config for internal metrics: %s", err)
		}
		svc.MonitoringProfile.MetricConfigs = []domain.MetricConfig{*config}
	}
	index, found := findInternalMetricConfig(svc)
	if !found {
		glog.Errorf("should have been able to find internal metrics config")
		return
	}
	config := &svc.MonitoringProfile.MetricConfigs[index]
	removeInternalMetrics(config)
	removeInternalGraphConfigs(svc)

	if len(svc.Startup) > 2 {
		addInternalMetrics(config)
		addInternalGraphConfigs(svc)
	}
}

var internalCounterStats = []string{
	"net.collisions", "net.multicast", "net.rx_bytes", "net.rx_compressed",
	"net.rx_crc_errors", "net.rx_dropped", "net.rx_errors", "net.rx_fifo_errors",
	"net.rx_frame_errors", "net.rx_length_errors", "net.rx_missed_errors",
	"net.rx_over_errors", "net.rx_packets", "net.tx_aborted_errors",
	"net.tx_bytes", "net.tx_carrier_errors", "net.tx_compressed",
	"net.tx_dropped", "net.tx_errors", "net.tx_fifo_errors",
	"net.tx_heartbeat_errors", "net.tx_packets", "net.tx_window_errors",
	"docker.usageinkernelmode", "docker.usageinusermode", "cgroup.memory.pgmajfault",
}
var internalGaugeStats = []string{
	"cgroup.memory.totalrss", "cgroup.memory.cache", "net.open_connections.tcp", "net.open_connections.udp",
	"net.open_connections.raw",
}

func removeInternalGraphConfigs(svc *service.Service) {
	var configs []domain.GraphConfig
	for _, config := range svc.MonitoringProfile.GraphConfigs {
		if config.BuiltIn {
			continue
		}
		configs = append(configs, config)
	}
	svc.MonitoringProfile.GraphConfigs = configs
}

func addInternalGraphConfigs(svc *service.Service) {

	tags := make(map[string][]string)
	tags["controlplane_service_id"] = []string{svc.ID}

	tRange := domain.GraphConfigRange{
		Start: "1h-ago",
		End:   "0s-ago",
	}
	zero := 0
	svc.MonitoringProfile.GraphConfigs = append(
		svc.MonitoringProfile.GraphConfigs,
		domain.GraphConfig{
			ID:          "internalusage",
			Name:        "CPU Usage",
			BuiltIn:     true,
			Format:      "%4.2f",
			ReturnSet:   "EXACT",
			Type:        "area",
			Tags:        tags,
			YAxisLabel:  "% CPU Used",
			Description: "% CPU Used Over Last Hour",
			MinY:        &zero,
			Range:       &tRange,
			Units:       "Percent",
			DataPoints: []domain.DataPoint{
				domain.DataPoint{
					Aggregator:   "avg",
					Format:       "%4.2f",
					Legend:       "System",
					Metric:       "docker.usageinkernelmode",
					MetricSource: "metrics",
					ID:           "docker.usageinkernelmode",
					Name:         "System",
					Rate:         false,
					Type:         "area",
				},
				domain.DataPoint{
					Aggregator:   "avg",
					Format:       "%4.2f",
					Legend:       "User",
					Metric:       "docker.usageinusermode",
					MetricSource: "metrics",
					ID:           "docker.usageinusermode",
					Name:         "User",
					Rate:         false,
					Type:         "area",
				},
			},
		},
	)

	// memory graph
	svc.MonitoringProfile.GraphConfigs = append(
		svc.MonitoringProfile.GraphConfigs,
		domain.GraphConfig{
			ID:          "internalMemoryUsage",
			Name:        "Memory Usage",
			BuiltIn:     true,
			Format:      "%4.2f",
			ReturnSet:   "EXACT",
			Type:        "area",
			Tags:        tags,
			YAxisLabel:  "bytes",
			Description: "Memory Used Over Last Hour",
			MinY:        &zero,
			Range:       &tRange,
			Units:       "Bytes",
			Base:        1024,
			DataPoints: []domain.DataPoint{
				domain.DataPoint{
					Aggregator:   "avg",
					Fill:         true,
					Format:       "%4.2f",
					Legend:       "RSS",
					Metric:       "cgroup.memory.totalrss",
					MetricSource: "metrics",
					ID:           "cgroup.memory.totalrss",
					Name:         "Total RSS",
					Rate:         false,
					Type:         "area",
				},
				domain.DataPoint{
					Aggregator:   "avg",
					Fill:         true,
					Format:       "%4.2f",
					Legend:       "Cache",
					Metric:       "cgroup.memory.cache",
					MetricSource: "metrics",
					ID:           "cgroup.memory.cache",
					Name:         "Cache",
					Rate:         false,
					Type:         "area",
				},
			},
		},
	)

	// open conns graph
	svc.MonitoringProfile.GraphConfigs = append(
		svc.MonitoringProfile.GraphConfigs,
		domain.GraphConfig{
			ID:          "internalOpenConnections",
			Name:        "Open Connections",
			BuiltIn:     true,
			Format:      "%4.2f",
			ReturnSet:   "EXACT",
			Type:        "area",
			Tags:        tags,
			YAxisLabel:  "open connections",
			Description: "Number of Open Connections",
			MinY:        &zero,
			Range:       &tRange,
			Units:       "Connections",
			Base:        1024,
			DataPoints: []domain.DataPoint{
				domain.DataPoint{
					Aggregator:   "avg",
					Fill:         true,
					Format:       "%4.2f",
					Legend:       "TCP",
					Metric:       "net.open_connections.tcp",
					MetricSource: "metrics",
					ID:           "net.open_connections.tcp",
					Name:         "TCP Open Connections",
					Rate:         false,
					Type:         "area",
				},
				domain.DataPoint{
					Aggregator:   "avg",
					Fill:         true,
					Format:       "%4.2f",
					Legend:       "UDP",
					Metric:       "net.open_connections.udp",
					MetricSource: "metrics",
					ID:           "net.open_connections.udp",
					Name:         "UDP Open Connections",
					Rate:         false,
					Type:         "area",
				},
				domain.DataPoint{
					Aggregator:   "avg",
					Fill:         true,
					Format:       "%4.2f",
					Legend:       "RAW",
					Metric:       "net.open_connections.raw",
					MetricSource: "metrics",
					ID:           "net.open_connections.raw",
					Name:         "RAW Open Connections",
					Rate:         false,
					Type:         "area",
				},
			},
		},
	)

	// network usage graph
	svc.MonitoringProfile.GraphConfigs = append(
		svc.MonitoringProfile.GraphConfigs,
		domain.GraphConfig{
			ID:          "internalNetworkUsage",
			Name:        "Network Usage",
			BuiltIn:     true,
			Format:      "%4.2f",
			ReturnSet:   "EXACT",
			Type:        "line",
			Tags:        tags,
			YAxisLabel:  "Bps",
			Range:       &tRange,
			Description: "Bytes per second over last hour",
			MinY:        &zero,
			Units:       "Bytes per second",
			Base:        1024,
			DataPoints: []domain.DataPoint{
				domain.DataPoint{
					Aggregator:   "avg",
					Format:       "%4.2f",
					Legend:       "TX",
					Metric:       "net.tx_bytes",
					MetricSource: "metrics",
					ID:           "net.tx_bytes",
					Name:         "TX kbps",
					Rate:         true,
					RateOptions: &domain.DataPointRateOptions{
						Counter: true,
						// supress extreme outliers
						ResetThreshold: 1,
					},
					Type: "area",
				},
				domain.DataPoint{
					Aggregator:   "avg",
					Format:       "%4.2f",
					Legend:       "RX",
					Metric:       "net.rx_bytes",
					MetricSource: "metrics",
					ID:           "net.rx_bytes",
					Name:         "RX kbps",
					Rate:         true,
					RateOptions: &domain.DataPointRateOptions{
						Counter: true,
						// supress extreme outliers
						ResetThreshold: 1,
					},
					Type: "area",
				},
			},
		},
	)
}

// addInternalMetrics adds internal metrics to the config. It assumes that
// the current config does not container any internal metrics
func addInternalMetrics(config *domain.MetricConfig) {

	for _, metricName := range internalCounterStats {
		config.Metrics = append(config.Metrics,
			domain.Metric{
				ID:      metricName,
				Name:    metricName,
				Counter: true,
				BuiltIn: true,
			})

	}
	for _, metricName := range internalGaugeStats {
		config.Metrics = append(config.Metrics,
			domain.Metric{
				ID:      metricName,
				Name:    metricName,
				Counter: false,
				BuiltIn: true,
			})

	}
}

func removeInternalMetrics(config *domain.MetricConfig) {
	// create an empty list of metrics
	var metrics []domain.Metric
	for _, metric := range config.Metrics {
		// and copy metrics, except built in ones
		if metric.BuiltIn {
			continue
		}
		metrics = append(metrics, metric)
	}
	config.Metrics = metrics
}

func findInternalMetricConfig(svc *service.Service) (index int, found bool) {
	// find the metric config
	for i := range svc.MonitoringProfile.MetricConfigs {
		if svc.MonitoringProfile.MetricConfigs[i].ID == "metrics" {
			return i, true
		}
	}
	builder, err := domain.NewMetricConfigBuilder("/metrics/api/performance/query", "POST")
	if err != nil {
		glog.Errorf("Could not create builder to add internal metrics: %s", err)
		return
	}
	config, err := builder.Config("metrics", "metrics", "metrics", "-1h")
	if err != nil {
		glog.Errorf("could not create metric config for internal metrics: %s", err)
	}
	svc.MonitoringProfile.MetricConfigs = append(
		svc.MonitoringProfile.MetricConfigs,
		*config)

	return len(svc.MonitoringProfile.MetricConfigs) - 1, true
}
