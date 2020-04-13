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

package addressassignment

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
)

//NewStore creates a AddressAssignmentStore store
func NewStore() *Store {
	return &Store{}
}

//Store type for interacting with AddressAssignment persistent storage
type Store struct {
	datastore.DataStore
}

func (s *Store) GetAllAddressAssignments(ctx datastore.Context) ([]AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.GetAllAddressAssignments"))
	q := datastore.NewQuery(ctx)
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]string{"type": kind},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func (s *Store) GetServiceAddressAssignments(ctx datastore.Context, serviceID string) ([]AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.GetServiceAddressAssignments"))
	q := datastore.NewQuery(ctx)
	// Build the request body.
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"ServiceID": serviceID}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}

	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func (s *Store) GetServiceAddressAssignmentsByPort(ctx datastore.Context, poolID string, port uint16) ([]AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.GetServiceAddressAssignmentsByPort"))
	if poolID = strings.TrimSpace(poolID); poolID == "" {
		return nil, fmt.Errorf("poolID cannot be empty")
	} else if port == 0 {
		return nil, fmt.Errorf("port must be greater than 0")
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"PoolID": poolID}},
					{"term": map[string]string{"Port": strconv.FormatUint(uint64(port), 10)}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}

	if results, err := datastore.NewQuery(ctx).Execute(search); err != nil {
		return nil, err
	} else {
		return convert(results)
	}
}

func (s *Store) FindAssignmentByServiceEndpoint(ctx datastore.Context, serviceID, endpointName string) (*AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.FindAssignmentByServiceEndpoint"))
	if serviceID = strings.TrimSpace(serviceID); serviceID == "" {
		return nil, fmt.Errorf("service ID cannot be empty")
	} else if endpointName = strings.TrimSpace(endpointName); endpointName == "" {
		return nil, fmt.Errorf("endpoint name cannot be empty")
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"ServiceID": serviceID}},
					{"term": map[string]string{"EndpointName": endpointName}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}

	if results, err := datastore.NewQuery(ctx).Execute(search); err != nil {
		return nil, err
	} else if output, err := convert(results); err != nil {
		return nil, err
	} else if len(output) == 0 {
		return nil, nil
	} else {
		return &output[0], nil
	}
}

func (s *Store) FindAssignmentByHostPort(ctx datastore.Context, poolID string, ipAddr string, port uint16) (*AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.FindAssignmentByHostPort"))
	if poolID = strings.TrimSpace(poolID); poolID == "" {
		return nil, fmt.Errorf("poolID cannot be empty")
	} else if ipAddr = strings.TrimSpace(ipAddr); ipAddr == "" {
		return nil, fmt.Errorf("hostIP cannot be empty")
	} else if port == 0 {
		return nil, fmt.Errorf("port must be greater than 0")
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"PoolID": poolID}},
					{"term": map[string]string{"IPAddr": ipAddr}},
					{"term": map[string]string{"Port": strconv.FormatUint(uint64(port), 10)}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}
	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}

	if results, err := datastore.NewQuery(ctx).Execute(search); err != nil {
		return nil, err
	} else if output, err := convert(results); err != nil {
		return nil, err
	} else if len(output) == 0 {
		return nil, nil
	} else {
		return &output[0], nil
	}
}

//Key creates a Key suitable for getting, putting and deleting AddressAssignment
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

func convert(results datastore.Results) ([]AddressAssignment, error) {
	assignments := make([]AddressAssignment, results.Len())
	for idx := range assignments {
		var aa AddressAssignment
		err := results.Get(idx, &aa)
		if err != nil {
			return nil, err
		}

		assignments[idx] = aa
	}
	return assignments, nil
}

var kind = "addressassignment"
