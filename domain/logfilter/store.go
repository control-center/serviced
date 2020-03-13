// Copyright 2017 The Serviced Authors.
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

package logfilter

import (
	"fmt"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/search"
)

// Store defines the interface to accessing LogFilter data.
type Store interface {
	Get(ctx datastore.Context, name, version string) (*LogFilter, error)
	Put(ctx datastore.Context, lf *LogFilter) error
	Delete(ctx datastore.Context, name, version string) error
	GetLogFilters(ctx datastore.Context) ([]*LogFilter, error)
}

type store struct{}

// NewStore returns a new object that implements the Store interface.
func NewStore() Store {
	return &store{}
}

// Get a LogFilter by id.  Return ErrNoSuchEntity if not found
func (s *store) Get(ctx datastore.Context, name, version string) (*LogFilter, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("LogFilterStore.Get"))
	val := &LogFilter{}
	if err := datastore.Get(ctx, Key(name, version), val); err != nil {
		return nil, err
	}
	return val, nil
}

// Put adds/updates a LogFilter
func (s *store) Put(ctx datastore.Context, lf *LogFilter) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("LogFilterStore.Put"))
	return datastore.Put(ctx, Key(lf.Name, lf.Version), lf)
}

// Delete removes a LogFilter
func (s *store) Delete(ctx datastore.Context, name, version string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("LogFilterStore.Delete"))
	return datastore.Delete(ctx, Key(name, version))
}

// GetLogFilters returns all LogFilters
func (s *store) GetLogFilters(ctx datastore.Context) ([]*LogFilter, error) {
	q := datastore.NewQuery(ctx)
	query := search.Query().Search("_exists_:Name")
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

//Key creates a Key suitable for getting, putting and deleting LogFilters
func Key(name, version string) datastore.Key {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	return datastore.NewKey(kind, buildID(name, version))
}

func buildID(name, version string) string {
	return fmt.Sprintf("%s-%s", name, version)
}

func convert(results datastore.Results) ([]*LogFilter, error) {
	filters := make([]*LogFilter, results.Len())
	for idx := range filters {
		filter := LogFilter{}
		if err := results.Get(idx, &filter); err != nil {
			return []*LogFilter{}, err
		}
		filters[idx] = &filter
	}
	return filters, nil
}
