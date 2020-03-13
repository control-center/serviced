// Copyright 2014 The Serviced Authors.
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
	"errors"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/search"
)

// Store is an interface for accessing pool data.
type Store interface {
	datastore.Store

	GetResourcePools(ctx datastore.Context) ([]ResourcePool, error)
	GetResourcePoolsByRealm(ctx datastore.Context, realm string) ([]ResourcePool, error)
	HasVirtualIP(ctx datastore.Context, poolID, virtualIP string) (bool, error)
}

type store struct{}

var kind = "resourcepool"

// NewStore returns a new object that implements the Store interface.
func NewStore() Store {
	return &store{}
}

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

//GetResourcePools Get a list of all the resource pools
func (s *store) GetResourcePools(ctx datastore.Context) ([]ResourcePool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("PoolStore.GetResourcePools"))
	return query(ctx, "_exists_:ID")
}

// GetResourcePoolsByRealm gets a list of resource pools for a given realm
func (s *store) GetResourcePoolsByRealm(ctx datastore.Context, realm string) ([]ResourcePool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("PoolStore.GetResourcePoolsByRealm"))
	id := strings.TrimSpace(realm)
	if id == "" {
		return nil, errors.New("empty realm not allowed")
	}
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("Realm", id)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// HasVirtualIP returns true if there is a virtual ip found for the given pool
func (s *store) HasVirtualIP(ctx datastore.Context, poolID, virtualIP string) (bool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("PoolStore.HasVirtualIP"))
	if poolID = strings.TrimSpace(poolID); poolID == "" {
		return false, errors.New("empty pool id not allowed")
	} else if virtualIP = strings.TrimSpace(virtualIP); virtualIP == "" {
		return false, errors.New("empty virtual ip not allowed")
	}

	search := search.Search("controlplane").Type(kind).Filter(
		"and",
		search.Filter().Terms("ID", poolID),
		search.Filter().Terms("VirtualIPs.IP", virtualIP),
	)

	results, err := datastore.NewQuery(ctx).Execute(search)
	if err != nil {
		return false, err
	}
	return results.Len() > 0, nil
}

// Key creates a Key suitable for getting, putting and deleting ResourcePools
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

func convert(results datastore.Results) ([]ResourcePool, error) {
	pools := make([]ResourcePool, results.Len())
	for idx := range pools {
		var pool ResourcePool
		err := results.Get(idx, &pool)
		if err != nil {
			return nil, err
		}

		pools[idx] = pool
	}
	return pools, nil
}

func query(ctx datastore.Context, query string) ([]ResourcePool, error) {
	q := datastore.NewQuery(ctx)
	elasticQuery := search.Query().Search(query)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(elasticQuery)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}
