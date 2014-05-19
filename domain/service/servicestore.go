// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package service

import (
	"github.com/mattbaird/elastigo/search"
	"github.com/zenoss/serviced/datastore"

	"errors"
	"fmt"
	"strings"
)

//NewStore creates a Service  store
func NewStore() *Store {
	return &Store{}
}

//Store type for interacting with Service persistent storage
type Store struct {
	datastore.DataStore
}

//GetServices returns all services
func (s *Store) GetServices(ctx datastore.Context) ([]*Service, error) {
	return query(ctx, "_exists_:Id")
}

//GetTaggedServices returns services with the given tags
func (s *Store) GetTaggedServices(ctx datastore.Context, tags ...string) ([]*Service, error) {
	if len(tags) == 0 {
		return nil, errors.New("empty tags not allowed")
	}
	qs := strings.Join(tags, " AND ")
	return query(ctx, qs)
}

//GetServicesByPool returns services with the given pool id
func (s *Store) GetServicesByPool(ctx datastore.Context, poolID string) ([]*Service, error) {
	id := strings.TrimSpace(poolID)
	if id == "" {
		return nil, errors.New("empty poolID not allowed")
	}
	queryString := fmt.Sprintf("PoolID:%s", id)
	return query(ctx, queryString)
}

//GetChildServices returns services that are children of the given parent service id
func (s *Store) GetChildServices(ctx datastore.Context, parentID string) ([]*Service, error) {
	id := strings.TrimSpace(parentID)
	if id == "" {
		return nil, errors.New("empty parent service id not allowed")
	}

	queryString := fmt.Sprintf("ParentServiceID:%s", parentID)
	return query(ctx, queryString)
}

func query(ctx datastore.Context, query string) ([]*Service, error) {
	q := datastore.NewQuery(ctx)
	elasticQuery := search.Query().Search(query)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(elasticQuery)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func convert(results datastore.Results) ([]*Service, error) {
	svcs := make([]*Service, results.Len())
	for idx := range svcs {
		var svc Service
		err := results.Get(idx, &svc)
		if err != nil {
			return nil, err
		}
		svcs[idx] = &svc
	}
	return svcs, nil
}

//Key creates a Key suitable for getting, putting and deleting ResourcePools
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

var kind = "service"
