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

package service

import (
	"errors"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
)

func (s *storeImpl) GetServiceHealth(ctx datastore.Context, svcId string) (*ServiceHealth, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.GetServiceHealth"))

	id := strings.TrimSpace(svcId)
	if id == "" {
		return nil, errors.New("empty service id not allowed")
	}

	searchRequest := newServiceHealthElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"ids": map[string]interface{}{
				"values": []string{id},
			},
		},
		"fields": serviceHealthFields,
		"size":   1,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	if results.HasNext() {
		var health ServiceHealth
		err = results.Next(&health)
		if err != nil {
			return nil, err
		}
		s.fillHealthVolatileInfo(&health)
		return &health, nil
	}

	key := datastore.NewKey(kind, svcId)
	return nil, datastore.ErrNoSuchEntity{Key: key}
}

func (s *storeImpl) GetAllServiceHealth(ctx datastore.Context) ([]ServiceHealth, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetServiceHealth"))
	searchRequest := newServiceHealthElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"query_string": map[string]string{
				"query": "_exists_:ID",
			},
		},
		"fields": serviceHealthFields,
		"size":   serviceHealthLimit,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	health := []ServiceHealth{}
	for results.HasNext() {
		var sh ServiceHealth
		err := results.Next(&sh)
		if err != nil {
			return nil, err
		}
		s.fillHealthVolatileInfo(&sh)
		health = append(health, sh)
	}

	return health, nil
}

func (s *storeImpl) fillHealthVolatileInfo(sh *ServiceHealth) {
	cacheEntry, ok := s.getVolatileInfo(sh.ID) // Uses Mutex RLock
	if ok {
		sh.DesiredState = cacheEntry.DesiredState
	} else {
		// If there's no ZK data, make sure the service is stopped.
		sh.DesiredState = int(SVCStop)
	}
}

func newServiceHealthElasticRequest(query interface{}) elastic.ElasticSearchRequest {
	return elastic.ElasticSearchRequest{
		Pretty: false,
		Index:  "controlplane",
		Type:   "service",
		Scroll: "",
		Scan:   0,
		Query:  query,
	}
}

var serviceHealthLimit = 50000

var serviceHealthFields = []string{
	"ID",
	"Name",
	"PoolID",
	"Instances",
	"DesiredState",
	"HealthChecks",
	"EmergencyShutdown",
}
