// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
	"errors"
)

var (
	ErrNilEntity  = errors.New("Nil Entity")
	ErrNilContext = errors.New("Nil Context")
	ErrNilKind    = errors.New("Nil Kind")
)

// Interface for storing and retrieving data types from a datastore.
type EntityStore interface {

	// Put adds or updates an entity
	Put(ctx Context, key Key, entity ValidEntity) error

	// Get adds an entity. Return ErrNoSuchEntity if nothing found for the key.
	Get(ctx Context, key Key, entity ValidEntity) error

	// Delete removes the entity
	Delete(ctx Context, key Key) error
}

type ValidEntity interface {
	ValidateEntity() error
}

func New() EntityStore {
	return &DataStore{}
}

type DataStore struct{}

func (ds *DataStore) Put(ctx Context, key Key, entity ValidEntity) error {
	if ctx == nil {
		return ErrNilContext
	}
	if key == nil {
		return ErrNilKind
	}
	if entity == nil {
		return ErrNilEntity
	}
	if err := entity.ValidateEntity(); err != nil {
		return err
	}

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

func (ds *DataStore) Get(ctx Context, key Key, entity ValidEntity) error {
	if ctx == nil {
		return ErrNilContext
	}
	if key == nil {
		return ErrNilKind
	}
	if entity == nil {
		return ErrNilEntity
	}
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

func (ds *DataStore) Delete(ctx Context, key Key) error {
	if ctx == nil {
		return ErrNilContext
	}
	if key == nil {
		return ErrNilKind
	}
	conn, err := ctx.Connection()
	if err != nil {
		return err
	}

	return conn.Delete(key)
}

func (ds *DataStore) serialize(kind string, entity interface{}) (JsonMessage, error) {
	// hook for looking up serializers by kind; default json Marshal for now
	data, err := json.Marshal(entity)
	if err != nil {
		return nil, err
	}
	return &jsonMessage{data}, nil
}

func (ds *DataStore) deserialize(kind string, jsonMsg JsonMessage, entity interface{}) error {
	// hook for looking up deserializers by kind; default json Unmarshal for now
	return SafeUnmarshal(jsonMsg.Bytes(), entity)
}
