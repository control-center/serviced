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

func (k *key) String() string {
	return fmt.Sprintf("Key: %s - %s", k.kind, k.id)
}
