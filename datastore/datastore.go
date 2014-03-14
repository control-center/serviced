// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"reflect"
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
type Entity struct {
	Key     Key
	Payload interface{}
}

func NewEntity(key Key, payload interface{}) *Entity {
	return &Entity{key, payload}
}
func New() DataStore {
	return &dataStore{}
}

type DataStore interface {
	Put(ctx Context, key Key, data interface{}) error

	Get(ctx Context, id string, data interface{}) error

	Delete(ctx Context, key Key) error

	Query(ctx Context) Query
}

type dataStore struct{}

func (ds *dataStore) Put(ctx Context, key Key, data interface{}) error {

	entity := NewEntity(key, data)
	return ctx.Driver().Put(entity)
}

func (ds *dataStore) Get(ctx Context, id string, data interface{}) error {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("data parm not a pointer")
	}
	if entity, err := ctx.Driver().Get(id); err != nil {
		return err
	} else {
		payload := reflect.ValueOf(entity.Payload)
		if payload.Kind() == reflect.Ptr {
			payload = payload.Elem()
		}
		v.Elem().Set(payload)
	}
	return nil
}

func (ds *dataStore) Delete(ctx Context, key Key) error {
	return ctx.Driver().Delete(key)
}

func (ds *dataStore) Query(ctx Context) Query{
	return newQuery(ctx)
}
