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

// NewStore creates a new image registry store
func NewStore() ImageRegistryStore {
	return &storeImpl{}
}

// RegistryImageStore is the database for the docker image registry
type ImageRegistryStore interface {
	// Get an image by id.  Return ErrNoSuchEntity if not found
	Get(ctx datastore.Context, id string) (*Image, error)

	// Put adds/updates an image to the registry
	Put(ctx datastore.Context, image *Image) error

	// Delete removes an image from the registry
	Delete(ctx datastore.Context, id string) error

	// GetImages returns all the images that are in the registry
	GetImages(ctx datastore.Context) ([]Image, error)

	// SearchLibraryByTag looks for repos that are registered under a library and tag
	SearchLibraryByTag(ctx datastore.Context, library, tag string) ([]Image, error)
}

type storeImpl struct {
	ds datastore.DataStore
}

// Get an image by id.  Return ErrNoSuchEntity if not found
func (s *storeImpl) Get(ctx datastore.Context, id string) (*Image, error) {
	image := &Image{}
	if err := s.ds.Get(ctx, Key(id), image); err != nil {
		return nil, err
	}
	return image, nil
}

// Put adds/updates an image to the registry
func (s *storeImpl) Put(ctx datastore.Context, image *Image) error {
	return s.ds.Put(ctx, image.key(), image)
}

// Delete removes an image from the registry
func (s *storeImpl) Delete(ctx datastore.Context, id string) error {
	return s.ds.Delete(ctx, Key(id))
}

// GetImages returns all the images that are in the registry
func (s *storeImpl) GetImages(ctx datastore.Context) ([]Image, error) {
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
func (s *storeImpl) SearchLibraryByTag(ctx datastore.Context, library, tag string) ([]Image, error) {
	if library = strings.TrimSpace(library); library == "" {
		return nil, errors.New("empty library not allowed")
	} else if tag = strings.TrimSpace(tag); tag == "" {
		return nil, errors.New("empty tag not allowed")
	}
	search := search.Search("controlplane").Type(kind).Size("50000").Filter(
		"and",
		search.Filter().Terms("Library", library),
		search.Filter().Terms("Tag", tag),
	)
	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
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
