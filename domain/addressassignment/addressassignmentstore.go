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
