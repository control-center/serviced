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

package host

import (
	"errors"
	"strconv"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/search"
)

// Store is an interface for accessing host data.
type Store interface {
	datastore.Store

	FindHostsWithPoolID(ctx datastore.Context, poolID string) ([]Host, error)
	GetHostByIP(ctx datastore.Context, hostIP string) (*Host, error)
	GetN(ctx datastore.Context, limit uint64) ([]Host, error)
}

type store struct{}

var kind = "host"

// NewStore returns an implementation of the Store interface.
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

// FindHostsWithPoolID returns all hosts with the given poolid.
func (s *store) FindHostsWithPoolID(ctx datastore.Context, poolID string) ([]Host, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("HostStore.FindHostsWithPoolID"))
	id := strings.TrimSpace(poolID)
	if id == "" {
		return nil, errors.New("empty poolId not allowed")
	}
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("PoolID", id)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetHostByIP looks up a host by the given ip address
func (s *store) GetHostByIP(ctx datastore.Context, hostIP string) (*Host, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("HostStore.GetHostByIP"))
	if hostIP = strings.TrimSpace(hostIP); hostIP == "" {
		return nil, errors.New("empty hostIP not allowed")
	}

	query := search.Query().Term("IPs.IPAddress", hostIP)
	search := search.Search("controlplane").Type(kind).Query(query)
	results, err := datastore.NewQuery(ctx).Execute(search)
	if err != nil {
		return nil, err
	}

	if results.Len() == 0 {
		return nil, nil
	} else if hosts, err := convert(results); err != nil {
		return nil, err
	} else {
		return &hosts[0], nil
	}
}

// GetN returns all hosts up to limit.
func (s *store) GetN(ctx datastore.Context, limit uint64) ([]Host, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("HostStore.GetN"))
	q := datastore.NewQuery(ctx)
	query := search.Query().Search("_exists_:ID")
	search := search.Search("controlplane").Type(kind).Size(strconv.FormatUint(limit, 10)).Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// Key creates a Key suitable for getting, putting and deleting Hosts
func Key(id string) datastore.Key {
	id = strings.TrimSpace(id)
	return datastore.NewKey(kind, id)
}

func convert(results datastore.Results) ([]Host, error) {
	hosts := make([]Host, results.Len())
	for idx := range hosts {
		var host Host
		err := results.Get(idx, &host)
		if err != nil {
			return nil, err
		}
		hosts[idx] = host
	}
	return hosts, nil
}
