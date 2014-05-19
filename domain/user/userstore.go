// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package user

import (
	"github.com/zenoss/serviced/datastore"

	"strings"
)

//NewStore creates a UserStore
func NewStore() *UserStore {
	return &UserStore{}
}

//UserStore type for interacting with User persistent storage
type UserStore struct {
	datastore.DataStore
}

//Key creates a Key suitable for getting, putting and deleting Users
func Key(id string) datastore.Key {
	id = strings.TrimSpace(id)
	return datastore.NewKey(kind, id)
}

var kind = "user"
