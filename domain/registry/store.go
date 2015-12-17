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
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/zenoss/elastigo/search"
)

// NewStore creates a new image registry store
func NewStore() *ImageRegistryStore {
	return &ImageRegistryStore{}
}

// RegistryImageStore is the database for the docker image registry
type ImageRegistryStore struct {
	ds datastore.DataStore
}

// Get an image by id.  Return ErrNoSuchEntity if not found
func (s *ImageRegistryStore) Get(ctx datastore.Context, id string) (*Image, error) {
	image := &Image{}
	if err := s.ds.Get(ctx, Key(id), image); err != nil {
		return nil, err
	}
	return image, nil
}

// Put adds/updates an image to the registry
func (s *ImageRegistryStore) Put(ctx datastore.Context, image *Image) error {
	return s.ds.Put(ctx, image.key(), image)
}

// Delete removes an image from the registry
func (s *ImageRegistryStore) Delete(ctx datastore.Context, id string) error {
	return s.ds.Delete(ctx, Key(id))
}

// GetImages returns all the images that are in the registry
func (s *ImageRegistryStore) GetImages(ctx datastore.Context) ([]Image, error) {
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
func (s *ImageRegistryStore) SearchLibraryByTag(ctx datastore.Context, library, tag string) ([]Image, error) {
	if library = strings.TrimSpace(library); library == "" {
		return nil, domain.EmptyFieldNotAllowed("library")
	} else if tag = strings.TrimSpace(tag); tag == "" {
		return nil, domain.EmptyFieldNotAllowed("tag")
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
