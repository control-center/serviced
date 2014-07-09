// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package servicetemplate

import (
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"

	"fmt"
)

//NewStore creates a ResourcePool store
func NewStore() *Store {
	return &Store{}
}

//Store type for interacting with ResourcePool persistent storage
type Store struct {
	ds datastore.DataStore
}

// Put adds or updates a ServiceTemplate
func (s *Store) Put(ctx datastore.Context, st ServiceTemplate) error {

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
func (s *Store) Get(ctx datastore.Context, id string) (*ServiceTemplate, error) {
	var wrapper serviceTemplateWrapper

	if err := s.ds.Get(ctx, Key(id), &wrapper); err != nil {
		return nil, err
	}

	return FromJSON(wrapper.Data)

}

// Delete removes the a ServiceTemplate if it exists
func (s *Store) Delete(ctx datastore.Context, id string) error {
	return s.ds.Delete(ctx, Key(id))
}

// GetServiceTemplates returns all ServiceTemplates
func (s *Store) GetServiceTemplates(ctx datastore.Context) ([]*ServiceTemplate, error) {
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
