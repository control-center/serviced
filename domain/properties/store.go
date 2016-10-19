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

package properties

import (
	"github.com/control-center/serviced/datastore"
)

const propsID = "cc_properties"

// NewStore creates a new RSA Key store
func NewStore() Store {
	return &storeImpl{}
}

// Store is used to access the stored properties
type Store interface {
	Get(ctx datastore.Context) (*StoredProperties, error)

	// Put adds/updates an RSA Key to the registry
	Put(ctx datastore.Context, properties *StoredProperties) error
}

type storeImpl struct {
	ds datastore.DataStore
}

// Get the single instance of StoredProperties.  Return ErrNoSuchEntity if not found
func (s *storeImpl) Get(ctx datastore.Context) (*StoredProperties, error) {
	val := &StoredProperties{}
	if err := s.ds.Get(ctx, Key(propsID), val); err != nil {
		return nil, err
	}
	return val, nil
}

// Put adds/updates an RSA Key to the registry
func (s *storeImpl) Put(ctx datastore.Context, properties *StoredProperties) error {
	return s.ds.Put(ctx, Key(propsID), properties)
}
