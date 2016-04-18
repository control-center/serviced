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
func NewStore() Store {
	return &storeImpl{}
}

// Store type for interacting with Host persistent storage
type Store interface {
	datastore.EntityStore

	// FindHostsWithPoolID returns all hosts with the given poolid.
	FindHostsWithPoolID(ctx datastore.Context, poolID string) ([]Host, error)

	// GetHostByIP looks up a host by the given ip address
	GetHostByIP(ctx datastore.Context, hostIP string) (*Host, error)

	// GetN returns all hosts up to limit.
	GetN(ctx datastore.Context, limit uint64) ([]Host, error)
}

type storeImpl struct {
	datastore.DataStore
}

// FindHostsWithPoolID returns all hosts with the given poolid.
func (hs *storeImpl) FindHostsWithPoolID(ctx datastore.Context, poolID string) ([]Host, error) {
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
func (hs *storeImpl) GetHostByIP(ctx datastore.Context, hostIP string) (*Host, error) {
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
func (hs *storeImpl) GetN(ctx datastore.Context, limit uint64) ([]Host, error) {
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

func convert(results datastore.Results) ([]Host, error) {
	hosts := make([]Host, results.Len())
	glog.V(4).Infof("Results are %v", results)
	for idx := range hosts {
		var host Host
		err := results.Get(idx, &host)
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof("Adding %v to hosts", host)
		hosts[idx] = host
	}
	return hosts, nil
}

var kind = "host"
