// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"github.com/control-center/serviced/domain"
)

//profile defines meta-data for the host/pool resource's metrics and graphs
var (
	zero       int = 0
	onehundred int = 100

	hostPoolProfile = domain.MonitorProfile{
		MetricConfigs: []domain.MetricConfig{
			//Loadavg
			domain.MetricConfig{
				ID:          "load",
				Name:        "Load Average",
				Description: "Load average stats",
				Metrics: []domain.Metric{
					domain.Metric{ID: "load.avg1m", Name: "1m Loadavg", Unit: "p"},
					domain.Metric{ID: "load.avg5m", Name: "5m Loadavg", Unit: "p"},
					domain.Metric{ID: "load.avg10m", Name: "10m Loadavg", Unit: "p"},
					domain.Metric{ID: "load.runningprocesses", Name: "Running Processes", Unit: "p"},
					domain.Metric{ID: "load.totalprocesses", Name: "Total Processes", Unit: "p"},
				},
			},
			//CPU
			domain.MetricConfig{
				ID:          "cpu",
				Name:        "CPU Usage",
				Description: "CPU Statistics",
				Metrics: []domain.Metric{
					domain.Metric{ID: "cpu.user", Name: "CPU User", Unit: "Percent", Counter: true},
					domain.Metric{ID: "cpu.nice", Name: "CPU Nice", Unit: "Percent", Counter: true},
					domain.Metric{ID: "cpu.system", Name: "CPU System", Unit: "Percent", Counter: true},
					domain.Metric{ID: "cpu.idle", Name: "CPU Idle", Unit: "Percent", Counter: true},
					domain.Metric{ID: "cpu.iowait", Name: "CPU IO Wait", Unit: "Percent", Counter: true},
					domain.Metric{ID: "cpu.irq", Name: "IRQ", Unit: "Percent", Counter: true},
					domain.Metric{ID: "cpu.softirq", Name: "Soft IRQ", Unit: "Percent", Counter: true},
					domain.Metric{ID: "cpu.steal", Name: "CPU Steal", Unit: "Percent", Counter: true},
				},
			},
			//Memory
			domain.MetricConfig{
				ID:          "memory",
				Name:        "Memory Usage",
				Description: "Memory Usage Statistics -- /proc/meminfo",
				Metrics: []domain.Metric{
					domain.Metric{ID: "memory.buffers", Name: "Memory Buffer", Unit: "Bytes"},
					domain.Metric{ID: "memory.cached", Name: "Memory Cache", Unit: "Bytes"},
					domain.Metric{ID: "memory.free", Name: "Memory Free", Unit: "Bytes"},
					domain.Metric{ID: "memory.total", Name: "Total Memory", Unit: "Bytes"},
					domain.Metric{ID: "memory.actualfree", Name: "Actual Free Memory", Unit: "Bytes"},
					domain.Metric{ID: "memory.actualused", Name: "Actual Used Memory", Unit: "Bytes"},
					domain.Metric{ID: "swap.total", Name: "Total Swap", Unit: "Bytes"},
					domain.Metric{ID: "swap.free", Name: "Free Swap", Unit: "Bytes"},
				},
			},
			//Virtual Memory
			domain.MetricConfig{
				ID:          "virtual.memory",
				Name:        "Virtual Memory Usage",
				Description: "Virtual Memory Usage Statistics -- /proc/vmstat",
				Metrics: []domain.Metric{
					domain.Metric{ID: "vmstat.pgfault", Name: "Minor Page Fault", Unit: "Page Faults", Counter: true},
					domain.Metric{ID: "vmstat.pgmajfault", Name: "Major Page Fault", Unit: "Page Faults", Counter: true},
					domain.Metric{ID: "vmstat.pgpgout", Name: "Bytes paged out", Unit: "Bytes", Counter: true},
					domain.Metric{ID: "vmstat.pgpgin", Name: "Bytes paged in", Unit: "Bytes", Counter: true},
					domain.Metric{ID: "vmstat.pswpout", Name: "Bytes swapped out", Unit: "Bytes", Counter: true},
					domain.Metric{ID: "vmstat.pswpin", Name: "Bytes swapped in", Unit: "Bytes", Counter: true},
				},
			},
			//Files
			domain.MetricConfig{
				ID:          "files",
				Name:        "File Usage",
				Description: "File Statistics",
				Metrics: []domain.Metric{
					domain.Metric{ID: "Serviced.OpenFileDescriptors", Name: "OpenFileDescriptors", Unit: "Open File Descriptors"},
				},
			},
		},
		ThresholdConfigs: []domain.ThresholdConfig{
			domain.ThresholdConfig{
				ID:           "swap.empty",
				Name:         "Swap empty",
				Description:  "Alert when swap reaches zero",
				MetricSource: "memory",
				DataPoints:   []string{"swap.free", "memory.free"},
				Type:         "MinMax",
				Threshold:    domain.MinMaxThreshold{Min: "0", Max: ""},
				EventTags: map[string]interface{}{
					"Severity":    1,
					"Resolution":  "Increase swap or memory",
					"Explanation": "Ran out of all available memory space",
					"EventClass":  "/Perf/Memory",
				},
			},
		},
	}
)

