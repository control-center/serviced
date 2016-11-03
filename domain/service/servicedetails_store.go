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

func (s *storeImpl) GetAllPublicEndpoints(ctx datastore.Context) ([]PublicEndpoint, error) {
	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					map[string]interface{}{
						"regexp": map[string]interface{}{
							"Endpoints.VHostList.Name": ".+",
						},
					},
					map[string]interface{}{
						"regexp": map[string]interface{}{
							"Endpoints.PortList.PortAddr": ".+",
						},
					},
				},
			},
		},
		"fields": publicEndpointFields,
		"size":   serviceDetailsLimit,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	peps := []PublicEndpoint{}
	for results.HasNext() {
		var result EndpointQueryResult
		err := results.Next(&result)
		if err != nil {
			return nil, err
		}
		endpoints := createPublicEndpoints(result)
		peps = append(peps, endpoints...)
	}

	return peps, nil
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

func createPublicEndpoints(result EndpointQueryResult) []PublicEndpoint {
	pubs := []PublicEndpoint{}

	for _, ep := range result.Endpoints {
		for _, vhost := range ep.VHostList {
			pubs = append(pubs, PublicEndpoint{
				ServiceID:   result.ID,
				ServiceName: result.Name,
				Application: ep.Application,
				Protocol:    "https",
				VHostName:   vhost.Name,
				Enabled:     vhost.Enabled,
			})
		}

		for _, port := range ep.PortList {
			pub := PublicEndpoint{
				ServiceID:   result.ID,
				ServiceName: result.Name,
				Application: ep.Application,
				PortAddress: port.PortAddr,
				Enabled:     port.Enabled,
			}

			if strings.HasPrefix(port.Protocol, "http") {
				pub.Protocol = port.Protocol
			} else if port.UseTLS {
				pub.Protocol = "Other, secure (TLS)"
			} else {
				pub.Protocol = "Other, non-secure"
			}

			pubs = append(pubs, pub)
		}
	}

	return pubs
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

var publicEndpointFields = []string{
	"ID",
	"Name",
	"Endpoints",
}

// EndpointQueryResult used for unmarshalling query results
type EndpointQueryResult struct {
	ID        string
	Name      string
	Endpoints []ServiceEndpoint
	datastore.VersionedEntity
}

// ValidEntity for EndpointQueryResult entity
func (d *EndpointQueryResult) ValidEntity() error {
	return nil
}
