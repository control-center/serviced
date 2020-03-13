// Copyright 2015 The Serviced Authors.
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

package registry

import (
	"errors"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/search"
)

// Store is an interface for accessing registry data.
type Store interface {
	Get(ctx datastore.Context, id string) (*Image, error)
	Put(ctx datastore.Context, image *Image) error
	Delete(ctx datastore.Context, id string) error
	GetImages(ctx datastore.Context) ([]Image, error)
	SearchLibraryByTag(ctx datastore.Context, library, tag string) ([]Image, error)
}

type store struct{}

// NewStore returns a new object that implements the Store interface.
func NewStore() Store {
	return &store{}
}

// Get an image by id.  Return ErrNoSuchEntity if not found
func (s *store) Get(ctx datastore.Context, id string) (*Image, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ImageRegistryStore.Get"))
	image := &Image{}
	if err := datastore.Get(ctx, Key(id), image); err != nil {
		return nil, err
	}
	return image, nil
}

// Put adds/updates an image to the registry
func (s *store) Put(ctx datastore.Context, image *Image) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ImageRegistryStore.Put"))
	return datastore.Put(ctx, image.key(), image)
}

// Delete removes an image from the registry
func (s *store) Delete(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ImageRegistryStore.Delete"))
	return datastore.Delete(ctx, Key(id))
}

// GetImages returns all the images that are in the registry
func (s *store) GetImages(ctx datastore.Context) ([]Image, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ImageRegistryStore.GetImages"))
	query := search.Query().Search("_exists_:Library")
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// SearchLibraryByTag looks for repos that are registered under a library and tag
func (s *store) SearchLibraryByTag(ctx datastore.Context, library, tag string) ([]Image, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ImageRegistryStore.SearchLibraryByTag"))
	if library = strings.TrimSpace(library); library == "" {
		return nil, errors.New("empty library not allowed")
	} else if tag = strings.TrimSpace(tag); tag == "" {
		return nil, errors.New("empty tag not allowed")
	}
	searchParams := search.Search("controlplane").Type(kind).Size("50000").Filter(
		"and",
		search.Filter().Terms("Library", library),
		search.Filter().Terms("Tag", tag),
	)
	q := datastore.NewQuery(ctx)
	results, err := q.Execute(searchParams)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func convert(results datastore.Results) ([]Image, error) {
	images := make([]Image, results.Len())
	for idx := range images {
		if err := results.Get(idx, &images[idx]); err != nil {
			return nil, err
		}
	}
	return images, nil
}
