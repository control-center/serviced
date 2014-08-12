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

package service

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/glog"

	"errors"
	"fmt"
	"strings"
)

//NewStore creates a Service  store
func NewStore() *Store {
	return &Store{}
}

//Store type for interacting with Service persistent storage
type Store struct {
	ds datastore.DataStore
}

// Put adds or updates a Service
func (s *Store) Put(ctx datastore.Context, svc *Service) error {
	//No need to store ConfigFiles
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)

	return s.ds.Put(ctx, Key(svc.ID), svc)
}

// fillBuiltinMetrics adds internal metrics to the monitoring profile
func fillBuiltinMetrics(svc *Service) {
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
	"cgroup.cpuacct.system", "cgroup.cpuacct.user", "cgroup.memory.pgmajfault",
}
var internalGuageStats = []string{
	"cgroup.memory.totalrss", "cgroup.memory.cache", "net.rx_bytes", "net.rx_compressed",
}

func removeInternalGraphConfigs(svc *Service) {
	var configs []domain.GraphConfig
	for _, config := range svc.MonitoringProfile.GraphConfigs {
		if config.BuiltIn {
			continue
		}
		configs = append(configs, config)
	}
	svc.MonitoringProfile.GraphConfigs = configs
}

func addInternalGraphConfigs(svc *Service) {

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
			Format:      "%d",
			ReturnSet:   "EXACT",
			Type:        "area",
			Tags:        tags,
			YAxisLabel:  "% CPU Used",
			Description: "% CPU Used Over Last Hour",
			MinY:        &zero,
			Range:       &tRange,
			DataPoints: []domain.DataPoint{
				domain.DataPoint{
					Aggregator:   "avg",
					Format:       "%d",
					Legend:       "System",
					Metric:       "cgroup.cpuacct.system",
					MetricSource: "metrics",
					ID:           "cgroup.cpuacct.system",
					Name:         "System",
					Rate:         true,
					Type:         "area",
				},
				domain.DataPoint{
					Aggregator:   "avg",
					Format:       "%d",
					Legend:       "User",
					Metric:       "cgroup.cpuacct.user",
					MetricSource: "metrics",
					ID:           "cgroup.cpuacct.user",
					Name:         "User",
					Rate:         true,
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
			Format:      "%6.2f",
			ReturnSet:   "EXACT",
			Type:        "area",
			Tags:        tags,
			YAxisLabel:  "GB",
			Description: "GB Memory Used Over Last Hour",
			MinY:        &zero,
			Range:       &tRange,
			DataPoints: []domain.DataPoint{
				domain.DataPoint{
					Aggregator:   "avg",
					Expression:   "rpn:1024,/,1024,/,1024,/",
					Fill:         true,
					Format:       "%6.2f",
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
					Expression:   "rpn:1024,/,1024,/,1024,/",
					Fill:         true,
					Format:       "%6.2f",
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

	// network usage graph
	svc.MonitoringProfile.GraphConfigs = append(
		svc.MonitoringProfile.GraphConfigs,
		domain.GraphConfig{
			ID:          "internalNetworkUsage",
			Name:        "Network Usage",
			BuiltIn:     true,
			Format:      "%6.2f",
			ReturnSet:   "EXACT",
			Type:        "area",
			Tags:        tags,
			YAxisLabel:  "kbps",
			Range:       &tRange,
			Description: "kbps over last hour",
			DataPoints: []domain.DataPoint{
				domain.DataPoint{
					Aggregator:   "avg",
					Expression:   "rpn:8,/,1024,/",
					Fill:         true,
					Format:       "%6.2f",
					Legend:       "TX",
					Metric:       "net.tx_bytes",
					MetricSource: "metrics",
					ID:           "net.tx_bytes",
					Name:         "TX kbps",
					Rate:         true,
					Type:         "area",
				},
				domain.DataPoint{
					Aggregator:   "avg",
					Expression:   "rpn:8,/,1024,/",
					Fill:         true,
					Format:       "%6.2f",
					Legend:       "RX",
					Metric:       "net.rx_bytes",
					MetricSource: "metrics",
					ID:           "net.rx_bytes",
					Name:         "RX kbps",
					Rate:         true,
					Type:         "area",
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
	for _, metricName := range internalGuageStats {
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

func findInternalMetricConfig(svc *Service) (index int, found bool) {
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

// Get a Service by id. Return ErrNoSuchEntity if not found
func (s *Store) Get(ctx datastore.Context, id string) (*Service, error) {
	svc := &Service{}
	if err := s.ds.Get(ctx, Key(id), svc); err != nil {
		return nil, err
	}

	//Copy original config files
	fillConfig(svc)

	//Add builtin metrics
	fillBuiltinMetrics(svc)
	return svc, nil
}

// Delete removes the a Service if it exists
func (s *Store) Delete(ctx datastore.Context, id string) error {
	return s.ds.Delete(ctx, Key(id))
}

//GetServices returns all services
func (s *Store) GetServices(ctx datastore.Context) ([]*Service, error) {
	return query(ctx, "_exists_:ID")
}

//GetTaggedServices returns services with the given tags
func (s *Store) GetTaggedServices(ctx datastore.Context, tags ...string) ([]*Service, error) {
	if len(tags) == 0 {
		return nil, errors.New("empty tags not allowed")
	}
	qs := strings.Join(tags, " AND ")
	return query(ctx, qs)
}

//GetServicesByPool returns services with the given pool id
func (s *Store) GetServicesByPool(ctx datastore.Context, poolID string) ([]*Service, error) {
	id := strings.TrimSpace(poolID)
	if id == "" {
		return nil, errors.New("empty poolID not allowed")
	}
	queryString := fmt.Sprintf("PoolID:%s", id)
	return query(ctx, queryString)
}

//GetServicesByDeployment returns services with the given deployment id
func (s *Store) GetServicesByDeployment(ctx datastore.Context, deploymentID string) ([]*Service, error) {
	id := strings.TrimSpace(deploymentID)
	if id == "" {
		return nil, errors.New("empty deploymentID not allowed")
	}
	queryString := fmt.Sprintf("DeploymentID:%s", id)
	return query(ctx, queryString)
}

//GetChildServices returns services that are children of the given parent service id
func (s *Store) GetChildServices(ctx datastore.Context, parentID string) ([]*Service, error) {
	id := strings.TrimSpace(parentID)
	if id == "" {
		return nil, errors.New("empty parent service id not allowed")
	}

	queryString := fmt.Sprintf("ParentServiceID:%s", parentID)
	return query(ctx, queryString)
}

func query(ctx datastore.Context, query string) ([]*Service, error) {
	q := datastore.NewQuery(ctx)
	elasticQuery := search.Query().Search(query)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(elasticQuery)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func fillConfig(svc *Service) {
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)
	for key, val := range svc.OriginalConfigs {
		svc.ConfigFiles[key] = val
	}
}

func convert(results datastore.Results) ([]*Service, error) {
	svcs := make([]*Service, results.Len())
	for idx := range svcs {
		var svc Service
		err := results.Get(idx, &svc)
		if err != nil {
			return nil, err
		}
		fillConfig(&svc)
		fillBuiltinMetrics(&svc)
		svcs[idx] = &svc
	}
	return svcs, nil
}

//Key creates a Key suitable for getting, putting and deleting Services
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

//confFileKey creates a Key suitable for getting, putting and deleting svcConfigFile
func confFileKey(id string) datastore.Key {
	return datastore.NewKey(confKind, id)
}

var (
	kind     = "service"
	confKind = "serviceconfig"
)
