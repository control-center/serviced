// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

type JsonMessage []byte

type Driver interface {
	GetConnection() Connection
}

type Connection interface{
	Put(key Key, data JsonMessage) error

	Get(key Key) (JsonMessage, error)

	Delete(key Key) error

	Query(query Query)([]JsonMessage, error)

}
