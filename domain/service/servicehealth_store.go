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
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/elastic/go-elasticsearch/v7/esapi"
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
				"values": []string{elastic.BuildID(id, kind)},
			},
		},
	}, 1, serviceHealthFields)

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
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"exists": map[string]string{"field": "ID"}},
					{"term": map[string]string{"type": "service"}},
				},
			},
		},
	}, serviceHealthLimit, serviceHealthFields)

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
		sh.CurrentState = cacheEntry.CurrentState
	} else {
		// If there's no ZK data, make sure the service is stopped.
		sh.DesiredState = int(SVCStop)
		sh.CurrentState = string(SVCCSUnknown)
	}
}

func newServiceHealthElasticRequest(query interface{}, size int, fields []string) esapi.SearchRequest {
	// Build the request body.
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		plog.Fatalf("Error encoding query: %s", err)
	}
	version := true
	seqNoPrimaryTerm := true
	return esapi.SearchRequest{
		Pretty:           false,
		Index:            []string{"controlplane"},
		Body:             &buf,
		Size:             &size,
		Version:          &version,
		SeqNoPrimaryTerm: &seqNoPrimaryTerm,
		SourceIncludes:   fields,
	}
}

var serviceHealthLimit = 10000

var serviceHealthFields = []string{
	"ID",
	"Name",
	"PoolID",
	"Instances",
	"DesiredState",
	"HealthChecks",
	"EmergencyShutdown",
	"RAMCommitment",
}
