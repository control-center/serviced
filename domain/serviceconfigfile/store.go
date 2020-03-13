// Copyright 2014 The Serviced Authors.
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

package serviceconfigfile

import (
	"strconv"

	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/search"
)

// Store is an interface for accessing service configuration files data.
type Store interface {
	datastore.Store

	GetConfigFiles(ctx datastore.Context, tenantID string, svcPath string) ([]*SvcConfigFile, error)
	GetConfigFile(ctx datastore.Context, tenantID, svcPath, filename string) (*SvcConfigFile, error)
}

type store struct{}

var kind = "svcconfigfile"

// NewStore returns a new object that implements the Store interface.
func NewStore() Store {
	return &store{}
}

// Put adds or updates an entity
func (s *store) Put(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	return datastore.Put(ctx, key, entity)
}

// Get an entity. Return ErrNoSuchEntity if nothing found for the key.
func (s *store) Get(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	return datastore.Get(ctx, key, entity)
}

// Delete removes the entity
func (s *store) Delete(ctx datastore.Context, key datastore.Key) error {
	return datastore.Delete(ctx, key)
}

// GetConfigFiles returns all Configuration Files in tenant service that have the given service
// path. The service path is a "/" delimited string of the service name hierarchy,
// i.e /Zenoss.Core/Zproxy
func (s *store) GetConfigFiles(ctx datastore.Context, tenantID string, svcPath string) ([]*SvcConfigFile, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceConfigFileStore.GetConfigFiles"))

	size := 20
	from := 0
	var confs []*SvcConfigFile

	for {
		search := search.Search("controlplane").Type(kind).Filter(
			"and",
			search.Filter().Terms("ServiceTenantID", tenantID),
			search.Filter().Terms("ServicePath", svcPath),
		).From(strconv.Itoa(from)).Size(strconv.Itoa(size))

		q := datastore.NewQuery(ctx)

		results, err := q.Execute(search)
		if err != nil {
			return nil, err
		}

		conv, err := convert(results)
		if err != nil {
			return nil, err
		} else if len(conv) == 0 {
			return confs, nil
		}

		from += size
		confs = append(confs, conv...)
	}
}

// GetConfigFile stuff
func (s *store) GetConfigFile(ctx datastore.Context, tenantID, svcPath, filename string) (*SvcConfigFile, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceConfigFileStore.GetConfigFile"))
	search := search.Search("controlplane").Type(kind).Filter(
		"and",
		search.Filter().Terms("ServiceTenantID", tenantID),
		search.Filter().Terms("ServicePath", svcPath),
		search.Filter().Terms("ConfFile.Filename", filename),
	)

	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}

	if results.Len() > 0 {
		matches, err := convert(results)
		if err != nil {
			return nil, err
		}
		return matches[0], nil
	}

	return nil, nil
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
