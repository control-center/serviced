package web

import (
	"github.com/zenoss/serviced/domain"

	"fmt"
)

//profile defines meta-data for the host/pool resource's metrics and graphs
var (
	zero       int = 0
	onehundred int = 100

	profile = domain.MonitorProfile{
		MetricConfigs: []domain.MetricConfig{
			//CPU
			domain.MetricConfig{
				ID:          "cpu",
				Name:        "CPU Usage",
				Description: "CPU Statistics",
				Metrics: []domain.Metric{
					domain.Metric{ID: "cpu.system", Name: "CPU System"},
					domain.Metric{ID: "cpu.user", Name: "CPU User"},
					domain.Metric{ID: "cpu.idle", Name: "CPU Idle"},
					domain.Metric{ID: "cpu.iowait", Name: "CPU IO Wait"},
					domain.Metric{ID: "cpu.nice", Name: "CPU Nice"},
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
					domain.Metric{ID: "memory.used", Name: "Used Memory"},
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
	}
)

//Open File Descriptors
func newOpenFileDescriptorsGraph(tags map[string][]string) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				ID:         "ofd",
				Aggregator: "avg",
				Color:      "#aec7e8",
				Fill:       false,
				Format:     "%6.2f",
				Legend:     "Serviced Open File Descriptors",
				Metric:     "Serviced.OpenFileDescriptors",
				Name:       "Serviced Open File Descriptors",
				Rate:       false,
				Type:       "line",
			},
		},
		Footer: false,
		Format: "%d",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		ReturnSet:  "EXACT",
		Type:       "line",
		DownSample: "1m-avg",
		Tags:       tags,
	}
}

//Major Page Faults
func newMajorPageFaultGraph(tags map[string][]string) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				Aggregator: "avg",
				ID:         "pgfault",
				Color:      "#aec7e8",
				Fill:       false,
				Format:     "%d",
				Legend:     "Major Page Faults",
				Metric:     "vmstat.pgmajfault",
				Name:       "Major Page Faults",
				Rate:       true,
				Type:       "line",
			},
		},
		Footer: false,
		Format: "%d",
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel: "Faults / Min",
		ReturnSet:  "EXACT",
		Type:       "line",
		DownSample: "1m-avg",
		Tags:       tags,
	}
}

//Cpu Usage
func newCpuConfigGraph(tags map[string][]string, totalCores int) domain.GraphConfig {
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				Aggregator: "avg",
				Color:      "#aec7e8",
				Expression: fmt.Sprintf("rpn:%d,/,100,*,60,/", totalCores),
				Fill:       false,
				Format:     "%6.2f",
				ID:         "system",
				Legend:     "CPU (System)",
				Metric:     "cpu.system",
				Name:       "CPU (System)",
				Rate:       true,
				Type:       "line",
			},
			domain.DataPoint{
				Aggregator: "avg",
				Color:      "#98df8a",
				Expression: fmt.Sprintf("rpn:%d,/,100,*,60,/", totalCores),
				ID:         "user",
				Fill:       false,
				Format:     "%6.2f",
				Legend:     "CPU (User)",
				Metric:     "cpu.user",
				Name:       "CPU (User)",
				Rate:       true,
				Type:       "line",
			},
		},
		Footer: false,
		Format: "%d",
		MinY:   &zero,
		MaxY:   &onehundred,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel: "% Used",
		ReturnSet:  "EXACT",
		Type:       "line",
		DownSample: "1m-avg",
		Tags:       tags,
	}
}

func newRSSConfigGraph(tags map[string][]string, totalMemory uint64) domain.GraphConfig {
	MaxY := int(totalMemory / 1024 / 1024 / 1024)
	return domain.GraphConfig{
		DataPoints: []domain.DataPoint{
			domain.DataPoint{
				Aggregator: "avg",
				Expression: "rpn:1024,/,1024,/,1024,/",
				Color:      "#aec7e8",
				Fill:       true,
				Format:     "%6.2f",
				Legend:     "Used",
				Metric:     "memory.used",
				Name:       "RSS",
				Type:       "area",
				ID:         "used",
			},
			domain.DataPoint{
				Aggregator: "avg",
				Expression: "rpn:1024,/,1024,/,1024,/",
				Color:      "#98df8a",
				Fill:       true,
				Format:     "%6.2f",
				Legend:     "Cache",
				Metric:     "memory.free",
				Name:       "Free",
				ID:         "Memory",
				Type:       "area",
			},
		},
		Footer: false,
		Format: "%6.2f",
		MaxY:   &MaxY,
		MinY:   &zero,
		Range: &domain.GraphConfigRange{
			End:   "0s-ago",
			Start: "1h-ago",
		},
		YAxisLabel: "GB",
		ReturnSet:  "EXACT",
		Type:       "line",
		DownSample: "1m-avg",
		Tags:       tags,
	}
}

//newProfile builds a MonitoringProfile without graphs pools
func newProfile(tags map[string][]string) (domain.MonitorProfile, error) {
	p := domain.MonitorProfile{
		MetricConfigs: make([]domain.MetricConfig, len(profile.MetricConfigs)),
	}

	build, err := domain.NewMetricConfigBuilder("/metrics/api/performance/query", "POST")
	if err != nil {
		return p, err
	}

	//add metrics to profile
	for i := range profile.MetricConfigs {
		metricConfig := &profile.MetricConfigs[i]
		for j := range metricConfig.Metrics {
			metric := &metricConfig.Metrics[j]
			build.Metric(metric.ID, metric.Name).SetTags(tags)
		}

		config, err := build.Config(metricConfig.ID, metricConfig.Name, metricConfig.Description, "1h-ago")
		if err != nil {
			return p, err
		}
		p.MetricConfigs[i] = *config
	}
	return p, nil
}

//newProfile builds a MonitoringProfile with graphs for hosts
func newProfileWithGraphs(tags map[string][]string, totalCores int, totalMemory uint64) (domain.MonitorProfile, error) {
	p, err := newProfile(tags)
	if err != nil {
		return p, err
	}

	//add graphs to profile
	p.GraphConfigs = make([]domain.GraphConfig, 4)
	p.GraphConfigs[0] = newOpenFileDescriptorsGraph(tags)
	p.GraphConfigs[1] = newMajorPageFaultGraph(tags)
	p.GraphConfigs[2] = newCpuConfigGraph(tags, totalCores)
	p.GraphConfigs[3] = newRSSConfigGraph(tags, totalMemory)
	return p, nil
}
