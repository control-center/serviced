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
	"errors"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain/registry"
)

var ErrImageNotFound = errors.New("registry index: image not found")

// RegistryIndex is the index for the docker registry on the server
type RegistryIndex interface {
	FindImage(image string) (*registry.Image, error)
	PushImage(image, uuid string, hash string) error
	RemoveImage(image string) error
	SearchLibraryByTag(library string, tag string) ([]registry.Image, error)
}

var _ = RegistryIndex(&RegistryIndexClient{})

// facade has the index for this docker registry
type facade interface {
	GetRegistryImage(ctx datastore.Context, image string) (*registry.Image, error)
	SetRegistryImage(ctx datastore.Context, rImage *registry.Image) error
	DeleteRegistryImage(ctx datastore.Context, image string) error
	GetRegistryImages(ctx datastore.Context) ([]registry.Image, error)
	SearchRegistryLibraryByTag(ctx datastore.Context, library, tag string) ([]registry.Image, error)
}

// RegistryIndexClient is the facade client that runs the docker registry server
type RegistryIndexClient struct {
	ctx    datastore.Context
	facade facade
}

// NewRegistryIndexClient creates a new client for the registry index.
func NewRegistryIndexClient(f facade) *RegistryIndexClient {
	return &RegistryIndexClient{
		ctx:    datastore.Get(),
		facade: f,
	}
}

// parseImage trims the registry host:port
func (client *RegistryIndexClient) parseImage(image string) (string, error) {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return "", err
	}
	if imageID.IsLatest() {
		imageID.Tag = docker.Latest
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
	image, err := client.parseImage(image)
	if err != nil {
		return nil, err
	}
	rImage, err := client.facade.GetRegistryImage(client.ctx, image)
	if datastore.IsErrNoSuchEntity(err) {
		return nil, ErrImageNotFound
	}
	return rImage, err
}

// PushImage implements RegistryIndex
func (client *RegistryIndexClient) PushImage(image, uuid string, hash string) error {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return err
	}
	if imageID.IsLatest() {
		imageID.Tag = docker.Latest
	}

	rImage := &registry.Image{
		Library: imageID.User,
		Repo:    imageID.Repo,
		Tag:     imageID.Tag,
		UUID:    uuid,
		Hash:    hash,
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
