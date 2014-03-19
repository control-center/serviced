// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
	"errors"
)

// Query is a query used to search for and return entities from a datastore
type Query interface {

	// Set accepts a query. For now this query is specific to the underlying Connection and Driver implementation. There
	// is no abstraction.
	Set(query interface{})

	// Run performs the query and returns an iterator to the results
	Run() (Iterator, error)
}

var NoSuchElement error = errors.New("NoSuchElement")

// Iterator iterates of the results returned from a query
type Iterator interface {
	// Next retrieves the next available result into entity and advances the iterator to the next available entity.
	// NoSuchElement is returned if no more results.
	Next(entity interface{}) error

	// HasNext returns true if a call to next would yield a value or false if no more entities are available
	HasNext() bool
}

type query struct {
	query interface{}
	ctx   Context
}

func newQuery(ctx Context) Query {
	return &query{nil, ctx}
}

func (q *query) Run() (Iterator, error) {
	conn, err := q.ctx.Connection()
	if err != nil {
		return nil, err
	}

	results, err := conn.Query(q)
	if err != nil {
		return nil, err
	}

	return newIterator(results), nil
}

func (q *query) Set(query interface{}) {
	q.query = query
}

type iterator struct {
	results []JsonMessage
	idx     int
}

func (i *iterator) Next(entity interface{}) error {
	if !i.HasNext() {
		return NoSuchElement
	}
	v := i.results[i.idx]
	i.idx = i.idx + 1
	err := json.Unmarshal(v.Bytes(), entity)
	return err
}

func (i *iterator) HasNext() bool {
	return i.idx < len(i.results)
}

func newIterator(results []JsonMessage) Iterator {
	return &iterator{results, 0}
}
