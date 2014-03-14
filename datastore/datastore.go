// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
)

type Key struct {
	id   string
	kind string
}

// Kind returns the key's kind (also known as entity type).
func (k *Key) Kind() string {
	return k.kind
}

// Kind returns the key's kind (also known as entity type).
func (k *Key) ID() string {
	return k.id
}

// Entity is the data to be stored in the store. Key is the unique key. Type is the type of the data being stored.
// Payload is the actual data being stored.  It is up to the datastore driver to serialize and deserialize the Entity
// and the payload
//type Entity struct {
//	Key     Key
//	Payload interface{}
//}

//func NewEntity(key Key, payload interface{}) *Entity {
//	return &Entity{key, payload}
//}
func New() DataStore {
	return &dataStore{}
}

type DataStore interface {
	Put(ctx Context, key Key, data interface{}) error

	Get(ctx Context, Key Key, data interface{}) error

	Delete(ctx Context, key Key) error

	Query(ctx Context) Query
}

type dataStore struct{}

func (ds *dataStore) Put(ctx Context, key Key, data interface{}) error {
	jsonMsg, err := ds.serialize(key.Kind(), data)
	if err != nil {
		return err
	}
	return ctx.Connection().Put(key, jsonMsg)
}

func (ds *dataStore) Get(ctx Context, key Key, obj interface{}) error {
	if jsonMsg, err := ctx.Connection().Get(key); err != nil {
		return err
	} else {
		err = ds.deserialize(key.Kind(), jsonMsg, obj)
		return err
	}
}

func (ds *dataStore) Delete(ctx Context, key Key) error {
	return ctx.Connection().Delete(key)
}

func (ds *dataStore) Query(ctx Context) Query {
	return newQuery(ctx)
}

func (ds *dataStore) serialize(kind string, obj interface{}) (JsonMessage, error) {
	// hook for looking up serializers by kind; default json Marshal for now
	return json.Marshal(obj)
}

func (ds *dataStore) deserialize(kind string, jsonMsg JsonMessage, obj interface{}) error {
	// hook for looking up deserializers by kind; default json Unmarshal for now
	return json.Unmarshal(jsonMsg, obj)
}
