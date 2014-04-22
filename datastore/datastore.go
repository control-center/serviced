// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
	"errors"
)

var (
	//ErrNilEntity returned when Entity parameter is nil
	ErrNilEntity = errors.New("nil Entity")
	//ErrNilContext returned when Context parameter is nil
	ErrNilContext = errors.New("nil Context")
	//ErrNilKind returned when Kind parameter is nil
	ErrNilKey = errors.New("nil Key")
	//ErrEmptyKindID returned when a key has an empty ID
	ErrEmptyKindID = errors.New("empty Kind id")
	//ErrEmptyKind returned when a key has an empty kind
	ErrEmptyKind = errors.New("empty Kind")
)

// EntityStore interface for storing and retrieving data types from a datastore.
type EntityStore interface {

	// Put adds or updates an entity
	Put(ctx Context, key Key, entity ValidEntity) error

	// Get adds an entity. Return ErrNoSuchEntity if nothing found for the key.
	Get(ctx Context, key Key, entity ValidEntity) error

	// Delete removes the entity
	Delete(ctx Context, key Key) error
}

//ValidEntity interface for entities that can be stored in the EntityStore
type ValidEntity interface {
	ValidEntity() error
}

//New returns a new EntityStore
func New() EntityStore {
	return &DataStore{}
}

//DataStore EntityStore type
type DataStore struct{}

// Put adds or updates an entity
func (ds *DataStore) Put(ctx Context, key Key, entity ValidEntity) error {
	if ctx == nil {
		return ErrNilContext
	}
	if key == nil {
		return ErrNilKey
	}
	if entity == nil {
		return ErrNilEntity
	}
	if err := validKey(key); err != nil {
		return err
	}

	if err := entity.ValidEntity(); err != nil {
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

// Get adds an entity. Return ErrNoSuchEntity if nothing found for the key.
func (ds *DataStore) Get(ctx Context, key Key, entity ValidEntity) error {
	if ctx == nil {
		return ErrNilContext
	}
	if key == nil {
		return ErrNilKey
	}
	if entity == nil {
		return ErrNilEntity
	}
	if err := validKey(key); err != nil {
		return err
	}

	conn, err := ctx.Connection()
	if err != nil {
		return err
	}

	jsonMsg, err := conn.Get(key)
	if err != nil {
		return err
	}
	err = ds.deserialize(key.Kind(), jsonMsg, entity)
	return err

}

// Delete removes the entity
func (ds *DataStore) Delete(ctx Context, key Key) error {
	if ctx == nil {
		return ErrNilContext
	}
	if key == nil {
		return ErrNilKey
	}
	if err := validKey(key); err != nil {
		return err
	}
	conn, err := ctx.Connection()
	if err != nil {
		return err
	}

	return conn.Delete(key)
}

func (ds *DataStore) serialize(kind string, entity interface{}) (JSONMessage, error) {
	// hook for looking up serializers by kind; default json Marshal for now
	data, err := json.Marshal(entity)
	if err != nil {
		return nil, err
	}
	return NewJSONMessage(data), nil
}

func (ds *DataStore) deserialize(kind string, jsonMsg JSONMessage, entity interface{}) error {
	// hook for looking up deserializers by kind; default json Unmarshal for now
	return SafeUnmarshal(jsonMsg.Bytes(), entity)
}

func validKey(k Key) error {
	if k.ID() == "" {
		return ErrEmptyKindID
	}
	if k.Kind() == "" {
		return ErrEmptyKind
	}
	return nil
}
