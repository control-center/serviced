// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package event

import (
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"

	"errors"
	"fmt"
	"strconv"
	"strings"
)

//NewStore creates an Store
func NewStore() *Store {
	return &Store{}
}

//Store is a entity store type for interacting with Event persistent storage
type Store struct {
	datastore.DataStore
}

// FindEventsWithServiceID returns all events with the given service id.
func (es *Store) FindEventsWithServiceID(ctx datastore.Context, serviceID string) ([]*Event, error) {
	id := strings.TrimSpace(serviceID)
	if id == "" {
		return nil, errors.New("empty serviceID not allowed")
	}

	q := datastore.NewQuery(ctx)
	queryString := fmt.Sprintf("ServiceID:%s", id)
	query := search.Query().Search(queryString)
	search := search.Search("controlplane").Type(kind).Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetN returns all hosts up to limit.
func (es *Store) GetN(ctx datastore.Context, limit uint64) ([]*Event, error) {
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
func EventKey(id string) datastore.Key {
	id = strings.TrimSpace(id)
	return datastore.NewKey(kind, id)
}

func convert(results datastore.Results) ([]*Event, error) {
	events := make([]*Event, results.Len())
	glog.V(4).Infof("Results are %v", results)
	for idx := range events {
		var event Event
		err := results.Get(idx, &event)
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof("Adding %v to events", event)
		events[idx] = &event
	}
	return events, nil
}

var kind = "event"
