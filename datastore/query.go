// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
)

type Query interface {
	Set(query interface{})
	Run() (Iterator, error)
}

type Iterator interface {
	Next(interface{}) error
	HasNext() bool
}

type query struct {
	query interface{}
	ctx   Context
}

func newQuery(ctx Context) Query {
	return &query{ctx}
}

func (q *query) Run() (Iterator, error) {
	results := q.ctx.Connection().Query(q)
	return newIterator(results), nil
}

type iterator struct {
	results []JsonMessage
	idx     int
}

func (i *iterator) Next(obj interface{}) error {
	v := i.results[i.idx]
	i.idx = i.idx + 1
	err := json.Unmarshal(v.Bytes(), obj)
	return err
}

func (i *iterator) HasNext() bool {
	return i.idx < len(i.results)
}

func newIterator(results []jsonMessage) Iterator {
	return &iterator{results, 0}
}
