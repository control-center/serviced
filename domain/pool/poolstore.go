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
	"github.com/control-center/serviced/datastore/elastic"
	"strings"

	"github.com/control-center/serviced/datastore"
)

//NewStore creates a ResourcePool store
func NewStore() Store {
	return &storeImpl{}
}

// Store type for interacting with ResourcePool persistent storage
type Store interface {
	datastore.EntityStore

	// GetResourcePools Get a list of all the resource pools
	GetResourcePools(ctx datastore.Context) ([]ResourcePool, error)

	// GetResourcePoolsByRealm gets a list of resource pools for a given realm
	GetResourcePoolsByRealm(ctx datastore.Context, realm string) ([]ResourcePool, error)

	// HasVirtualIP returns true if there is a virtual ip found for the given pool
	HasVirtualIP(ctx datastore.Context, poolID, virtualIP string) (bool, error)
}

type storeImpl struct {
	datastore.DataStore
}

//GetResourcePools Get a list of all the resource pools
func (ps *storeImpl) GetResourcePools(ctx datastore.Context) ([]ResourcePool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("PoolStore.GetResourcePools"))
	req := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": map[string]interface{}{
					"exists": map[string]interface{}{
						"field": "ID",
					},
				},
				"filter": map[string]interface{}{
					"match": map[string]interface{}{
						"type": kind,
					},
				},
			},
		},
	}
	return query(ctx, req)
}

// GetResourcePoolsByRealm gets a list of resource pools for a given realm
func (s *storeImpl) GetResourcePoolsByRealm(ctx datastore.Context, realm string) ([]ResourcePool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("PoolStore.GetResourcePoolsByRealm"))
	id := strings.TrimSpace(realm)
	if id == "" {
		return nil, errors.New("empty realm not allowed")
	}
	q := datastore.NewQuery(ctx)

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"Realm": id}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// HasVirtualIP returns true if there is a virtual ip found for the given pool
func (s *storeImpl) HasVirtualIP(ctx datastore.Context, poolID, virtualIP string) (bool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("PoolStore.HasVirtualIP"))
	if poolID = strings.TrimSpace(poolID); poolID == "" {
		return false, errors.New("empty pool id not allowed")
	} else if virtualIP = strings.TrimSpace(virtualIP); virtualIP == "" {
		return false, errors.New("empty virtual ip not allowed")
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"ID": poolID}},
					{"term": map[string]string{"VirtualIPs.IP": virtualIP}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return false, err
	}

	results, err := datastore.NewQuery(ctx).Execute(search)
	if err != nil {
		return false, err
	}
	return results.Len() > 0, nil
}

//Key creates a Key suitable for getting, putting and deleting ResourcePools
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

func query(ctx datastore.Context, query interface{}) ([]ResourcePool, error) {
	q := datastore.NewQuery(ctx)

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}

	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

var kind = "resourcepool"
