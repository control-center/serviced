// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package pool

import (
	"github.com/mattbaird/elastigo/search"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
)

//NewStore creates a ResourcePool store
func NewStore() *Store {
	return &Store{}
}

//Store type for interacting with ResourcePool persistent storage
type Store struct {
	datastore.DataStore
}

//GetResourcePools Get a list of all the resource pools
func (ps *Store) GetResourcePools(ctx datastore.Context) ([]*ResourcePool, error) {
	glog.V(3).Infof("Pool Store.GetResourcePools")
	q := datastore.NewQuery(ctx)
	query := search.Query().Search("_exists_:ID")
	search := search.Search("controlplane").Type(kind).Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

//Key creates a Key suitable for getting, putting and deleting ResourcePools
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

func convert(results datastore.Results) ([]*ResourcePool, error) {
	pools := make([]*ResourcePool, results.Len())
	for idx := range pools {
		var pool ResourcePool
		err := results.Get(idx, &pool)
		if err != nil {
			return nil, err
		}

		pools[idx] = &pool
	}
	return pools, nil
}

var kind = "resourcepool"
