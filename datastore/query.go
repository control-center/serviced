// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

//TODO: figure this out
type Query interface {
	Run() (Iterator, error)
}

type Iterator interface {
	Next(interface{}) error
	HasNext() bool
}

type query struct {
	ctx Context
}

func newQuery(ctx Context)Query{
	return &query{ctx}
}
func (q *query) Run() (Iterator, error) {
	return nil, nil
}
