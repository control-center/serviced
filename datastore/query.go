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

	// Run performs the query and returns an Results to the results
	Run() (Results, error)
}

var NoSuchElement error = errors.New("NoSuchElement")

// Results iterates or indexes into the results returned from a query
type Results interface {
	// Next retrieves the next available result into entity and advances the Results to the next available entity.
	// NoSuchElement is returned if no more results.
	Next(entity interface{}) error

	// HasNext returns true if a call to next would yield a value or false if no more entities are available
	HasNext() bool

	//Len return the length of the results
	Len() int

	//Len return the length of the results
	Get(idx int, entity interface{}) error
}

type query struct {
	query interface{}
	ctx   Context
}

func newQuery(ctx Context) Query {
	return &query{nil, ctx}
}

func (q *query) Run() (Results, error) {
	conn, err := q.ctx.Connection()
	if err != nil {
		return nil, err
	}

	results, err := conn.Query(q.query)
	if err != nil {
		return nil, err
	}

	return newResults(results), nil
}

func (q *query) Set(query interface{}) {
	q.query = query
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
		return NoSuchElement
	}
	v := r.data[idx]
	err := json.Unmarshal(v.Bytes(), entity)
	return err
}

func (r *results) Next(entity interface{}) error {
	if !r.HasNext() {
		return NoSuchElement
	}
	v := r.data[r.idx]
	r.idx = r.idx + 1
	err := json.Unmarshal(v.Bytes(), entity)
	return err
}

func (r *results) HasNext() bool {
	return r.idx < len(r.data)
}

func newResults(data []JsonMessage) Results {
	return &results{data, 0}
}
