// Copyright 2016 The Serviced Authors.
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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package properties

import (
	"github.com/control-center/serviced/datastore"
)

// CCVERSION is the key used to access the version property
const CCVERSION = "cc.version"

// New create a new StoredProperties
func New() *StoredProperties {
	return &StoredProperties{Props: make(map[string]string)}
}

// StoredProperties entity containing a properties map
type StoredProperties struct {
	Props map[string]string
	datastore.VersionedEntity
}

// CCVersion returns the CC version property
func (s *StoredProperties) CCVersion() (string, bool) {
	val, ok := s.Props[CCVERSION]
	return val, ok
}

// SetCCVersion sets the CC version property
func (s *StoredProperties) SetCCVersion(version string) {
	s.Props[CCVERSION] = version
}
