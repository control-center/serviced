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

package pool

import (
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/datastore"
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
