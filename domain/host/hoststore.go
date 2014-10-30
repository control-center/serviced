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
	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/glog"

	"errors"
	"strconv"
	"strings"
)

//NewStore creates a HostStore
func NewStore() *HostStore {
	return &HostStore{}
}

//HostStore type for interacting with Host persistent storage
type HostStore struct {
	datastore.DataStore
}

// FindHostsWithPoolID returns all hosts with the given poolid.
func (hs *HostStore) FindHostsWithPoolID(ctx datastore.Context, poolID string) ([]*Host, error) {
	id := strings.TrimSpace(poolID)
	if id == "" {
		return nil, errors.New("empty poolId not allowed")
	}
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("PoolID", id)
	search := search.Search("controlplane").Type(kind).Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetN returns all hosts up to limit.
func (hs *HostStore) GetN(ctx datastore.Context, limit uint64) ([]*Host, error) {
	q := datastore.NewQuery(ctx)
	query := search.Query().Search("_exists_:ID")
	search := search.Search("controlplane").Type(kind).Size(strconv.FormatUint(limit, 10)).Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

//HostKey creates a Key suitable for getting, putting and deleting Hosts
func HostKey(id string) datastore.Key {
	id = strings.TrimSpace(id)
	return datastore.NewKey(kind, id)
}

func convert(results datastore.Results) ([]*Host, error) {
	hosts := make([]*Host, results.Len())
	glog.V(4).Infof("Results are %v", results)
	for idx := range hosts {
		var host Host
		err := results.Get(idx, &host)
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof("Adding %v to hosts", host)
		hosts[idx] = &host
	}
	return hosts, nil
}

var kind = "host"
