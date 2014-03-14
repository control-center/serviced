// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

type Driver interface {
	Put(data *Entity) error

	Get(id string) (*Entity, error)

	Delete(key Key) error

	Query(query Query)([]*Entity, error)
}
