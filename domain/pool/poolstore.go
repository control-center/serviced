// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package pool

import (
	"github.com/mattbaird/elastigo/search"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/context"
	"github.com/zenoss/serviced/datastore/key"

)

//NewStore creates a HostStore
func NewStore() *Store {
	return &Store{}
}

//HostStore type for interacting with Host persistent storage
type Store struct {
	datastore.DataStore
}

//GetResourcePools Get a list of all the resource pools
func (ps *Store) GetResourcePools(ctx context.Context) ([]*ResourcePool, error) {
	glog.V(3).Infof("Pool Store.GetResourcePools")
	q := datastore.NewQuery(ctx)
	query := search.Query().Search("_exists_:Id")
	search := search.Search("controlplane").Type(kind).Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

//Key creates a Key suitable for getting, putting and deleting Hosts
func Key(id string) key.Key {
	return key.New(kind, id)
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
