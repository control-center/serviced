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

package hostkey

import (
	"github.com/control-center/serviced/datastore"
)

// NewStore creates a new RSA Key store
func NewStore() Store {
	return &storeImpl{}
}

// Store is the database for the RSA keys per host
type Store interface {
	// Get an RSA Key by host id.  Return ErrNoSuchEntity if not found
	Get(ctx datastore.Context, id string) (*HostKey, error)

	// Put adds/updates an RSA Key to the registry
	Put(ctx datastore.Context, id string, val *HostKey) error

	// Delete removes an RSA Key from the registry
	Delete(ctx datastore.Context, id string) error
}

type storeImpl struct {
	ds datastore.DataStore
}

// Get an RSA Key by host id.  Return ErrNoSuchEntity if not found
func (s *storeImpl) Get(ctx datastore.Context, id string) (*HostKey, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("HostKeyStore.Get"))
	val := &HostKey{}
	if err := s.ds.Get(ctx, Key(id), val); err != nil {
		return nil, err
	}
	return val, nil
}

// Put adds/updates an RSA Key to the registry
func (s *storeImpl) Put(ctx datastore.Context, id string, val *HostKey) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("HostKeyStore.Put"))
	return s.ds.Put(ctx, Key(id), val)
}

// Delete removes an RSA Key from the registry
func (s *storeImpl) Delete(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("HostKeyStore.Delete"))
	return s.ds.Delete(ctx, Key(id))
}
