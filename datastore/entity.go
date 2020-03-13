// Copyright 2017 The Serviced Authors.
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

// Entity is the interface for encapsulating an object's name
// and type for use with modules such as a logger.
type Entity interface {
	GetID() string
	GetType() string
}

//ValidEntity interface for entities that can be stored in the Store
type ValidEntity interface {
	ValidEntity() error
	GetDatabaseVersion() int
	SetDatabaseVersion(int)
}

// VersionedEntity contains the database version.
type VersionedEntity struct {
	DatabaseVersion int `json:",omitempty"`
}

// GetDatabaseVersion returns the database version
func (e *VersionedEntity) GetDatabaseVersion() int {
	return e.DatabaseVersion
}

// SetDatabaseVersion sets the database version
func (e *VersionedEntity) SetDatabaseVersion(i int) {
	e.DatabaseVersion = i
}