//profile defines meta-data for the volume resource's metrics and graphs
var (
	tenGb = int64(10000000000)

	volumeProfile = domain.MonitorProfile{
		MetricConfigs: []domain.MetricConfig{
			domain.MetricConfig{
				ID: "storage",
				Name: "storage",
				Description: "DFS usage data",
				Metrics: []domain.Metric{
					domain.Metric{ID: "storage.total", Name: "total", Unit: "bytes"},
					domain.Metric{ID: "storage.used", Name: "used", Unit: "bytes"},
				},
			},
		},
		ThresholdConfigs: []domain.ThresholdConfig{
			domain.ThresholdConfig{
				ID: "dfs.space.low",
				Name: "DFS space low",
				Description: "DFS free space is low",
				MetricSource: "storage",
				DataPoints: []string{"storage.used"},
				Type: "MinMax",
				Threshold:    domain.MinMaxThreshold{Min: "", Max: "here.totalBytes * 0.80"},
				EventTags: map[string]interface{}{
					"Severity":    3,
					"Resolution":  "Increase free space on the DFS",
					"Explanation": "Low on free space on the DFS",
					"EventClass":  "/Storage/Full",
				},
			},
			domain.ThresholdConfig{
				ID: "dfs.space.very.low",
				Name: "DFS space very low",
				Description: "DFS free space is very low",
				MetricSource: "storage",
				DataPoints: []string{"storage.used"},
				Type: "MinMax",
				Threshold:    domain.MinMaxThreshold{Min: "", Max: "here.totalBytes * 0.90"},
				EventTags: map[string]interface{}{
					"Severity":    4,
					"Resolution":  "Increase free space on the DFS",
					"Explanation": "Very low on free space on the DFS",
					"EventClass":  "/Storage/Full",
				},
			},
		},
	}
)

//Open File Descriptors
func newOpenFileDescriptorsGraph(tags map[string][]string) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				ID:           "ofd",
				Aggregator:   "avg",
				Color:        "#aec7e8",
				Fill:         false,
				Format:       "%4.2f",
				Legend:       "Serviced Open File Descriptors",
				Metric:       "Serviced.OpenFileDescriptors",
				MetricSource: "files",
				Name:         "Serviced Open File Descriptors",
				Rate:         false,
				Type:         "line",
			},
		},
		ID:     "serviced.ofd",
		Name:   "Open File Descriptors",
		Footer: false,
		Format: "%4.2f",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		ReturnSet:   "EXACT",
		Type:        "line",
		Tags:        tags,
		Units:       "File Descriptors",
		Description: "For the serviced process",
	}
}

