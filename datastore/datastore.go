// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
)

// Interface for storing and retrieving data types from a datastore.
type DataStore interface {

	// Put adds or updates an entity
	Put(ctx Context, key Key, entity interface{}) error

	// Get adds or updates an entity. Return ErrNoSuchEntity if nothing found for the
	Get(ctx Context, Key Key, entity interface{}) error

	// Delete removes the entity
	Delete(ctx Context, key Key) error

	// Query returns a Query type to be exectued
	Query(ctx Context) Query
}

func New() DataStore {
	return &dataStore{}
}

type dataStore struct{}

func (ds *dataStore) Put(ctx Context, key Key, entity interface{}) error {
	jsonMsg, err := ds.serialize(key.Kind(), entity)
	if err != nil {
		return err
	}
	conn, err := ctx.Connection()
	if err != nil {
		return err
	}
	return conn.Put(key, jsonMsg)
}

func (ds *dataStore) Get(ctx Context, key Key, entity interface{}) error {
	conn, err := ctx.Connection()
	if err != nil {
		return err
	}

	if jsonMsg, err := conn.Get(key); err != nil {
		return err
	} else {
		err = ds.deserialize(key.Kind(), jsonMsg, entity)
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

func (ds *dataStore) serialize(kind string, entity interface{}) (JsonMessage, error) {
	// hook for looking up serializers by kind; default json Marshal for now
	data, err := json.Marshal(entity)
	if err != nil {
		return nil, err
	}
	return &jsonMessage{data}, nil
}

func (ds *dataStore) deserialize(kind string, jsonMsg JsonMessage, entity interface{}) error {
	// hook for looking up deserializers by kind; default json Unmarshal for now
	return SafeUnmarshal(jsonMsg.Bytes(), entity)
}
