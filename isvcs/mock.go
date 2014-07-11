package isvcs

import (
	// "github.com/zenoss/glog"
	"github.com/zenoss/serviced/domain"
	. "github.com/zenoss/serviced/domain/service"
)

var oneHundred int = 100
var zero int = 0

var InternalServicesISVC Service
var ElasticsearchISVC Service
var ZookeeperISVC Service
var LogstashISVC Service
var OpentsdbISVC Service
var CeleryISVC Service
var DockerRegistryISVC Service
var ISVCSMap map[string]*Service

func init() {
	InternalServicesISVC = Service{
		Name: "Internal Services",
		ID:   "isvc-internalservices",
	}
	ElasticsearchISVC = Service{
		Name:            "Elastic Search",
		ID:              "isvc-elasticsearch",
		ParentServiceID: "isvc-internalservices",
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "CpuacctStat.system", Name: "CPU System"},
						domain.Metric{ID: "CpuacctStat.user", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "MemoryStat.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   &oneHundred,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"elasticsearch"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (System)",
							Metric:     "CpuacctStat.system",
							Name:       "CPU (System)",
							Rate:       true,
							Type:       "area",
						},
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (User)",
							Metric:     "CpuacctStat.user",
							Name:       "CPU (User)",
							Rate:       true,
							Type:       "area",
						},
					},
				},
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "GB",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"elasticsearch"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "rssmemory",
							Expression: "rpn:1024,/,1024,/,1024,/",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "Memory Usage",
							Metric:     "MemoryStat.totalrss",
							Name:       "Memory Usage",
							Rate:       false,
							Type:       "area",
						},
					},
				},
			},
		},
	}
	ZookeeperISVC = Service{
		Name:            "Zookeeper",
		ID:              "isvc-zookeeper",
		ParentServiceID: "isvc-internalservices",
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "CpuacctStat.system", Name: "CPU System"},
						domain.Metric{ID: "CpuacctStat.user", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "MemoryStat.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   &oneHundred,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"zookeeper"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (System)",
							Metric:     "CpuacctStat.system",
							Name:       "CPU (System)",
							Rate:       true,
							Type:       "area",
						},
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (User)",
							Metric:     "CpuacctStat.user",
							Name:       "CPU (User)",
							Rate:       true,
							Type:       "area",
						},
					},
				},
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "GB",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"zookeeper"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "rssmemory",
							Expression: "rpn:1024,/,1024,/,1024,/",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "Memory Usage",
							Metric:     "MemoryStat.totalrss",
							Name:       "Memory Usage",
							Rate:       false,
							Type:       "area",
						},
					},
				},
			},
		},
	}
	LogstashISVC = Service{
		Name:            "Logstash",
		ID:              "isvc-logstash",
		ParentServiceID: "isvc-internalservices",
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "CpuacctStat.system", Name: "CPU System"},
						domain.Metric{ID: "CpuacctStat.user", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "MemoryStat.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   &oneHundred,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"logstash"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (System)",
							Metric:     "CpuacctStat.system",
							Name:       "CPU (System)",
							Rate:       true,
							Type:       "area",
						},
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (User)",
							Metric:     "CpuacctStat.user",
							Name:       "CPU (User)",
							Rate:       true,
							Type:       "area",
						},
					},
				},
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "GB",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"logstash"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "rssmemory",
							Expression: "rpn:1024,/,1024,/,1024,/",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "Memory Usage",
							Metric:     "MemoryStat.totalrss",
							Name:       "Memory Usage",
							Rate:       false,
							Type:       "area",
						},
					},
				},
			},
		},
	}
	OpentsdbISVC = Service{
		Name:            "OpenTSDB",
		ID:              "isvc-opentsdb",
		ParentServiceID: "isvc-internalservices",
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "CpuacctStat.system", Name: "CPU System"},
						domain.Metric{ID: "CpuacctStat.user", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "MemoryStat.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   &oneHundred,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"opentsdb"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (System)",
							Metric:     "CpuacctStat.system",
							Name:       "CPU (System)",
							Rate:       true,
							Type:       "area",
						},
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (User)",
							Metric:     "CpuacctStat.user",
							Name:       "CPU (User)",
							Rate:       true,
							Type:       "area",
						},
					},
				},
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "GB",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"opentsdb"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "rssmemory",
							Expression: "rpn:1024,/,1024,/,1024,/",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "Memory Usage",
							Metric:     "MemoryStat.totalrss",
							Name:       "Memory Usage",
							Rate:       false,
							Type:       "area",
						},
					},
				},
			},
		},
	}
	CeleryISVC = Service{
		Name:            "Celery",
		ID:              "isvc-celery",
		ParentServiceID: "isvc-internalservices",
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "CpuacctStat.system", Name: "CPU System"},
						domain.Metric{ID: "CpuacctStat.user", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "MemoryStat.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   &oneHundred,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"celery"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (System)",
							Metric:     "CpuacctStat.system",
							Name:       "CPU (System)",
							Rate:       true,
							Type:       "area",
						},
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (User)",
							Metric:     "CpuacctStat.user",
							Name:       "CPU (User)",
							Rate:       true,
							Type:       "area",
						},
					},
				},
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "GB",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"celery"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "rssmemory",
							Expression: "rpn:1024,/,1024,/,1024,/",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "Memory Usage",
							Metric:     "MemoryStat.totalrss",
							Name:       "Memory Usage",
							Rate:       false,
							Type:       "area",
						},
					},
				},
			},
		},
	}
	DockerRegistryISVC = Service{
		Name:            "Docker Registry",
		ID:              "isvc-dockerRegistry",
		ParentServiceID: "isvc-internalservices",
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "CpuacctStat.system", Name: "CPU System"},
						domain.Metric{ID: "CpuacctStat.user", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "MemoryStat.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   &oneHundred,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"docker-registry"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (System)",
							Metric:     "CpuacctStat.system",
							Name:       "CPU (System)",
							Rate:       true,
							Type:       "area",
						},
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "CPU (User)",
							Metric:     "CpuacctStat.user",
							Name:       "CPU (User)",
							Rate:       true,
							Type:       "area",
						},
					},
				},
				domain.GraphConfig{
					Footer: false,
					Format: "%d",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "GB",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"docker-registry"}},
					DownSample: "1m-avg",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "rssmemory",
							Expression: "rpn:1024,/,1024,/,1024,/",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%6.2f",
							Legend:     "Memory Usage",
							Metric:     "MemoryStat.totalrss",
							Name:       "Memory Usage",
							Rate:       false,
							Type:       "area",
						},
					},
				},
			},
		},
	}

	ISVCSMap = map[string]*Service{
		"isvc-internalservices": &InternalServicesISVC,
		"isvc-elasticsearch":    &ElasticsearchISVC,
		"isvc-zookeeper":        &ZookeeperISVC,
		"isvc-logstash":         &LogstashISVC,
		"isvc-opentsdb":         &OpentsdbISVC,
		"isvc-celery":           &CeleryISVC,
		"isvc-dockerRegistry":   &DockerRegistryISVC,
	}

}
