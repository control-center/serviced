// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/mattbaird/elastigo/search"
	"github.com/zenoss/serviced/datastore"

	"strconv"
	"strings"
)

//NewStore creates a HostStore
func NewStore() HostStore {
	return HostStore{}
}

//HostStore type for interacting with Host persistent storage
type HostStore struct {
	datastore.DataStore
}

// GetN returns all hosts up to limit.
func (hs *HostStore) GetN(ctx datastore.Context, limit uint64) ([]Host, error) {
	q := datastore.NewQuery(ctx)
	query := search.Query().Search("_exists_:Id")
	search := search.Search("controlplane").Type(kind).Size(strconv.FormatUint(limit, 10)).Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	hosts := make([]Host, results.Len())
	var host Host
	for idx := range hosts {
		err := results.Get(idx, &host)
		if err != nil {
			return nil, err
		}
		hosts[idx] = host
	}

	return hosts, nil
}

//HostKey creates a Key suitable for getting, putting and deleting Hosts
func HostKey(id string) datastore.Key {
	id = strings.TrimSpace(id)
	return datastore.NewKey(kind, id)
}

var kind = "host"
