package web

import (
	"github.com/zenoss/serviced/domain"
)

//profile defines meta-data for the host/pool resource's metrics and graphs
var (
	zero       int = 0
	onehundred int = 100

	zeroInt64 int64 = 0

	hostPoolProfile = domain.MonitorProfile{
		MetricConfigs: []domain.MetricConfig{
			//CPU
			domain.MetricConfig{
				ID:          "cpu",
				Name:        "CPU Usage",
				Description: "CPU Statistics",
				Metrics: []domain.Metric{
					domain.Metric{ID: "cpu.user", Name: "CPU User"},
					domain.Metric{ID: "cpu.nice", Name: "CPU Nice"},
					domain.Metric{ID: "cpu.system", Name: "CPU System"},
					domain.Metric{ID: "cpu.idle", Name: "CPU Idle"},
					domain.Metric{ID: "cpu.iowait", Name: "CPU IO Wait"},
					domain.Metric{ID: "cpu.irq", Name: "IRQ"},
					domain.Metric{ID: "cpu.softirq", Name: "Soft IRQ"},
					domain.Metric{ID: "cpu.steal", Name: "CPU Steal"},
				},
			},
			//Memory
			domain.MetricConfig{
				ID:          "memory",
				Name:        "Memory Usage",
				Description: "Memory Usage Statistics -- /proc/meminfo",
				Metrics: []domain.Metric{
					domain.Metric{ID: "memory.buffers", Name: "Memory Buffer"},
					domain.Metric{ID: "memory.cached", Name: "Memory Cache"},
					domain.Metric{ID: "memory.free", Name: "Memory Free"},
					domain.Metric{ID: "memory.total", Name: "Total Memory"},
					domain.Metric{ID: "memory.actualfree", Name: "Actual Free Memory"},
					domain.Metric{ID: "memory.actualused", Name: "Actual Used Memory"},
					domain.Metric{ID: "swap.total", Name: "Total Swap"},
					domain.Metric{ID: "swap.free", Name: "Free Swap"},
				},
			},
			//Virtual Memory
			domain.MetricConfig{
				ID:          "virtual.memory",
				Name:        "Virtual Memory Usage",
				Description: "Virtual Memory Usage Statistics -- /proc/vmstat",
				Metrics: []domain.Metric{
					domain.Metric{ID: "vmstat.pgfault", Name: "Minor Page Fault"},
					domain.Metric{ID: "vmstat.pgmajfault", Name: "Major Page Fault"},
				},
			},
			//Files
			domain.MetricConfig{
				ID:          "files",
				Name:        "File Usage",
				Description: "File Statistics",
				Metrics: []domain.Metric{
					domain.Metric{ID: "Serviced.OpenFileDescriptors", Name: "OpenFileDescriptors"},
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
				Threshold:    domain.MinMaxThreshold{Min: &zeroInt64, Max: nil},
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

//Open File Descriptors
func newOpenFileDescriptorsGraph(tags map[string][]string) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				ID:           "ofd",
				Aggregator:   "avg",
				Color:        "#aec7e8",
				Fill:         false,
				Format:       "%6.2f",
				Legend:       "Serviced Open File Descriptors",
				Metric:       "Serviced.OpenFileDescriptors",
				MetricSource: "files",
				Name:         "Serviced Open File Descriptors",
				Rate:         false,
				Type:         "line",
			},
		},
		ID:     "serviced.ofd",
		Name:   "Serviced Open File Descriptors",
		Footer: false,
		Format: "%d",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		ReturnSet:   "EXACT",
		Type:        "line",
		Tags:        tags,
		Description: "Graph of serviced's total open file descriptors over time",
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
				Format:       "%d",
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
		Format: "%d",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel:  "Faults / Min",
		ReturnSet:   "EXACT",
		Type:        "line",
		Tags:        tags,
		Description: "Graph of major memory page faults over time",
	}
}

//Cpu Usage
func newCpuConfigGraph(tags map[string][]string, totalCores int) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#729ed7",
				Fill:         false,
				Format:       "%6.2f",
				ID:           "nice",
				Legend:       "Nice",
				Metric:       "cpu.nice",
				MetricSource: "cpu",
				Name:         "Nice",
				Rate:         true,
				Type:         "area",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#aee8cf",
				Fill:         false,
				Format:       "%6.2f",
				ID:           "user",
				Legend:       "User",
				Metric:       "cpu.user",
				MetricSource: "cpu",
				Name:         "User",
				Rate:         true,
				Type:         "area",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#eaf0f9",
				Fill:         false,
				Format:       "%6.2f",
				ID:           "idle",
				Legend:       "Idle",
				Metric:       "cpu.idle",
				MetricSource: "cpu",
				Name:         "Idle",
				Rate:         true,
				Type:         "area",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#d7729e",
				Fill:         false,
				Format:       "%6.2f",
				ID:           "system",
				Legend:       "System",
				Metric:       "cpu.system",
				MetricSource: "cpu",
				Name:         "System",
				Rate:         true,
				Type:         "area",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#e8aec7",
				Fill:         false,
				Format:       "%6.2f",
				ID:           "iowait",
				Legend:       "IOWait",
				Metric:       "cpu.iowait",
				MetricSource: "cpu",
				Name:         "IOWait",
				Rate:         true,
				Type:         "area",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#e8cfae",
				Fill:         false,
				Format:       "%6.2f",
				ID:           "irq",
				Legend:       "IRQ",
				Metric:       "cpu.irq",
				MetricSource: "cpu",
				Name:         "IRQ",
				Rate:         true,
				Type:         "area",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Color:        "#ff0000",
				Fill:         false,
				Format:       "%6.2f",
				ID:           "steal",
				Legend:       "Steal",
				Metric:       "cpu.steal",
				MetricSource: "cpu",
				Name:         "Steal",
				Rate:         true,
				Type:         "area",
			},
		},
		ID:     "cpu.usage",
		Name:   "CPU Usage",
		Footer: false,
		Format: "%d",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel:  "% Used",
		ReturnSet:   "EXACT",
		Type:        "area",
		Tags:        tags,
		Description: "Graph of system and user cpu usage over time",
	}
}