//Major Page Faults
func newMajorPageFaultGraph(tags map[string][]string) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				Aggregator:   "avg",
				ID:           "pgfault",
				Color:        "#aec7e8",
				Fill:         false,
				Format:       "%4.2f",
				Legend:       "Major Page Faults",
				Metric:       "vmstat.pgmajfault",
				MetricSource: "virtual.memory",
				Name:         "Major Page Faults",
				Rate:         true,
				Type:         "line",
			},
		},
		ID:     "memory.major.pagefault",
		Name:   "Memory Major Page Faults",
		Footer: false,
		Format: "%.2f",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel:  "Faults / Min",
		ReturnSet:   "EXACT",
		Type:        "line",
		Tags:        tags,
		Units:       "Page Faults",
		Description: "Faults per minute",
	}
}

// Paging graph
func newPagingGraph(tags map[string][]string) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				ID:           "vmstat.pgpgout",
				Aggregator:   "avg",
				Fill:         false,
				Legend:       "page out",
				Metric:       "vmstat.pgpgout",
				MetricSource: "vmstat",
				Name:         "page out",
				Rate:         true,
				Type:         "line",
			},
			domain.DataPoint{
				ID:           "vmstat.pgpgin",
				Aggregator:   "avg",
				Fill:         false,
				Legend:       "page in",
				Metric:       "vmstat.pgpgin",
				MetricSource: "vmstat",
				Name:         "page in",
				Rate:         true,
				Type:         "line",
			},
			domain.DataPoint{
				ID:           "vmstat.pswpout",
				Aggregator:   "avg",
				Fill:         false,
				Legend:       "swap out",
				Metric:       "vmstat.pswpout",
				MetricSource: "vmstat",
				Name:         "swap out",
				Rate:         true,
				Type:         "line",
			},
			domain.DataPoint{
				ID:           "vmstat.pswpin",
				Aggregator:   "avg",
				Fill:         false,
				Legend:       "swap in",
				Metric:       "vmstat.pswpin",
				MetricSource: "vmstat",
				Name:         "swap in",
				Rate:         true,
				Type:         "line",
			},
		},
		ID:     "paging",
		Name:   "Paging",
		Footer: false,
		Format: "%4.2f",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		ReturnSet:   "EXACT",
		Type:        "line",
		Tags:        tags,
		Units:       "bytes",
		Description: "System paging",
	}
}

// Load average graphs
func newLoadAverageGraph(tags map[string][]string) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				ID:           "load.avg1m",
				Aggregator:   "avg",
				Fill:         false,
				Legend:       "1m loadavg",
				Metric:       "load.avg1m",
				MetricSource: "load",
				Name:         "1m Loadavg",
				Rate:         false,
				Type:         "line",
			},
			domain.DataPoint{
				ID:           "load.avg5m",
				Aggregator:   "avg",
				Fill:         false,
				Legend:       "5m loadavg",
				Metric:       "load.avg5m",
				MetricSource: "load",
				Name:         "5m Loadavg",
				Rate:         false,
				Type:         "line",
			},
			domain.DataPoint{
				ID:           "load.avg10m",
				Aggregator:   "avg",
				Fill:         false,
				Legend:       "10m loadavg",
				Metric:       "load.avg10m",
				MetricSource: "load",
				Name:         "10m Loadavg",
				Rate:         false,
				Type:         "line",
			},
		},
		ID:     "loadavg",
		Name:   "Load Average",
		Footer: false,
		Format: "%4.2f",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		ReturnSet:   "EXACT",
		Type:        "line",
		Tags:        tags,
		Units:       "processes",
		Description: "Host load average",
	}
}

