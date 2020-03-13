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

package servicetemplate

import (
	"fmt"

	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/search"
)

// Store is an interface for accessing service template data.
type Store interface {
	Put(ctx datastore.Context, st ServiceTemplate) error
	Get(ctx datastore.Context, id string) (*ServiceTemplate, error)
	Delete(ctx datastore.Context, id string) error
	GetServiceTemplates(ctx datastore.Context) ([]*ServiceTemplate, error)
}

type store struct{}

// NewStore returns a new object that implements the Store interface.
func NewStore() Store {
	return &store{}
}

var kind = "servicetemplatewrapper"

// Put adds or updates a ServiceTemplate
func (s *store) Put(ctx datastore.Context, st ServiceTemplate) error {
	if err := st.ValidEntity(); err != nil {
		return fmt.Errorf("error validating template: %v", err)
	}
	wrapper, err := newWrapper(st)
	if err != nil {
		return err
	}
	return datastore.Put(ctx, Key(wrapper.ID), wrapper)
}

// Get a ServiceTemplate by id. Return ErrNoSuchEntity if not found
func (s *store) Get(ctx datastore.Context, id string) (*ServiceTemplate, error) {
	var wrapper serviceTemplateWrapper
	if err := datastore.Get(ctx, Key(id), &wrapper); err != nil {
		return nil, err
	}
	return FromJSON(wrapper.Data)
}

// Delete removes the a ServiceTemplate if it exists
func (s *store) Delete(ctx datastore.Context, id string) error {
	return datastore.Delete(ctx, Key(id))
}

// GetServiceTemplates returns all ServiceTemplates
func (s *store) GetServiceTemplates(ctx datastore.Context) ([]*ServiceTemplate, error) {
	q := datastore.NewQuery(ctx)
	query := search.Query().Search("_exists_:ID")
	search := search.Search("controlplane").Type(kind).Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

//Key creates a Key suitable for getting, putting and deleting ResourcePools
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

func convert(results datastore.Results) ([]*ServiceTemplate, error) {
	templates := make([]*ServiceTemplate, results.Len())
	for idx := range templates {
		var stw serviceTemplateWrapper
		if err := results.Get(idx, &stw); err != nil {
			return []*ServiceTemplate{}, err
		}
		st, err := FromJSON(stw.Data)
		if err != nil {
			return []*ServiceTemplate{}, err
		}
		templates[idx] = st
	}
	return templates, nil
}
