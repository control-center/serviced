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
	"fmt"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
)

// GetAllServiceDetails returns service details for an id
func (s *storeImpl) GetAllServiceDetails(ctx datastore.Context) ([]ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetAllServiceDetails"))
	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"query_string": map[string]string{
				"query": "_exists_:ID",
			},
		},
		"fields": serviceDetailsFields,
		"size":   serviceDetailsLimit,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	parentMap := make(map[string]struct{})

	details := []ServiceDetails{}
	for results.HasNext() {
		var d ServiceDetails
		err := results.Next(&d)
		if err != nil {
			return nil, err
		}

		parentMap[d.ParentServiceID] = struct{}{}
		s.fillDetailsVolatileInfo(&d)
		details = append(details, d)
	}

	for i, d := range details {
		_, details[i].HasChildren = parentMap[d.ID]
	}

	return details, nil
}

// GetServiceDetails returns service details for an id
func (s *storeImpl) GetServiceDetails(ctx datastore.Context, serviceID string) (*ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetServiceDetails"))
	id := strings.TrimSpace(serviceID)
	if id == "" {
		return nil, errors.New("empty service id not allowed")
	}

	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"ids": map[string]interface{}{
				"values": []string{id},
			},
		},
		"fields": serviceDetailsFields,
		"size":   1,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	if results.HasNext() {
		var details ServiceDetails
		err = results.Next(&details)
		if err != nil {
			return nil, err
		}

		if hasChildren, err := s.hasChildren(ctx, details.ID); err == nil {
			details.HasChildren = hasChildren
		} else {
			return nil, err
		}

		s.fillDetailsVolatileInfo(&details)
		return &details, nil
	}

	key := datastore.NewKey(kind, serviceID)
	return nil, datastore.ErrNoSuchEntity{Key: key}
}

// GetChildServiceDetailsByParentID returns service details given parent service id
func (s *storeImpl) GetServiceDetailsByParentID(ctx datastore.Context, parentID string) ([]ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetServiceDetailsByParentID"))
	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]string{"ParentServiceID": parentID},
		},
		"fields": serviceDetailsFields,
		"size":   serviceDetailsLimit,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	details := []ServiceDetails{}
	for results.HasNext() {
		var d ServiceDetails
		err := results.Next(&d)
		if err != nil {
			return nil, err
		}

		if hasChildren, err := s.hasChildren(ctx, d.ID); err == nil {
			d.HasChildren = hasChildren
		} else {
			return nil, err
		}

		s.fillDetailsVolatileInfo(&d)
		details = append(details, d)
	}

	return details, nil
}

// GetServiceDetailsByIDOrName returns the service details for any services
// whose serviceID matches the query exactly or whose names contain the query
// as a substring
func GetServiceDetailsByIDOrName(ctx datastore.Context, query string) ([]ServiceDetails, error) {
	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					map[string]interface{}{
						"ids": map[string]interface{}{
							"values": []string{query},
						},
					},
					map[string]interface{}{
						"wildcard": map[string]interface{}{
							"Name": fmt.Sprintf("*%s", query),
						},
					},
				},
			},
		},
		"fields": serviceDetailsFields,
		"size":   serviceDetailsLimit,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	details := []ServiceDetails{}
	for results.HasNext() {
		var d ServiceDetails
		err := results.Next(&d)
		if err != nil {
			return nil, err
		}

		if hasChildren, err := s.hasChildren(ctx, d.ID); err == nil {
			d.HasChildren = hasChildren
		} else {
			return nil, err
		}

		details = append(details, d)
	}

	return details, nil
}

func (s *storeImpl) hasChildren(ctx datastore.Context, serviceID string) (bool, error) {
	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]string{"ParentServiceID": serviceID},
		},
		"fields": []string{"ID"},
		"size":   1,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return false, err
	}

	return results.Len() > 0, nil

}

func (s *storeImpl) fillDetailsVolatileInfo(d *ServiceDetails) {
	cacheEntry, ok := s.getVolatileInfo(d.ID) // Uses Mutex RLock
	if ok {
		d.DesiredState = cacheEntry.DesiredState
        } else {
                // If there's no ZK data, make sure the service is stopped.
                d.DesiredState = int(SVCStop)
	}
}

func newServiceDetailsElasticRequest(query interface{}) elastic.ElasticSearchRequest {
	return elastic.ElasticSearchRequest{
		Pretty: false,
		Index:  "controlplane",
		Type:   "service",
		Scroll: "",
		Scan:   0,
		Query:  query,
	}
}

var serviceDetailsLimit = 50000

var serviceDetailsFields = []string{
	"ID",
	"Name",
	"Description",
	"PoolID",
	"ImageID",
	"ParentServiceID",
	"Instances",
	"InstanceLimits",
	"RAMCommitment",
	"Startup",
	"DeploymentID",
	"Launch",
}
