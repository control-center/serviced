// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package addressassignment

import (
	"fmt"
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/serviced/datastore"
)

//NewStore creates a AddressAssignmentStore store
func NewStore() *Store {
	return &Store{}
}

//Store type for interacting with AddressAssignment persistent storage
type Store struct {
	datastore.DataStore
}

func (s *Store) GetServiceAddressAssignments(ctx datastore.Context, serviceID string) ([]*AddressAssignment, error) {
	query := fmt.Sprintf("ServiceID:%s", serviceID)
	q := datastore.NewQuery(ctx)
	elasticQuery := search.Query().Search(query)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(elasticQuery)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

//Key creates a Key suitable for getting, putting and deleting AddressAssignment
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

func convert(results datastore.Results) ([]*AddressAssignment, error) {
	assignments := make([]*AddressAssignment, results.Len())
	for idx := range assignments {
		var aa AddressAssignment
		err := results.Get(idx, &aa)
		if err != nil {
			return nil, err
		}

		assignments[idx] = &aa
	}
	return assignments, nil
}

var kind = "addressassignment"
