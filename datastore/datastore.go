// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
)

type Key interface {
	Kind() string
	ID() string
}

type DataStore interface {
	Put(ctx Context, key Key, data interface{}) error

	Get(ctx Context, Key Key, data interface{}) error

	Delete(ctx Context, key Key) error

	Query(ctx Context) Query
}

type key struct {
	id   string
	kind string
}

// Kind returns the key's kind (also known as entity type).
func (k *key) Kind() string {
	return k.kind
}

// Kind returns the key's kind (also known as entity type).
func (k *key) ID() string {
	return k.id
}

func NewKey(kind string, id string) Key {
	return &key{id, kind}
}

func New() DataStore {
	return &dataStore{}
}

type dataStore struct{}

func (ds *dataStore) Put(ctx Context, key Key, data interface{}) error {
	jsonMsg, err := ds.serialize(key.Kind(), data)
	if err != nil {
		return err
	}
	conn, err := ctx.Connection()
	if err != nil {
		return err
	}
	return conn.Put(key, jsonMsg)
}

func (ds *dataStore) Get(ctx Context, key Key, obj interface{}) error {
	conn, err := ctx.Connection()
	if err != nil {
		return err
	}

	if jsonMsg, err := conn.Get(key); err != nil {
		return err
	} else {
		err = ds.deserialize(key.Kind(), jsonMsg, obj)
		return err
	}
}

func (ds *dataStore) Delete(ctx Context, key Key) error {
	conn, err := ctx.Connection()
	if err != nil {
		return err
	}

	return conn.Delete(key)
}

func (ds *dataStore) Query(ctx Context) Query {
	return newQuery(ctx)
}

func (ds *dataStore) serialize(kind string, obj interface{}) (JsonMessage, error) {
	// hook for looking up serializers by kind; default json Marshal for now
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return &jsonMessage{data}, nil
}

func (ds *dataStore) deserialize(kind string, jsonMsg JsonMessage, obj interface{}) error {
	// hook for looking up deserializers by kind; default json Unmarshal for now
	return json.Unmarshal(jsonMsg.Bytes(), obj)
}
