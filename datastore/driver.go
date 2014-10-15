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

// Driver is the interface for a driver to a datastore
type Driver interface {
	// GetConnection returns a connection to the datastore.
	GetConnection() (Connection, error)
}

// Connection is the interface for interacting with a datastore
type Connection interface {

	// Put adds or updates an entity in the datastore using the Key.
	Put(key Key, data JSONMessage) error

	// Get returns an entity from the datastore. Can return ErrNoSuchEntity if the entity does not exists
	Get(key Key) (JSONMessage, error)

	// Delete deletes an entity associated with the key
	Delete(key Key) error

	// Query evaluates the query and returns a list of entities form the datastore
	Query(query interface{}) ([]JSONMessage, error)
}
