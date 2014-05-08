// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package addressassignment

import (
	"github.com/zenoss/serviced/datastore"
)

//NewAddressAssignmentStore creates a AddressAssignmentStore store
func NewStore() *Store {
	return &Store{}
}

//Store type for interacting with AddressAssignment persistent storage
type Store struct {
	datastore.DataStore
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
