// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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

	//Remove built in metrics
	removeBuiltinMetrics(svc)

	return s.ds.Put(ctx, Key(svc.ID), svc)
}

// removeBuiltinMetrics removes internal metrics from the monitoring profile
func removeBuiltinMetrics(svc *Service) {
}

/*
        MetricConfigs    []MetricConfig    //metrics for domain object
        GraphConfigs     []GraphConfig     //graphs for a domain object
        ThresholdConfigs []ThresholdConfig //thresholds for a domain object


// Metric defines the meta-data for a single metric
type Metric struct {
        ID          string //id is a unique idenitifier for the metric
        Name        string //name is a canonical name for the metric
        Description string //description of this metric
        Counter     bool   // Counter is true if this metric is a constantly incrementing measure
        CounterMax  *int64 `json:"CounterMax,omitempty"`
        ResetValue  int64  // If metric is a counter, ResetValue is the maximum counter value before a rollover occurs
        Unit        string // Unit of measure for metric
        BuiltIn     bool   // is this metric supplied by the serviced runtime?
}


*/

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
	findInternalMetricConfig(svc)
}

func findInternalMetricConfig(svc *Service) (index int, found bool) {
	// find the metric config
	for i := range svc.MonitoringProfile.MetricConfigs {
		if svc.MonitoringProfile.MetricConfigs[i].ID == "metrics" {
			return i, true
		}
	}
	return -1, false
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
