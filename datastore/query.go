// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"errors"
)

// Query is a query used to search for and return entities from a datastore
type Query interface {

	// Execute performs the query and returns an Results to the results.  For now this query is specific to the
	// underlying Connection and Driver implementation.
	Execute(query interface{}) (Results, error)
}

// NewQuery returns a Query type to be executed
func NewQuery(ctx Context) Query {
	return &query{ctx}
}

var ErrNoSuchElement error = errors.New("NoSuchElement")

// Results iterates or indexes into the results returned from a query
type Results interface {
	// Next retrieves the next available result into entity and advances the Results to the next available entity.
	// ErrNoSuchElement is returned if no more results.
	Next(entity interface{}) error

	// HasNext returns true if a call to next would yield a value or false if no more entities are available
	HasNext() bool

	//Len return the length of the results
	Len() int

	//Len return the length of the results
	Get(idx int, entity interface{}) error
}

type query struct {
	ctx Context
}

func (q *query) Execute(query interface{}) (Results, error) {
	conn, err := q.ctx.Connection()
	if err != nil {
		return nil, err
	}

	results, err := conn.Query(query)
	if err != nil {
		return nil, err
	}

	return newResults(results), nil
}

type results struct {
	data []JsonMessage
	idx  int
}

func (r *results) Len() int {
	return len(r.data)
}

func (r *results) Get(idx int, entity interface{}) error {
	if idx >= len(r.data) {
		return ErrNoSuchElement
	}
	v := r.data[idx]
	err := SafeUnmarshal(v.Bytes(), entity)
	return err
}

func (r *results) Next(entity interface{}) error {
	if !r.HasNext() {
		return ErrNoSuchElement
	}
	v := r.data[r.idx]
	r.idx = r.idx + 1
	err := SafeUnmarshal(v.Bytes(), entity)
	return err
}

func (r *results) HasNext() bool {
	return r.idx < len(r.data)
}

func newResults(data []JsonMessage) Results {
	return &results{data, 0}
}
