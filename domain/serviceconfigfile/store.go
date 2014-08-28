// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package serviceconfigfile

import (
	"github.com/zenoss/elastigo/search"
	"github.com/control-center/serviced/datastore"
)

//NewStore creates a Service  store
func NewStore() *Store {
	return &Store{}
}

//Store type for interacting with Service persistent storage
type Store struct {
	datastore.DataStore
}

//GetConfigFiles returns all Configuration Files in tenant service that have the given service path. The service path
//is a "/" delimited string of the service name hierarchy, i.e /Zenoss.Core/Zproxy
func (s *Store) GetConfigFiles(ctx datastore.Context, tenantID string, svcPath string) ([]*SvcConfigFile, error) {
	search := search.Search("controlplane").Type(kind).Filter(
		"and",
		search.Filter().Terms("ServiceTenantID", tenantID),
		search.Filter().Terms("ServicePath", svcPath),
	)

	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func convert(results datastore.Results) ([]*SvcConfigFile, error) {
	result := make([]*SvcConfigFile, results.Len())
	for idx := range result {
		var cf SvcConfigFile
		err := results.Get(idx, &cf)
		if err != nil {
			return nil, err
		}
		result[idx] = &cf
	}
	return result, nil
}

//Key creates a Key suitable for getting, putting and deleting SvcConfigFile
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

var (
	kind = "svcconfigfile"
)
