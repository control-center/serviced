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
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
)

// Search executes a query and resturns the result.
func (s *store) Search(ctx datastore.Context, query Query) ([]ServiceDetails, error) {
	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"query_string": map[string]interface{}{
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

	var r *regexp.Regexp
	if query.Name != "" {
		r, _ = regexp.Compile(query.Name)
	}

	var since time.Time
	if query.Since > 0 {
		since = time.Now().Add(-query.Since)
	}

	details := []ServiceDetails{}
	for results.HasNext() {
		var d ServiceDetails
		err := results.Next(&d)
		if err != nil {
			return nil, err
		}

		parentMap[d.ParentServiceID] = struct{}{}

		if r != nil && !r.MatchString(d.Name) {
			continue
		}

		if query.Since > 0 && d.UpdatedAt.Before(since) {
			continue
		}

		if query.Tenants && d.ParentServiceID != "" {
			continue
		}

		if len(query.Tags) > 0 {
			tagsMatch := true
			for _, t := range query.Tags {
				if !contains(d.Tags, t) {
					tagsMatch = false
					break
				}
			}
			if !tagsMatch {
				continue
			}
		}

		fillDetailsVolatileInfo(&d)
		details = append(details, d)
	}

	for i, d := range details {
		_, details[i].HasChildren = parentMap[d.ID]
	}

	return details, nil
}

func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

// GetServiceDetails returns service details for an id
func (s *store) GetServiceDetails(ctx datastore.Context, serviceID string) (*ServiceDetails, error) {
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

		if hasChildren, err := hasChildren(ctx, details.ID); err == nil {
			details.HasChildren = hasChildren
		} else {
			return nil, err
		}

		fillDetailsVolatileInfo(&details)
		return &details, nil
	}

	key := datastore.NewKey(kind, serviceID)
	return nil, datastore.ErrNoSuchEntity{Key: key}
}

// GetServiceDetailsByParentID returns service details given parent service id
func (s *store) GetServiceDetailsByParentID(ctx datastore.Context, parentID string, since time.Duration) ([]ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetServiceDetailsByParentID"))
	query := map[string]interface{}{}
	termQuery := map[string]string{
		"ParentServiceID": parentID,
	}
	if since > 0 {
		t0 := time.Now().Add(-since)
		query["bool"] = map[string]interface{}{
			"must": []map[string]interface{}{
				map[string]interface{}{
					"term": termQuery,
				},
				map[string]interface{}{
					"range": map[string]interface{}{
						"UpdatedAt": map[string]string{
							"gte": t0.Format(time.RFC3339),
						},
					},
				},
			},
		}
	} else {
		query["term"] = termQuery
	}

	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query":  query,
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

		if hasChildren, err := hasChildren(ctx, d.ID); err == nil {
			d.HasChildren = hasChildren
		} else {
			return nil, err
		}

		fillDetailsVolatileInfo(&d)
		details = append(details, d)
	}

	return details, nil
}

// GetServiceDetailsByIDOrName returns the service details for any services
// whose serviceID matches the query exactly or whose names contain the query
// as a substring
func (s *store) GetServiceDetailsByIDOrName(ctx datastore.Context, query string, noprefix bool) ([]ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetServiceDetailsByIDOrName"))

	// Because we don't analyze any of our fields, we have to do this extremely
	// idiotic regular expression to handle service names with mixed case
	regex := make([]rune, len(query)*4)
	idx := 0
	for _, r := range []rune(query) {
		if unicode.IsLetter(r) {
			regex[idx] = '['
			regex[idx+1] = unicode.ToLower(r)
			regex[idx+2] = unicode.ToUpper(r)
			regex[idx+3] = ']'
			idx += 4
		} else if isRegexReservedChar(r) {
			// Escape all reserved characters
			regex[idx] = '\\'
			regex[idx+1] = r
			idx += 2
		} else {
			// anything else should be usable as is
			regex[idx] = r
			idx++
		}
	}

	newquery := fmt.Sprintf("%s", string(regex[:idx]))

	if noprefix {
		// Set query to "ends with" style
		newquery = fmt.Sprintf(".*%s", newquery)
	} else {
		// Set query to "contains" style
		newquery = fmt.Sprintf(".*%s.*", newquery)
	}

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
						"regexp": map[string]interface{}{
							"Name": newquery,
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

		if hasChildren, err := hasChildren(ctx, d.ID); err == nil {
			d.HasChildren = hasChildren
		} else {
			return nil, err
		}
		fillDetailsVolatileInfo(&d)
		details = append(details, d)
	}

	return details, nil
}

