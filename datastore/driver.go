// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

// Drive is the interface for a driver to a datastore
type Driver interface {
	// GetConnection returns a connection to the datastore.
	GetConnection() (Connection, error)
}

// Connection is the interface for interacting with a datastore
type Connection interface {

	// Put adds or updates an entity in the datastore using the Key.
	Put(key Key, data JsonMessage) error

	// Get returns an entity from the datastore. Can return ErrNoSuchEntity if the entity does not exists
	Get(key Key) (JsonMessage, error)

	// Delete deletes an entity associated with the key
	Delete(key Key) error

	// Query evaluates the query and returns a list of entities form the datastore
	Query(query interface{}) ([]JsonMessage, error)
}
