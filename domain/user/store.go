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

package user

import (
	"github.com/control-center/serviced/datastore"

	"strings"
)

// Store is an interface for accessing user data.
type Store interface {
	datastore.Store
}

type store struct{}

// NewStore returns a new object that implements the Store interface.
func NewStore() Store {
	return &store{}
}

// Put adds or updates an entity
func (s *store) Put(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	return datastore.Put(ctx, key, entity)
}

// Get an entity. Return ErrNoSuchEntity if nothing found for the key.
func (s *store) Get(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	return datastore.Get(ctx, key, entity)
}

// Delete removes the entity
func (s *store) Delete(ctx datastore.Context, key datastore.Key) error {
	return datastore.Delete(ctx, key)
}

//Key creates a Key suitable for getting, putting and deleting Users
func Key(id string) datastore.Key {
	id = strings.TrimSpace(id)
	return datastore.NewKey(kind, id)
}

var kind = "user"
