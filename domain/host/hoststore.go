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

type HostStore interface {
	Put(ctx datastore.Context, host *Host) error

	Get(ctx datastore.Context, id string) (*Host, error)

	Delete(ctx datastore.Context, id string) error

	// GetUpTo returns all hosts up to limit.
	GetUpTo(ctx datastore.Context, limit uint64) ([]Host, error)
}

func NewStore() HostStore {
	return &hostStore{datastore.New()}
}

type hostStore struct {
	ds datastore.DataStore
}

var kind string = "host"

func (hs *hostStore) Put(ctx datastore.Context, host *Host) error {
	return hs.ds.Put(ctx, hostKey(host), host)
}

func (hs *hostStore) Get(ctx datastore.Context, id string) (*Host, error) {
	var host Host
	err := hs.ds.Get(ctx, makeKey(id), &host)
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (hs *hostStore) GetUpTo(ctx datastore.Context, limit uint64) ([]Host, error) {
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

func (hs *hostStore) Delete(ctx datastore.Context, id string) error {
	return hs.ds.Delete(ctx, makeKey(id))
}

func makeKey(id string) datastore.Key {
	id = strings.TrimSpace(id)

	return datastore.NewKey(kind, id)
}
func hostKey(host *Host) datastore.Key {
	return makeKey(host.Id)
}
