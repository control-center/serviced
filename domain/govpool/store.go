// Copyright 2015 The Serviced Authors.
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

package govpool

import (
	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/search"
)

// NewStore creates a new GovernedPool store
func NewStore() *Store {
	return &Store{}
}

// Store is for interacting with ResourcePool persistent storage
type Store struct {
	ds datastore.DataStore
}

// Put adds or updates a governed pool
func (s *Store) Put(ctx datastore.Context, pool *GovernedPool) error {
	return s.ds.Put(ctx, Key(pool.RemotePoolID), pool)
}

// Get returns a governed pool by its id.  Return ErrNoSuchEntity if not found.
func (s *Store) Get(ctx datastore.Context, remotePoolID string) (*GovernedPool, error) {
	var pool GovernedPool
	if err := s.ds.Get(ctx, Key(remotePoolID), &pool); err != nil {
		return nil, err
	}
	return &pool, nil
}

// Delete removes a governed pool if it exists
func (s *Store) Delete(ctx datastore.Context, remotePoolID string) error {
	return s.ds.Delete(ctx, Key(remotePoolID))
}

// GetGovernedPools gets a list of all governed pools
func (s *Store) GetGovernedPools(ctx datastore.Context) ([]GovernedPool, error) {
	return query(ctx, "_exists_:RemotePoolID")
}

// Key creates a Key suitable for getting, putting, and deleting governed pools
func Key(poolID string) datastore.Key {
	return datastore.NewKey(kind, poolID)
}

func query(ctx datastore.Context, query string) ([]GovernedPool, error) {
	q := datastore.NewQuery(ctx)
	esquery := search.Query().Search(query)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(esquery)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func convert(results datastore.Results) ([]GovernedPool, error) {
	pools := make([]GovernedPool, results.Len())
	for idx := range pools {
		var pool GovernedPool
		if err := results.Get(idx, &pool); err != nil {
			return nil, err
		}
		pools[idx] = pool
	}
	return pools, nil
}

var kind = "governedpool"