// GetAllPublicEndpoints returns all the public endpoints in the system
func (s *store) GetAllPublicEndpoints(ctx datastore.Context) ([]PublicEndpoint, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetAllPublicEndpoints"))
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
		"fields": serviceEndpointFields,
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

// GetAllExportedEndpoints returns all the exported endpoints in the system
func (s *store) GetAllExportedEndpoints(ctx datastore.Context) ([]ExportedEndpoint, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetAllExportedEndpoints"))
	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"Endpoints.Purpose": "export",
			},
		},
		"fields": exportedEndpointFields,
		"size":   serviceDetailsLimit,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	eps := []ExportedEndpoint{}
	for results.HasNext() {
		var result EndpointQueryResult
		err := results.Next(&result)
		if err != nil {
			return nil, err
		}
		endpoints := createExportedEndpoints(result)
		eps = append(eps, endpoints...)
	}

	return eps, nil
}

// GetAllIPAssignments returns an array of BaseIPAssignment objects
func (s *store) GetAllIPAssignments(ctx datastore.Context) ([]BaseIPAssignment, error) {
	// All services where Endpoints.AddressConfig.Port > 0 and Endpoints.Protocol != ""
	searchRequest := newServiceDetailsElasticRequest(map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					map[string]interface{}{
						"range": map[string]interface{}{
							"Endpoints.AddressConfig.Port": map[string]interface{}{
								"gt":  0,
								"lte": 65535, // largest valid port is 65535 (unsigned 16-bit int)
							},
						},
					},
					map[string]interface{}{
						"regexp": map[string]interface{}{
							"Endpoints.Protocol": ".+",
						},
					},
				},
			},
		},
		"fields": serviceEndpointFields,
		"size":   serviceDetailsLimit,
	})

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	ipAssignments := []BaseIPAssignment{}
	for results.HasNext() {
		var result EndpointQueryResult
		err := results.Next(&result)
		if err != nil {
			return nil, err
		}
		assignment := createIPAssignment(result)
		ipAssignments = append(ipAssignments, assignment...)
	}

	return ipAssignments, nil
}

func hasChildren(ctx datastore.Context, serviceID string) (bool, error) {
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

func fillDetailsVolatileInfo(d *ServiceDetails) {
	cacheEntry, ok := getVolatileInfo(d.ID) // Uses Mutex RLock
	if ok {
		d.DesiredState = cacheEntry.DesiredState
		d.CurrentState = cacheEntry.CurrentState
	} else {
		// If there's no ZK data, make sure the service is stopped.
		d.DesiredState = int(SVCStop)
		d.CurrentState = string(SVCCSUnknown)
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

func createExportedEndpoints(result EndpointQueryResult) []ExportedEndpoint {
	eps := []ExportedEndpoint{}

	for _, ep := range result.Endpoints {
		if ep.Purpose == "export" {
			eps = append(eps, ExportedEndpoint{
				ServiceID:   result.ID,
				ServiceName: result.Name,
				Application: ep.Application,
				Protocol:    ep.Protocol,
			})
		}
	}

	return eps
}
func createIPAssignment(result EndpointQueryResult) []BaseIPAssignment {
	ipAssignments := []BaseIPAssignment{}
	for _, ep := range result.Endpoints {
		if ep.AddressConfig.Port == 0 {
			continue
		}
		assignment := BaseIPAssignment{
			ServiceID:       result.ID,
			ParentServiceID: result.ParentServiceID,
			ServiceName:     result.Name,
			PoolID:          result.PoolID,

			Application:  ep.Application,
			EndpointName: ep.Name,
			Port:         ep.AddressConfig.Port,
		}
		ipAssignments = append(ipAssignments, assignment)

	}
	return ipAssignments
}

// isRegexReservedChar returns true if r is one of the reserved characters for Elasticsearch's
// regular expression syntax. See the following for more info:
// https://www.elastic.co/guide/en/elasticsearch/reference/2.0/query-dsl-regexp-query.html#regexp-syntax
func isRegexReservedChar(r rune) bool {
	const reservedList = ".?+*|{}[]()\"\\#@&<>~"
	if strings.IndexRune(reservedList, r) == -1 {
		return false
	}
	return true
}

func newServiceDetailsElasticRequest(query interface{}) elastic.SearchRequest {
	return elastic.SearchRequest{
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
	"RAMThreshold",
	"Startup",
	"DeploymentID",
	"Launch",
	"Tags",
	"EmergencyShutdown",
	"UpdatedAt",
	"CreatedAt",
	"Version",
}

var serviceEndpointFields = []string{
	"ID",
	"Name",
	"PoolID",
	"ParentServiceID",
	"Endpoints",
}

var exportedEndpointFields = []string{
	"ID",
	"Name",
	"Endpoints",
}

// EndpointQueryResult used for unmarshalling query results
type EndpointQueryResult struct {
	ID              string
	Name            string
	PoolID          string
	ParentServiceID string
	Endpoints       []ServiceEndpoint
	datastore.VersionedEntity
}

// ValidEntity for EndpointQueryResult entity
func (d *EndpointQueryResult) ValidEntity() error {
	return nil
}
