// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	"github.com/zenoss/serviced/datastore"
)

type elasticConnection struct {
}

func (ec *elasticConnection) Put(datastore.Key, datastore.JsonMessage) error {
	return nil
}

func (ec *elasticConnection) Get(key datastore.Key) (datastore.JsonMessage, error) {
	return nil, nil
}

func (ec *elasticConnection) Query(query datastore.Query) ([]datastore.JsonMessage, error) {
	return make([]datastore.JsonMessage, 0), nil
}

func (ec *elasticConnection) Delete(key datastore.Key) error {
	return nil
}
