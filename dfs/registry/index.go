// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/registry"
)

// RegistryIndex is the index for the docker registry on the server
type RegistryIndex interface {
	FindImage(image string) (*registry.Image, error)
	PushImage(image, uuid string) error
	RemoveImage(image string) error
	SearchLibraryByTag(library string, tag string) ([]registry.Image, error)
}

// facade has the index for this docker registry
type facade interface {
	GetRegistryImage(ctx datastore.Context, image string) (*registry.Image, error)
	SetRegistryImage(ctx datastore.Context, rImage *registry.Image) error
	DeleteRegistryImage(ctx datastore.Context, image string) error
	SearchRegistryLibraryByTag(ctx datastore.Context, library, tag string) ([]registry.Image, error)
}

// RegistryIndexClient is the facade client that runs the docker registry server
type RegistryIndexClient struct {
	ctx    datastore.Context
	facade facade
}

// parseImage trims the registry host:port
func (client *RegistryIndexClient) parseImage(image string) (string, error) {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return "", err
	}
	if imageID.IsLatest() {
		imageID.Tag = docker.DockerLatest
	}
	image = commons.ImageID{
		User: imageID.User,
		Repo: imageID.Repo,
		Tag:  imageID.Tag,
	}.String()
	return image, nil
}

// FindImage implements RegistryIndex
func (client *RegistryIndexClient) FindImage(image string) (*registry.Image, error) {
	var err error
	if image, err = client.parseImage(image); err != nil {
		return nil, err
	}
	return client.facade.GetRegistryImage(client.ctx, image)
}

// PushImage implements RegistryIndex
func (client *RegistryIndexClient) PushImage(image, uuid string) error {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return err
	}
	if imageID.IsLatest() {
		imageID.Tag = docker.DockerLatest
	}
	rImage := &registry.Image{
		Library: imageID.User,
		Repo:    imageID.Repo,
		Tag:     imageID.Tag,
		UUID:    uuid,
	}
	return client.facade.SetRegistryImage(client.ctx, rImage)
}

// RemoveImage implements RegistryIndex
func (client *RegistryIndexClient) RemoveImage(image string) error {
	var err error
	if image, err = client.parseImage(image); err != nil {
		return err
	}
	return client.facade.DeleteRegistryImage(client.ctx, image)
}

// SearchLibraryByTag implements RegistryIndex
func (client *RegistryIndexClient) SearchLibraryByTag(library, tag string) ([]registry.Image, error) {
	return client.facade.SearchRegistryLibraryByTag(client.ctx, library, tag)
}