func newRSSConfigGraph(tags map[string][]string, totalMemory uint64) domain.GraphConfig {
	MaxY := int(totalMemory / 1024 / 1024 / 1024)
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				Aggregator:   "avg",
				Expression:   "rpn:1024,/,1024,/,1024,/",
				Color:        "#e8aec7",
				Fill:         true,
				Format:       "%6.2f",
				Legend:       "Used",
				Metric:       "memory.actualused",
				MetricSource: "memory",
				Name:         "Used",
				Type:         "area",
				ID:           "used",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Expression:   "rpn:1024,/,1024,/,1024,/",
				Color:        "#b2aee8",
				Fill:         true,
				Format:       "%6.2f",
				Legend:       "Cached",
				Metric:       "memory.cached",
				MetricSource: "memory",
				Name:         "Cached",
				Type:         "area",
				ID:           "cached",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Expression:   "rpn:1024,/,1024,/,1024,/",
				Color:        "#aec7e8",
				Fill:         true,
				Format:       "%6.2f",
				Legend:       "Buffers",
				Metric:       "memory.buffers",
				MetricSource: "memory",
				Name:         "Buffers",
				Type:         "area",
				ID:           "buffers",
			},
			domain.DataPoint{
				Aggregator:   "avg",
				Expression:   "rpn:1024,/,1024,/,1024,/",
				Color:        "#aee4e8",
				Fill:         true,
				Format:       "%6.2f",
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
		Format: "%6.2f",
		MaxY:   &MaxY,
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel:  "GB",
		ReturnSet:   "EXACT",
		Type:        "area",
		Tags:        tags,
		Description: "Graph of memory free (-buffers/+cache) vs used (total - free) over time",
	}
}