//Cpu Usage
func newCpuConfigGraph(tags map[string][]string, totalCores int) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#aee8cf",
				Fill:         false,
				Format:       "%4.2f",
				ID:           "user",
				Legend:       "User",
				Metric:       "cpu.user",
				MetricSource: "cpu",
				Name:         "User",
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
				Color:        "#729ed7",
				Fill:         false,
				Format:       "%4.2f",
				ID:           "system",
				Legend:       "System",
				Metric:       "cpu.system",
				MetricSource: "cpu",
				Name:         "System",
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
				Fill:         false,
				Color:        "#d7729e",
				Format:       "%4.2f",
				ID:           "nice",
				Legend:       "Nice",
				Metric:       "cpu.nice",
				MetricSource: "cpu",
				Name:         "Nice",
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
				Color:        "#e8aec7",
				Fill:         false,
				Format:       "%4.2f",
				ID:           "iowait",
				Legend:       "IOWait",
				Metric:       "cpu.iowait",
				MetricSource: "cpu",
				Name:         "IOWait",
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
				Color:        "#e8cfae",
				Fill:         false,
				Format:       "%4.2f",
				ID:           "irq",
				Legend:       "IRQ",
				Metric:       "cpu.irq",
				MetricSource: "cpu",
				Name:         "IRQ",
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
				Color:        "#ff0000",
				Fill:         false,
				Format:       "%4.2f",
				ID:           "steal",
				Legend:       "Steal",
				Metric:       "cpu.steal",
				MetricSource: "cpu",
				Name:         "Steal",
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
				Color:        "#EBE6EA",
				Fill:         false,
				Format:       "%6.2f",
				ID:           "idle",
				Legend:       "Idle",
				Metric:       "cpu.idle",
				MetricSource: "cpu",
				Name:         "Idle",
				Rate:         true,
				RateOptions: &domain.DataPointRateOptions{
					Counter: true,
					// supress extreme outliers
					ResetThreshold: 1,
				},
				Type: "area",
			},
		},
		ID:     "cpu.usage",
		Name:   "CPU Usage",
		Footer: false,
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel:  "% Used",
		ReturnSet:   "EXACT",
		Type:        "area",
		Tags:        tags,
		Units:       "Percent",
		Description: "Total cpu utilization (all cores)",
	}
}

func newRSSConfigGraph(tags map[string][]string, totalMemory uint64) domain.GraphConfig {
	MaxY := int(totalMemory)
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#e8aec7",
				Fill:         true,
				Format:       "%4.2f",
				Legend:       "Used",
				Metric:       "memory.actualused",
				MetricSource: "memory",
				Name:         "Used",
				Type:         "area",
				ID:           "used",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#b2aee8",
				Fill:         true,
				Format:       "%4.2f",
				Legend:       "Cached",
				Metric:       "memory.cached",
				MetricSource: "memory",
				Name:         "Cached",
				Type:         "area",
				ID:           "cached",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#aec7e8",
				Fill:         true,
				Format:       "%4.2f",
				Legend:       "Buffers",
				Metric:       "memory.buffers",
				MetricSource: "memory",
				Name:         "Buffers",
				Type:         "area",
				ID:           "buffers",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#aee4e8",
				Fill:         true,
				Format:       "%4.2f",
				Legend:       "Free",
				Metric:       "memory.free",
				MetricSource: "memory",
				Name:         "Free",
				Type:         "area",
				ID:           "free",
			},
		},
		ID:     "memory.usage",
		Name:   "Memory Usage",
		Footer: false,
		Format: "%4.2f",
		MaxY:   &MaxY,
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel:  "bytes",
		ReturnSet:   "EXACT",
		Type:        "area",
		Tags:        tags,
		Units:       "bytes",
		Base:        1024,
		Description: "Bytes used",
	}
}

//Major Page Faults
func newVolumeUsageGraph(tags map[string][]string) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				Aggregator:   "avg",
				ID:           "used",
				Color:        "#aec7e8",
				Fill:         false,
				Format:       "%4.2f",
				Legend:       "Used Bytes",
				Metric:       "storage.used",
				MetricSource: "storage",
				Name:         "Used Bytes",
				Type:         "line",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				ID:           "total",
				Color:        "#aee4e8",
				Fill:         false,
				Format:       "%4.2f",
				Legend:       "Total Bytes",
				Metric:       "storage.total",
				MetricSource: "storage",
				Name:         "Total Bytes",
				Type:         "line",
			},
		},
		ID:     "storage.usage",
		Name:   "DFS Usage",
		Footer: false,
		Format: "%.2f",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel:  "Bytes",
		ReturnSet:   "EXACT",
		Type:        "line",
		Tags:        tags,
		Units:       "Bytes",
		Description: "DFS usage",
	}
}
