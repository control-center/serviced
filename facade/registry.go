// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/registry"
)

// GetRegistryImage returns information about an image that is stored in the
// docker registry index.
// e.g. GetRegistryImage(ctx, "library/reponame:tagname")
func (f *Facade) GetRegistryImage(ctx datastore.Context, image string) (*registry.Image, error) {
	rImage, err := f.registryStore.Get(ctx, image)
	if err != nil {
		return nil, err
	}
	return rImage, nil
}

// SetRegistryImage creates/updates an image in the docker registry index.
func (f *Facade) SetRegistryImage(ctx datastore.Context, rImage *registry.Image) error {
	if err := f.registryStore.Put(ctx, rImage); err != nil {
		return err
	}
	// TODO: update zookeeper
	return nil
}

// DeleteRegistryImage removes an image from the docker registry index.
// e.g. DeleteRegistryImage(ctx, "library/reponame:tagname")
func (f *Facade) DeleteRegistryImage(ctx datastore.Context, image string) error {
	if err := f.registryStore.Delete(ctx, image); err != nil {
		return err
	}
	// TODO: update zookeeper
	return nil
}

// SearchRegistryLibrary searches the docker registry index for images at a
// particular library and tag.
// e.g. library/reponame:tagname => SearchRegistryLibrary("library", "tagname")
func (f *Facade) SearchRegistryLibraryByTag(ctx datastore.Context, library, tagname string) ([]registry.Image, error) {
	rImages, err := f.registryStore.SearchLibraryByTag(ctx, library, tagname)
	if err != nil {
		return nil, err
	}
	return rImages, nil
}
