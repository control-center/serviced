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

// Store is an interface for accessing host key data.
type Store interface {
	Get(ctx datastore.Context, id string) (*RSAKey, error)
	Put(ctx datastore.Context, id string, val *RSAKey) error
	Delete(ctx datastore.Context, id string) error
}

type store struct{}

// NewStore returns a new object that implements the Store interface.
func NewStore() Store {
	return &store{}
}

// Get an RSA Key by host id.  Return ErrNoSuchEntity if not found
func (s *store) Get(ctx datastore.Context, id string) (*RSAKey, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("HostKeyStore.Get"))
	val := &RSAKey{}
	if err := datastore.Get(ctx, Key(id), val); err != nil {
		return nil, err
	}
	return val, nil
}

// Put adds/updates an RSA Key to the registry
func (s *store) Put(ctx datastore.Context, id string, val *RSAKey) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("HostKeyStore.Put"))
	return datastore.Put(ctx, Key(id), val)
}

// Delete removes an RSA Key from the registry
func (s *store) Delete(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("HostKeyStore.Delete"))
	return datastore.Delete(ctx, Key(id))
}
