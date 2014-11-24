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
	"github.com/zenoss/elastigo/search"
)

//NewStore creates a AddressAssignmentStore store
func NewStore() *Store {
	return &Store{}
}

//Store type for interacting with AddressAssignment persistent storage
type Store struct {
	datastore.DataStore
}

func (s *Store) GetServiceAddressAssignments(ctx datastore.Context, serviceID string) ([]AddressAssignment, error) {
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("ServiceID", serviceID)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func (s *Store) GetServiceAddressAssignmentsByPort(ctx datastore.Context, port uint16) ([]AddressAssignment, error) {
	if port == 0 {
		return nil, fmt.Errorf("port must be greater than 0")
	}

	query := search.Query().Term("Port", strconv.FormatUint(uint64(port), 10))
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)

	if results, err := datastore.NewQuery(ctx).Execute(search); err != nil {
		return nil, err
	} else {
		return convert(results)
	}
}

func (s *Store) FindAssignmentByServiceEndpoint(ctx datastore.Context, serviceID, endpointName string) (*AddressAssignment, error) {
	if serviceID = strings.TrimSpace(serviceID); serviceID == "" {
		return nil, fmt.Errorf("service ID cannot be empty")
	} else if endpointName = strings.TrimSpace(endpointName); endpointName == "" {
		return nil, fmt.Errorf("endpoint name cannot be empty")
	}

	search := search.Search("controlplane").Type(kind).Filter(
		"and",
		search.Filter().Terms("ServiceID", serviceID),
		search.Filter().Terms("EndpointName", endpointName),
	)

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

func (s *Store) FindAssignmentByHostPort(ctx datastore.Context, ipAddr string, port uint16) (*AddressAssignment, error) {
	if ipAddr = strings.TrimSpace(ipAddr); ipAddr == "" {
		return nil, fmt.Errorf("hostIP cannot be empty")
	} else if port == 0 {
		return nil, fmt.Errorf("port must be greater than 0")
	}

	search := search.Search("controlplane").Type(kind).Filter(
		"and",
		search.Filter().Terms("IPAddr", ipAddr),
		search.Filter().Terms("Port", strconv.FormatUint(uint64(port), 10)),
	)

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
