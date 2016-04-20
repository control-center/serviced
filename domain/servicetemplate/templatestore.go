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
	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/glog"

	"fmt"
)

//NewStore creates a ResourcePool store
func NewStore() Store {
	return &storeImpl{}
}

// Store type for interacting with ResourcePool persistent storage
type Store interface {
	// Get a ServiceTemplate by id. Return ErrNoSuchEntity if not found
	Get(ctx datastore.Context, id string) (*ServiceTemplate, error)

	// Put adds or updates a ServiceTemplate
	Put(ctx datastore.Context, st ServiceTemplate) error

	// Delete removes the a ServiceTemplate if it exists
	Delete(ctx datastore.Context, id string) error

	// GetServiceTemplates returns all ServiceTemplates
	GetServiceTemplates(ctx datastore.Context) ([]*ServiceTemplate, error)
}

type storeImpl struct {
	ds datastore.DataStore
}

// Put adds or updates a ServiceTemplate
func (s *storeImpl) Put(ctx datastore.Context, st ServiceTemplate) error {
	if err := st.ValidEntity(); err != nil {
		return fmt.Errorf("error validating template: %v", err)
	}
	wrapper, err := newWrapper(st)
	if err != nil {
		return err
	}
	return s.ds.Put(ctx, Key(wrapper.ID), wrapper)
}

// Get a ServiceTemplate by id. Return ErrNoSuchEntity if not found
func (s *storeImpl) Get(ctx datastore.Context, id string) (*ServiceTemplate, error) {
	var wrapper serviceTemplateWrapper

	if err := s.ds.Get(ctx, Key(id), &wrapper); err != nil {
		return nil, err
	}

	return FromJSON(wrapper.Data)

}

// Delete removes the a ServiceTemplate if it exists
func (s *storeImpl) Delete(ctx datastore.Context, id string) error {
	return s.ds.Delete(ctx, Key(id))
}

// GetServiceTemplates returns all ServiceTemplates
func (s *storeImpl) GetServiceTemplates(ctx datastore.Context) ([]*ServiceTemplate, error) {
	glog.V(3).Infof("Store.GetServiceTemplates")
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

var kind = "servicetemplatewrapper"
