// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/serviced/datastore"
)

type HostStore interface {
	Put(ctx datastore.Context, host *Host) error

	Get(id string) (*Host, error)

	Delete(id string) error
}

func NewStore() HostStore {
	return &hostStore{datastore.New()}
}

type hostStore struct {
	ds datastore.DataStore
}

func (hs *hostStore) Put(ctx datastore.Context, host *Host) error {
	return nil
}

func (hs *hostStore) Get(id string) (*Host, error) {
	return nil, nil
}

func (hs *hostStore) Delete(id string) error {
	return nil
}
