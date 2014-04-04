// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"github.com/zenoss/serviced/datastore/context"
	"github.com/zenoss/serviced/datastore/key"
	"github.com/zenoss/serviced/datastore/driver"

	"encoding/json"
	"errors"
)

var (
	//ErrNilEntity returned when Entity parameter is nil
	ErrNilEntity = errors.New("nil Entity")
	//ErrNilContext returned when Context parameter is nil
	ErrNilContext = errors.New("nil Context")
	//ErrNilKind returned when Kind parameter is nil
	ErrNilKind = errors.New("nil Kind")
)

// EntityStore interface for storing and retrieving data types from a datastore.
type EntityStore interface {

	// Put adds or updates an entity
	Put(ctx context.Context, key key.Key, entity ValidEntity) error

	// Get adds an entity. Return ErrNoSuchEntity if nothing found for the key.
	Get(ctx context.Context, key key.Key, entity ValidEntity) error

	// Delete removes the entity
	Delete(ctx context.Context, key key.Key) error
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
func (ds *DataStore) Put(ctx context.Context, key key.Key, entity ValidEntity) error {
	if ctx == nil {
		return ErrNilContext
	}
	if key == nil {
		return ErrNilKind
	}
	if entity == nil {
		return ErrNilEntity
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
func (ds *DataStore) Get(ctx context.Context, key key.Key, entity ValidEntity) error {
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

	jsonMsg, err := conn.Get(key)
	if err != nil {
		return err
	}
	err = ds.deserialize(key.Kind(), jsonMsg, entity)
	return err

}

// Delete removes the entity
func (ds *DataStore) Delete(ctx context.Context, key key.Key) error {
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

func (ds *DataStore) serialize(kind string, entity interface{}) (driver.JSONMessage, error) {
	// hook for looking up serializers by kind; default json Marshal for now
	data, err := json.Marshal(entity)
	if err != nil {
		return nil, err
	}
	return driver.NewJSONMessage(data), nil
}

func (ds *DataStore) deserialize(kind string, jsonMsg driver.JSONMessage, entity interface{}) error {
	// hook for looking up deserializers by kind; default json Unmarshal for now
	return SafeUnmarshal(jsonMsg.Bytes(), entity)
}
