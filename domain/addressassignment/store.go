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

// Store is an interface for accessing address assignment data.
type Store interface {
	datastore.Store

	GetAllAddressAssignments(ctx datastore.Context) ([]AddressAssignment, error)
	GetServiceAddressAssignments(ctx datastore.Context, serviceID string) ([]AddressAssignment, error)
	GetServiceAddressAssignmentsByPort(ctx datastore.Context, poolID string, port uint16) ([]AddressAssignment, error)
	FindAssignmentByHostPort(ctx datastore.Context, poolID string, ipAddr string, port uint16) (*AddressAssignment, error)
	FindAssignmentByServiceEndpoint(ctx datastore.Context, serviceID, endpointName string) (*AddressAssignment, error)
}

type store struct{}

// NewStore returns a new object that implements the Store interface.
func NewStore() Store {
	return &store{}
}

var kind = "addressassignment"

// Put adds or updates an entity
func (s *store) Put(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	return datastore.Put(ctx, key, entity)
}

// Get an entity. Return ErrNoSuchEntity if nothing found for the key.
func (s *store) Get(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	return datastore.Get(ctx, key, entity)
}

// Delete removes the entity
func (s *store) Delete(ctx datastore.Context, key datastore.Key) error {
	return datastore.Delete(ctx, key)
}

// GetAllAddressAssignments stuff
func (s *store) GetAllAddressAssignments(ctx datastore.Context) ([]AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.GetAllAddressAssignments"))
	q := datastore.NewQuery(ctx)
	search := search.Search("controlplane").Type(kind).Size("50000")
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetServiceAddressAssignments stuff
func (s *store) GetServiceAddressAssignments(ctx datastore.Context, serviceID string) ([]AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.GetServiceAddressAssignments"))
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("ServiceID", serviceID)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetServiceAddressAssignmentsByPort stuff
func (s *store) GetServiceAddressAssignmentsByPort(ctx datastore.Context, poolID string, port uint16) ([]AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.GetServiceAddressAssignmentsByPort"))
	if poolID = strings.TrimSpace(poolID); poolID == "" {
		return nil, fmt.Errorf("poolID cannot be empty")
	} else if port == 0 {
		return nil, fmt.Errorf("port must be greater than 0")
	}

	search := search.Search("controlplane").Type(kind).Size("50000").Filter(
		"and",
		search.Filter().Terms("PoolID", poolID),
		search.Filter().Terms("Port", strconv.FormatUint(uint64(port), 10)),
	)

	results, err := datastore.NewQuery(ctx).Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// FindAssignmentByServiceEndpoint stuff
func (s *store) FindAssignmentByServiceEndpoint(ctx datastore.Context, serviceID, endpointName string) (*AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.FindAssignmentByServiceEndpoint"))
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

// FindAssignmentByHostPort suff
func (s *store) FindAssignmentByHostPort(ctx datastore.Context, poolID string, ipAddr string, port uint16) (*AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddressAssignmentStore.FindAssignmentByHostPort"))
	if poolID = strings.TrimSpace(poolID); poolID == "" {
		return nil, fmt.Errorf("poolID cannot be empty")
	} else if ipAddr = strings.TrimSpace(ipAddr); ipAddr == "" {
		return nil, fmt.Errorf("hostIP cannot be empty")
	} else if port == 0 {
		return nil, fmt.Errorf("port must be greater than 0")
	}

	search := search.Search("controlplane").Type(kind).Size("50000").Filter(
		"and",
		search.Filter().Terms("PoolID", poolID),
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
