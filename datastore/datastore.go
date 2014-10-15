// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	// Get an entity. Return ErrNoSuchEntity if nothing found for the key.
	Get(ctx Context, key Key, entity ValidEntity) error

	// Delete removes the entity
	Delete(ctx Context, key Key) error
}

//ValidEntity interface for entities that can be stored in the EntityStore
type ValidEntity interface {
	ValidEntity() error
	GetDatabaseVersion() int
	SetDatabaseVersion(int)
}

//New returns a new EntityStore
func New() EntityStore {
	return &DataStore{}
}

type VersionedEntity struct {
	DatabaseVersion int `json:",omitempty"`
}

func (e *VersionedEntity) GetDatabaseVersion() int {
	return e.DatabaseVersion
}

func (e *VersionedEntity) SetDatabaseVersion(i int) {
	e.DatabaseVersion = i
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

// Get an entity. Return ErrNoSuchEntity if nothing found for the key.
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

func (ds *DataStore) serialize(kind string, entity ValidEntity) (JSONMessage, error) {
	// hook for looking up serializers by kind; default json Marshal for now
	data, err := json.Marshal(entity)
	if err != nil {
		return nil, err
	}
	msg := NewJSONMessage(data, entity.GetDatabaseVersion())
	return msg, nil
}

func (ds *DataStore) deserialize(kind string, jsonMsg JSONMessage, entity ValidEntity) error {
	// hook for looking up deserializers by kind; default json Unmarshal for now
	if err := SafeUnmarshal(jsonMsg.Bytes(), entity); err != nil {
		return err
	}
	entity.SetDatabaseVersion(jsonMsg.Version())
	return nil
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
