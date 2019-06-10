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

package isvcs

import (
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	s "github.com/control-center/serviced/domain/service"
)

var zero int = 0

var InternalServicesISVC s.Service
var ElasticsearchLogStashISVC s.Service
var ElasticsearchServicedISVC s.Service
var ZookeeperISVC s.Service
var LogstashISVC s.Service
var OpentsdbISVC s.Service
var DockerRegistryISVC s.Service
var KibanaISVC s.Service
var ApiKeyProxyISVC s.Service
var ISVCSMap map[string]*s.Service

var InternalServicesIRS dao.RunningService
var ElasticsearchLogStashIRS dao.RunningService
var ElasticsearchServicedIRS dao.RunningService
var ZookeeperIRS dao.RunningService
var LogstashIRS dao.RunningService
var OpentsdbIRS dao.RunningService
var DockerRegistryIRS dao.RunningService
var KibanaIRS dao.RunningService
var ApiKeyProxyIRS dao.RunningService
var IRSMap map[string]*dao.RunningService

func init() {
	InternalServicesIRS = dao.RunningService{
		Name:         "Internal Services",
		Description:  "Internal Services",
		ID:           "isvc-internalservices",
		ServiceID:    "isvc-internalservices",
		DesiredState: 1,
		StartedAt:    time.Now(),
	}

	tags := map[string][]string{"isvc": []string{"true"}}

	InternalServicesISVC = s.Service{
		Name:         "Internal Services",
		ID:           "isvc-internalservices",
		Startup:      "N/A",
		Description:  "Internal Services",
		DeploymentID: "Internal",
		DesiredState: 1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		MonitoringProfile: domain.MonitorProfile{
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					ID:     "cpuUsage",
					Name:   "CPU Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       tags,
					Units:      "Percent",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%4.2f",
							Legend:     "CPU (System)",
							Metric:     "docker.usageinkernelmode",
							Name:       "CPU (System)",
							Rate:       false,
							Type:       "area",
						},
						domain.DataPoint{
							ID:         "system",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%4.2f",
							Legend:     "CPU (User)",
							Metric:     "docker.usageinusermode",
							Name:       "CPU (User)",
							Rate:       false,
							Type:       "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "memoryUsage",
					Name:   "Memory Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "bytes",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       tags,
					Units:      "Bytes",
					Base:       1024,
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:         "rssmemory",
							Aggregator: "avg",
							Fill:       false,
							Format:     "%4.2f",
							Legend:     "Memory Usage",
							Metric:     "cgroup.memory.totalrss",
							Name:       "Memory Usage",
							Rate:       false,
							Type:       "area",
						},
					},
				},
			},
		},
	}
	ElasticsearchLogStashIRS = dao.RunningService{
		Name:         "Elastic Search - LogStash",
		Description:  "Internal Elastic Search - LogStash",
		ID:           "isvc-elasticsearch-logstash",
		ServiceID:    "isvc-elasticsearch-logstash",
		DesiredState: 1,
		StartedAt:    time.Now(),
	}
	ElasticsearchServicedIRS = dao.RunningService{
		Name:         "Elastic Search - Serviced",
		Description:  "Internal Elastic Search - Serviced",
		ID:           "isvc-elasticsearch-serviced",
		ServiceID:    "isvc-elasticsearch-serviced",
		DesiredState: 1,
		StartedAt:    time.Now(),
	}
	ElasticsearchLogStashISVC = s.Service{
		Name:            "Elastic Search - LogStash",
		ID:              "isvc-elasticsearch-logstash",
		Startup:         "/opt/elasticsearch-2.3.3/bin/elasticsearch",
		Description:     "Internal Elastic Search - LogStash",
		ParentServiceID: "isvc-internalservices",
		DesiredState:    1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "metrics",
					Name:        "Metrics",
					Description: "Metrics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "docker.usageinkernelmode", Name: "CPU System"},
						domain.Metric{ID: "docker.usageinusermode", Name: "CPU User"},
						domain.Metric{ID: "cgroup.memory.totalrss", Name: "Total RSS Memory"},
					},
				},
				domain.MetricConfig{
					ID:          "isvcs.jvm.gc",
					Name:        "JVM Garbage Collection",
					Description: "JVM Garbage Collection Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "isvcs.jvm.gc.old.collection_time", Name: "JVM Garbage Collection Old Generation Time"},
						domain.Metric{ID: "isvcs.jvm.gc.old.collection_count", Name: "JVM Garbage Collection Old Generation Runs"},
						domain.Metric{ID: "isvcs.jvm.gc.young.collection_time", Name: "JVM Garbage Collection Young Generation Time"},
						domain.Metric{ID: "isvcs.jvm.gc.young.collection_count", Name: "JVM Garbage Collection Young Generation Runs"},
					},
				},
				domain.MetricConfig{
					ID:          "isvcs.jvm.threads",
					Name:        "JVM Thread",
					Description: "JVM Thread Type Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "isvcs.jvm.threads.count", Name: "JVM Thread Count"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					ID:     "cpuUsage",
					Name:   "CPU Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"elasticsearch-logstash"}},
					Units:      "Percent",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "system",
							MetricSource: "metrics",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (System)",
							Metric:       "docker.usageinkernelmode",
							Name:         "CPU (System)",
							Rate:         false,
							Type:         "area",
						},
						domain.DataPoint{
							ID:           "system",
							MetricSource: "metrics",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (User)",
							Metric:       "docker.usageinusermode",
							Name:         "CPU (User)",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "memoryUsage",
					Name:   "Memory Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "bytes",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"elasticsearch-logstash"}},
					Units:      "Bytes",
					Base:       1024,
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "rssmemory",
							MetricSource: "metrics",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "Memory Usage",
							Metric:       "cgroup.memory.totalrss",
							Name:         "Memory Usage",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "jvm_gc_runs",
					Name:   "JVM Garbage Collection Runs",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "count",
					ReturnSet:  "EXACT",
					Type:       "line",
					Tags:       map[string][]string{"controlplane_service_id": []string{"isvc-elasticsearch-logstash"}},
					Units:      "Count",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "jvm_gc_runs",
							MetricSource: "isvcs.jvm.gc",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "JVM GC Old Generation Runs",
							Metric:       "isvcs.jvm.gc.old.collection_count",
							Name:         "JVM GC Old Generation Runs",
							Rate:         false,
							Type:         "line",
						},
						domain.DataPoint{
							ID:           "jvm_gc_runs",
							MetricSource: "isvcs.jvm.gc",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "JVM GC Young Generation Runs",
							Metric:       "isvcs.jvm.gc.young.collection_count",
							Name:         "JVM GC Young Generation Runs",
							Rate:         false,
							Type:         "line",
						},
					},
				},
				domain.GraphConfig{
					ID:     "jvm_gc_time",
					Name:   "JVM Garbage Collection Time",
					Footer: false,
					Format: "%.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "seconds",
					ReturnSet:  "EXACT",
					Type:       "line",
					Tags:       map[string][]string{"controlplane_service_id": []string{"isvc-elasticsearch-logstash"}},
					Units:      "Miliseconds",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "jvm_gc_time",
							MetricSource: "isvcs.jvm.gc",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%S",
							Legend:       "JVM GC Old Generation Time",
							Metric:       "isvcs.jvm.gc.old.collection_time",
							Name:         "JVM GC Old Generation Time",
							Rate:         false,
							Type:         "line",
						},
						domain.DataPoint{
							ID:           "jvm_gc_time",
							MetricSource: "isvcs.jvm.gc",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%S",
							Legend:       "JVM GC Young Generation Time",
							Metric:       "isvcs.jvm.gc.young.collection_time",
							Name:         "JVM GC Young Generation Time",
							Rate:         false,
							Type:         "line",
						},
					},
				},
				domain.GraphConfig{
					ID:     "jvm_thread",
					Name:   "JVM Thread",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "count",
					ReturnSet:  "EXACT",
					Type:       "line",
					Tags:       map[string][]string{"controlplane_service_id": []string{"isvc-elasticsearch-logstash"}},
					Units:      "Count",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "jvm_thread",
							MetricSource: "isvcs.jvm.threads",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "JVM Thread Count",
							Metric:       "isvcs.jvm.threads.count",
							Name:         "JVM Thread Count",
							Rate:         false,
							Type:         "line",
						},
					},
				},
			},
		},
	}
	ElasticsearchServicedISVC = s.Service{
		Name:            "Elastic Search - Serviced",
		ID:              "isvc-elasticsearch-serviced",
		Startup:         "/opt/elasticsearch-2.3.3/bin/elasticsearch",
		Description:     "Internal Elastic Search - Serviced",
		ParentServiceID: "isvc-internalservices",
		DesiredState:    1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "metrics",
					Name:        "Metrics",
					Description: "Metrics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "docker.usageinkernelmode", Name: "CPU System"},
						domain.Metric{ID: "docker.usageinusermode", Name: "CPU User"},
						domain.Metric{ID: "cgroup.memory.totalrss", Name: "Total RSS Memory"},
					},
				},
				domain.MetricConfig{
					ID:          "isvcs.jvm.gc",
					Name:        "JVM Garbage Collection",
					Description: "JVM Garbage Collection Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "isvcs.jvm.gc.old.collection_time", Name: "JVM Garbage Collection Old Generation Time"},
						domain.Metric{ID: "isvcs.jvm.gc.old.collection_count", Name: "JVM Garbage Collection Old Generation Runs"},
						domain.Metric{ID: "isvcs.jvm.gc.young.collection_time", Name: "JVM Garbage Collection Young Generation Time"},
						domain.Metric{ID: "isvcs.jvm.gc.young.collection_count", Name: "JVM Garbage Collection Young Generation Runs"},
					},
				},
				domain.MetricConfig{
					ID:          "isvcs.jvm.threads",
					Name:        "JVM Thread",
					Description: "JVM Thread Type Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "isvcs.jvm.threads.count", Name: "JVM Thread Count"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					ID:     "cpuUsage",
					Name:   "CPU Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"elasticsearch-serviced"}},
					Units:      "Percent",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "system",
							MetricSource: "metrics",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (System)",
							Metric:       "docker.usageinkernelmode",
							Name:         "CPU (System)",
							Rate:         false,
							Type:         "area",
						},
						domain.DataPoint{
							ID:           "system",
							MetricSource: "metrics",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (User)",
							Metric:       "docker.usageinusermode",
							Name:         "CPU (User)",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "memoryUsage",
					Name:   "Memory Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "bytes",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"elasticsearch-serviced"}},
					Units:      "Bytes",
					Base:       1024,
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "rssmemory",
							MetricSource: "metrics",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "Memory Usage",
							Metric:       "cgroup.memory.totalrss",
							Name:         "Memory Usage",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "jvm_gc_runs",
					Name:   "JVM Garbage Collection Runs",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "count",
					ReturnSet:  "EXACT",
					Type:       "line",
					Tags:       map[string][]string{"controlplane_service_id": []string{"isvc-elasticsearch-serviced"}},
					Units:      "Count",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "jvm_gc_runs",
							MetricSource: "isvcs.jvm.gc",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "JVM GC Old Generation Runs",
							Metric:       "isvcs.jvm.gc.old.collection_count",
							Name:         "JVM GC Old Generation Runs",
							Rate:         false,
							Type:         "line",
						},
						domain.DataPoint{
							ID:           "jvm_gc_runs",
							MetricSource: "isvcs.jvm.gc",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "JVM GC Young Generation Runs",
							Metric:       "isvcs.jvm.gc.young.collection_count",
							Name:         "JVM GC Young Generation Runs",
							Rate:         false,
							Type:         "line",
						},
					},
				},
				domain.GraphConfig{
					ID:     "jvm_gc_time",
					Name:   "JVM Garbage Collection Time",
					Footer: false,
					Format: "%.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "seconds",
					ReturnSet:  "EXACT",
					Type:       "line",
					Tags:       map[string][]string{"controlplane_service_id": []string{"isvc-elasticsearch-serviced"}},
					Units:      "Miliseconds",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "jvm_gc_time",
							MetricSource: "isvcs.jvm.gc",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%S",
							Legend:       "JVM GC Old Generation Time",
							Metric:       "isvcs.jvm.gc.old.collection_time",
							Name:         "JVM GC Old Generation Time",
							Rate:         false,
							Type:         "line",
						},
						domain.DataPoint{
							ID:           "jvm_gc_time",
							MetricSource: "isvcs.jvm.gc",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%S",
							Legend:       "JVM GC Young Generation Time",
							Metric:       "isvcs.jvm.gc.young.collection_time",
							Name:         "JVM GC Young Generation Time",
							Rate:         false,
							Type:         "line",
						},
					},
				},
				domain.GraphConfig{
					ID:     "jvm_thread",
					Name:   "JVM Thread",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "count",
					ReturnSet:  "EXACT",
					Type:       "line",
					Tags:       map[string][]string{"controlplane_service_id": []string{"isvc-elasticsearch-serviced"}},
					Units:      "Count",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "jvm_thread",
							MetricSource: "isvcs.jvm.threads",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "JVM Thread Count",
							Metric:       "isvcs.jvm.threads.count",
							Name:         "JVM Thread Count",
							Rate:         false,
							Type:         "line",
						},
					},
				},
			},
		},
	}
	ZookeeperIRS = dao.RunningService{
		Name:         "ZooKeeper",
		Description:  "Internal ZooKeeper",
		ID:           "isvc-zookeeper",
		ServiceID:    "isvc-zookeeper",
		DesiredState: 1,
		StartedAt:    time.Now(),
	}
	ZookeeperISVC = s.Service{
		Name:            "Zookeeper",
		ID:              "isvc-zookeeper",
		Startup:         "/opt/zookeeper-3.4.10/bin/zkServer.sh start-foreground",
		Description:     "Internal ZooKeeper",
		ParentServiceID: "isvc-internalservices",
		DesiredState:    1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "docker.usageinkernelmode", Name: "CPU System"},
						domain.Metric{ID: "docker.usageinusermode", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "cgroup.memory.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					ID:     "cpuUsage",
					Name:   "CPU Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"zookeeper"}},
					Units:      "Percent",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (System)",
							Metric:       "docker.usageinkernelmode",
							Name:         "CPU (System)",
							Rate:         false,
							Type:         "area",
						},
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (User)",
							Metric:       "docker.usageinusermode",
							Name:         "CPU (User)",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "memoryUsage",
					Name:   "Memory Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "bytes",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"zookeeper"}},
					Units:      "Bytes",
					Base:       1024,
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "rssmemory",
							MetricSource: "memory",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "Memory Usage",
							Metric:       "cgroup.memory.totalrss",
							Name:         "Memory Usage",
							Rate:         false,
							Type:         "area",
						},
					},
				},
			},
		},
	}
	LogstashIRS = dao.RunningService{
		Name:         "Logstash",
		Description:  "Internal Logstash",
		ID:           "isvc-logstash",
		ServiceID:    "isvc-logstash",
		DesiredState: 1,
		StartedAt:    time.Now(),
	}
	LogstashISVC = s.Service{
		Name:            "Logstash",
		ID:              "isvc-logstash",
		Startup:         "/opt/logstash-1.4.2/bin/logstash agent -f /usr/local/serviced/resources/logstash/logstash.conf",
		Description:     "Internal Logstash",
		ParentServiceID: "isvc-internalservices",
		DesiredState:    1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "docker.usageinkernelmode", Name: "CPU System"},
						domain.Metric{ID: "docker.usageinusermode", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "cgroup.memory.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					ID:     "cpuUsage",
					Name:   "CPU Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"logstash"}},
					Units:      "Percent",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (System)",
							Metric:       "docker.usageinkernelmode",
							Name:         "CPU (System)",
							Rate:         false,
							Type:         "area",
						},
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (User)",
							Metric:       "docker.usageinusermode",
							Name:         "CPU (User)",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "memoryUsage",
					Name:   "Memory Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "bytes",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"logstash"}},
					Units:      "Bytes",
					Base:       1024,
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "rssmemory",
							MetricSource: "memory",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "Memory Usage",
							Metric:       "cgroup.memory.totalrss",
							Name:         "Memory Usage",
							Rate:         false,
							Type:         "area",
						},
					},
				},
			},
		},
	}
	OpentsdbIRS = dao.RunningService{
		Name:         "OpenTSDB",
		Description:  "Internal Open TSDB",
		ID:           "isvc-opentsdb",
		ServiceID:    "isvc-opentsdb",
		DesiredState: 1,
		StartedAt:    time.Now(),
	}
	OpentsdbISVC = s.Service{
		Name:            "OpenTSDB",
		ID:              "isvc-opentsdb",
		Startup:         "cd /opt/zenoss && exec supervisord -n -c /opt/zenoss/etc/supervisor.conf",
		Description:     "Internal Open TSDB",
		ParentServiceID: "isvc-internalservices",
		DesiredState:    1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "docker.usageinkernelmode", Name: "CPU System"},
						domain.Metric{ID: "docker.usageinusermode", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "cgroup.memory.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					ID:     "cpuUsage",
					Name:   "CPU Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"opentsdb"}},
					Units:      "Percent",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (System)",
							Metric:       "docker.usageinkernelmode",
							Name:         "CPU (System)",
							Rate:         false,
							Type:         "area",
						},
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (User)",
							Metric:       "docker.usageinusermode",
							Name:         "CPU (User)",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "memoryUsage",
					Name:   "Memory Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "bytes",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"opentsdb"}},
					Units:      "Bytes",
					Base:       1024,
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "rssmemory",
							MetricSource: "memory",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "Memory Usage",
							Metric:       "cgroup.memory.totalrss",
							Name:         "Memory Usage",
							Rate:         false,
							Type:         "area",
						},
					},
				},
			},
		},
	}
	DockerRegistryIRS = dao.RunningService{
		Name:         "Docker Registry",
		Description:  "Internal Docker Registry",
		ID:           "isvc-docker-registry",
		ServiceID:    "isvc-docker-registry",
		DesiredState: 1,
		StartedAt:    time.Now(),
	}
	DockerRegistryISVC = s.Service{
		Name:            "Docker Registry",
		ID:              "isvc-docker-registry",
		Startup:         "DOCKER_REGISTRY_CONFIG=/docker-registry/config/config_sample.yml SETTINGS_FLAVOR=serviced docker-registry",
		Description:     "Internal Docker Registry",
		ParentServiceID: "isvc-internalservices",
		DesiredState:    1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "docker.usageinkernelmode", Name: "CPU System"},
						domain.Metric{ID: "docker.usageinusermode", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "cgroup.memory.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					ID:     "cpuUsage",
					Name:   "CPU Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"docker-registry"}},
					Units:      "Percent",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (System)",
							Metric:       "docker.usageinkernelmode",
							Name:         "CPU (System)",
							Rate:         false,
							Type:         "area",
						},
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (User)",
							Metric:       "docker.usageinusermode",
							Name:         "CPU (User)",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "memoryUsage",
					Name:   "Memory Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "bytes",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"docker-registry"}},
					Units:      "Bytes",
					Base:       1024,
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "rssmemory",
							MetricSource: "memory",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "Memory Usage",
							Metric:       "cgroup.memory.totalrss",
							Name:         "Memory Usage",
							Rate:         false,
							Type:         "area",
						},
					},
				},
			},
		},
	}
	KibanaIRS = dao.RunningService{
		Name:         "Kibana",
		Description:  "Internal Kibana",
		ID:           "isvc-kibana",
		ServiceID:    "isvc-kibana",
		DesiredState: 1,
		StartedAt:    time.Now(),
	}
	KibanaISVC = s.Service{
		Name:            "Kibana",
		ID:              "isvc-kibana",
		Startup:         "/opt/kibana-4.5.2/bin/kibana",
		Description:     "Internal Kibana",
		ParentServiceID: "isvc-internalservices",
		DesiredState:    1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "docker.usageinkernelmode", Name: "CPU System"},
						domain.Metric{ID: "docker.usageinusermode", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "cgroup.memory.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					ID:     "cpuUsage",
					Name:   "CPU Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"kibana"}},
					Units:      "Percent",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (System)",
							Metric:       "docker.usageinkernelmode",
							Name:         "CPU (System)",
							Rate:         false,
							Type:         "area",
						},
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (User)",
							Metric:       "docker.usageinusermode",
							Name:         "CPU (User)",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "memoryUsage",
					Name:   "Memory Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "bytes",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"kibana"}},
					Units:      "Bytes",
					Base:       1024,
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "rssmemory",
							MetricSource: "memory",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "Memory Usage",
							Metric:       "cgroup.memory.totalrss",
							Name:         "Memory Usage",
							Rate:         false,
							Type:         "area",
						},
					},
				},
			},
		},
	}
	ApiKeyProxyIRS = dao.RunningService{
		Name:         "API Key Proxy",
		Description:  "Internal API Key Proxy",
		ID:           "isvc-api-key-proxy",
		ServiceID:    "isvc-api-key-proxy",
		DesiredState: 1,
		StartedAt:    time.Now(),
	}
	ApiKeyProxyISVC = s.Service{
		Name:            "API Key Proxy",
		ID:              "isvc-api-key-proxy",
		Startup:         "KEYPROXY_ZPROXY_LOCATION=https://localhost:443 KEYPROXY_PROXY_LOCATION_USES_TLS=true",
		Description:     "Internal API key proxy",
		ParentServiceID: "isvc-internalservices",
		DesiredState:    1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MonitoringProfile: domain.MonitorProfile{
			MetricConfigs: []domain.MetricConfig{
				domain.MetricConfig{
					ID:          "cpu",
					Name:        "CPU Usage",
					Description: "CPU Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "docker.usageinkernelmode", Name: "CPU System"},
						domain.Metric{ID: "docker.usageinusermode", Name: "CPU User"},
					},
				},
				domain.MetricConfig{
					ID:          "memory",
					Name:        "Memory Usage",
					Description: "Memory Usage Statistics",
					Metrics: []domain.Metric{
						domain.Metric{ID: "cgroup.memory.totalrss", Name: "Total RSS Memory"},
					},
				},
			},
			GraphConfigs: []domain.GraphConfig{
				domain.GraphConfig{
					ID:     "cpuUsage",
					Name:   "CPU Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "% Used",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"api-key-proxy"}},
					Units:      "Percent",
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (System)",
							Metric:       "docker.usageinkernelmode",
							Name:         "CPU (System)",
							Rate:         false,
							Type:         "area",
						},
						domain.DataPoint{
							ID:           "system",
							MetricSource: "cpu",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "CPU (User)",
							Metric:       "docker.usageinusermode",
							Name:         "CPU (User)",
							Rate:         false,
							Type:         "area",
						},
					},
				},
				domain.GraphConfig{
					ID:     "memoryUsage",
					Name:   "Memory Usage",
					Footer: false,
					Format: "%4.2f",
					MaxY:   nil,
					MinY:   &zero,
					Range: &domain.GraphConfigRange{
						End:   "0s-ago",
						Start: "1h-ago",
					},
					YAxisLabel: "bytes",
					ReturnSet:  "EXACT",
					Type:       "area",
					Tags:       map[string][]string{"isvcname": []string{"api-key-proxy"}},
					Units:      "Bytes",
					Base:       1024,
					DataPoints: []domain.DataPoint{
						domain.DataPoint{
							ID:           "rssmemory",
							MetricSource: "memory",
							Aggregator:   "avg",
							Fill:         false,
							Format:       "%4.2f",
							Legend:       "Memory Usage",
							Metric:       "cgroup.memory.totalrss",
							Name:         "Memory Usage",
							Rate:         false,
							Type:         "area",
						},
					},
				},
			},
		},
	}


	ISVCSMap = map[string]*s.Service{
		"isvc-internalservices":       &InternalServicesISVC,
		"isvc-elasticsearch-logstash": &ElasticsearchLogStashISVC,
		"isvc-elasticsearch-serviced": &ElasticsearchServicedISVC,
		"isvc-zookeeper":              &ZookeeperISVC,
		"isvc-logstash":               &LogstashISVC,
		"isvc-opentsdb":               &OpentsdbISVC,
		"isvc-docker-registry":        &DockerRegistryISVC,
		"isvc-kibana":                 &KibanaISVC,
		"isvc-api-key-proxy":          &ApiKeyProxyISVC,
	}

	IRSMap = map[string]*dao.RunningService{
		"isvc-internalservices":       &InternalServicesIRS,
		"isvc-elasticsearch-logstash": &ElasticsearchLogStashIRS,
		"isvc-elasticsearch-serviced": &ElasticsearchServicedIRS,
		"isvc-zookeeper":              &ZookeeperIRS,
		"isvc-logstash":               &LogstashIRS,
		"isvc-opentsdb":               &OpentsdbIRS,
		"isvc-docker-registry":        &DockerRegistryIRS,
		"isvc-kibana":                 &KibanaIRS,
		"isvc-api-key-proxy":          &ApiKeyProxyIRS,
	}
}

func InitAllIsvcs(bigtable bool) {
	initZK()
	initOTSDB(bigtable)
	initLogstash()
	initElasticSearch()
	initDockerRegistry()
	initKibana()
	initApiKeyProxy()
}
