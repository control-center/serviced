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
	"fmt"
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
	GetType() string
	SetType(string)
	GetSeqNo() int
	SetSeqNo(int)
	GetPrimaryTerm() int
	SetPrimaryTerm(int)
}

//New returns a new EntityStore
func New() EntityStore {
	return &DataStore{}
}

type VersionedEntity struct {
	DatabaseVersion int    `json:"_version,omitempty"`
	IfSeqNo         int    `json:"_if_seq_no,omitempty"`
	IfPrimaryTerm   int    `json:"_if_primary_term,omitempty"`
	Type            string `json:"type,omitempty"`
}

func (e *VersionedEntity) GetDatabaseVersion() int {
	return e.DatabaseVersion
}

func (e *VersionedEntity) SetDatabaseVersion(i int) {
	e.DatabaseVersion = i
}

func (e *VersionedEntity) GetType() string {
	return e.Type
}

func (e *VersionedEntity) SetType(t string) {
	e.Type = t
}

func (e *VersionedEntity) GetSeqNo() int {
	return e.IfSeqNo
}

func (e *VersionedEntity) SetSeqNo(i int) {
	e.IfSeqNo = i
}

func (e *VersionedEntity) GetPrimaryTerm() int {
	return e.IfPrimaryTerm
}

func (e *VersionedEntity) SetPrimaryTerm(i int) {
	e.IfPrimaryTerm = i
}

//DataStore EntityStore type
type DataStore struct{}

// Put adds or updates an entity
func (ds *DataStore) Put(ctx Context, key Key, entity ValidEntity) error {
	if ctx.Metrics().Enabled {
		defer ctx.Metrics().Stop(ctx.Metrics().Start(fmt.Sprintf("%s EntityStore.Put", key.Kind())))
	}
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
	if ctx.Metrics().Enabled {
		defer ctx.Metrics().Stop(ctx.Metrics().Start(fmt.Sprintf("%s EntityStore.Get", key.Kind())))
	}
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
	if ctx.Metrics().Enabled {
		defer ctx.Metrics().Stop(ctx.Metrics().Start(fmt.Sprintf("%s EntityStore.Delete", key.Kind())))
	}
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
	entity.SetType(kind)
	// The internal _version change was deprecated since 7.0, so we do sanitize the version value to 0 to omit serialization
	entity.SetDatabaseVersion(0)
	data, err := json.Marshal(entity)
	if err != nil {
		return nil, err
	}
	msg := NewJSONMessage(data, map[string]int{
		"version":     entity.GetDatabaseVersion(),
		"primaryTerm": entity.GetPrimaryTerm(),
		"seqNo":       entity.GetSeqNo(),
	})
	return msg, nil
}

func (ds *DataStore) deserialize(kind string, jsonMsg JSONMessage, entity ValidEntity) error {
	// hook for looking up deserializers by kind; default json Unmarshal for now
	if err := SafeUnmarshal(jsonMsg.Bytes(), entity); err != nil {
		return err
	}
	entity.SetDatabaseVersion(jsonMsg.Version()["version"])
	entity.SetPrimaryTerm(jsonMsg.Version()["primaryTerm"])
	entity.SetSeqNo(jsonMsg.Version()["seqNo"])
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
