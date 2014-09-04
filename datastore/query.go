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

package datastore

import "errors"

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

// ErrNoSuchElement if requested element not available
var ErrNoSuchElement = errors.New("no such element")

// Results iterates or indexes into the results returned from a query
type Results interface {
	// Next retrieves the next available result into entity and advances the Results to the next available entity.
	// ErrNoSuchElement is returned if no more results.
	Next(entity ValidEntity) error

	// HasNext returns true if a call to next would yield a value or false if no more entities are available
	HasNext() bool

	//Len return the length of the results
	Len() int

	//Len return the length of the results
	Get(idx int, entity ValidEntity) error
}

type query struct {
	ctx Context
}

func (q *query) Execute(query interface{}) (Results, error) {
	ctx := q.ctx
	conn, err := ctx.Connection()
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
	data []JSONMessage
	idx  int
}

func (r *results) Len() int {
	return len(r.data)
}

func (r *results) Get(idx int, entity ValidEntity) error {
	if idx >= len(r.data) {
		return ErrNoSuchElement
	}
	v := r.data[idx]
	if err := SafeUnmarshal(v.Bytes(), entity); err != nil {
		return err
	}
	entity.SetDatabaseVersion(v.Version())
	return nil
}

func (r *results) Next(entity ValidEntity) error {
	if !r.HasNext() {
		return ErrNoSuchElement
	}
	v := r.data[r.idx]
	r.idx = r.idx + 1
	if err := SafeUnmarshal(v.Bytes(), entity); err != nil {
		return err
	}
	entity.SetDatabaseVersion(v.Version())
	return nil
}

func (r *results) HasNext() bool {
	return r.idx < len(r.data)
}

func newResults(data []JSONMessage) Results {
	return &results{data, 0}
}
