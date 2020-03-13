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
	"fmt"
)

// Store interface for storing and retrieving data types from a datastore.
type Store interface {
	Put(ctx Context, key Key, entity ValidEntity) error
	Get(ctx Context, key Key, entity ValidEntity) error
	Delete(ctx Context, key Key) error
}

// Put adds or updates an entity
func Put(ctx Context, key Key, entity ValidEntity) error {
	if ctx.Metrics().Enabled {
		defer ctx.Metrics().Stop(ctx.Metrics().Start(fmt.Sprintf("%s Store.Put", key.Kind())))
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

	// jsonMsg, err := ds.serialize(key.Kind(), entity)
	jsonMsg, err := serialize(key.Kind(), entity)
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
func Get(ctx Context, key Key, entity ValidEntity) error {
	if ctx.Metrics().Enabled {
		defer ctx.Metrics().Stop(ctx.Metrics().Start(fmt.Sprintf("%s Store.Get", key.Kind())))
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
	// err = ds.deserialize(key.Kind(), jsonMsg, entity)
	err = deserialize(key.Kind(), jsonMsg, entity)
	return err

}

// Delete removes the entity
func Delete(ctx Context, key Key) error {
	if ctx.Metrics().Enabled {
		defer ctx.Metrics().Stop(ctx.Metrics().Start(fmt.Sprintf("%s Store.Delete", key.Kind())))
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

func validKey(k Key) error {
	if k.ID() == "" {
		return ErrEmptyKindID
	}
	if k.Kind() == "" {
		return ErrEmptyKind
	}
	return nil
}
