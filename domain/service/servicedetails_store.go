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

package service

import (
	"errors"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
)

// GetChildServiceDetails returns service details given parent service id
func (s *storeImpl) GetChildServiceDetails(ctx datastore.Context, parentID string) ([]ServiceDetails, error) {
	id := strings.TrimSpace(parentID)
	if id == "" {
		return nil, errors.New("empty parent service id not allowed")
	}

	searchRequest := elastic.ElasticSearchRequest{
		Pretty: false,
		Index:  "controlplane",
		Type:   "service",
		Scroll: "",
		Scan:   0,
	}

	searchRequest.Query = map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]string{"ParentServiceID": id},
		},
		"fields": []string{"ID", "Name", "Description", "PoolID", "ParentServiceID"},
		"size":   50000,
	}

	results, err := datastore.NewQuery(ctx).Execute(searchRequest)
	if err != nil {
		return nil, err
	}

	details := []ServiceDetails{}
	for results.HasNext() {
		var d ServiceDetails
		err := results.Next(&d)
		if err != nil {
			return nil, err
		}
		details = append(details, d)
	}

	return details, nil
}
