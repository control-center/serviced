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
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"strconv"
)

//NewStore creates a Service Config File store
func NewStore() Store {
	return &storeImpl{}
}

//Store type for interacting with Service Config File persistent storage
type Store interface {
	datastore.EntityStore

	GetConfigFiles(ctx datastore.Context, tenantID string, svcPath string) ([]*SvcConfigFile, error)
	GetConfigFile(ctx datastore.Context, tenantID, svcPath, filename string) (*SvcConfigFile, error)
}

type storeImpl struct {
	datastore.DataStore
}

//GetConfigFiles returns all Configuration Files in tenant service that have the given service path. The service path
//is a "/" delimited string of the service name hierarchy, i.e /Zenoss.Core/Zproxy
func (s *storeImpl) GetConfigFiles(ctx datastore.Context, tenantID string, svcPath string) ([]*SvcConfigFile, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceConfigFileStore.GetConfigFiles"))

	size := 20
	from := 0
	var confs []*SvcConfigFile

	for {
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []map[string]interface{}{
						{"term": map[string]string{"type": kind}},
						{"term": map[string]string{"ServiceTenantID": tenantID}},
						{"term": map[string]string{"ServicePath": svcPath}},
					},
				},
			},
			"from": from,
			"size": strconv.Itoa(size),
		}

		search, err := elastic.BuildSearchRequest(query, "controlplane")
		if err != nil {
			return nil, err
		}
		q := datastore.NewQuery(ctx)

		results, err := q.Execute(search)
		if err != nil {
			return nil, err
		}

		conv, err := convert(results)
		confs = append(confs, conv...)
		if err != nil {
			return nil, err
		} else if len(conv) == 0 || len(conv) < size {
			return confs, nil
		}

		from += size
	}
}

func (s *storeImpl) GetConfigFile(ctx datastore.Context, tenantID, svcPath, filename string) (*SvcConfigFile, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceConfigFileStore.GetConfigFile"))
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"ServiceTenantID": tenantID}},
					{"term": map[string]string{"ServicePath": svcPath}},
					{"term": map[string]string{"ConfFile.Filename": filename}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}

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

var (
	kind = "svcconfigfile"
)
