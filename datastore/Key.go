// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

// Key is a unique identifier for an entity. A key is composed of a kind or type of entity and the id of the entity.
type Key interface {
	// Kind is the type of the entity
	Kind() string
	// ID is the id of the entity
	ID() string
}

//NewKey returns an initialized Key
func NewKey(kind string, id string) Key {
	return &key{id, kind}
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
