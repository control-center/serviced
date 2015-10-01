// Copyright 2015 The Serviced Authors.
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

package docker

import (
	"io"

	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
)

const (
	Latest    = "latest"
	MaxLayers = 127 - 2
)

// Docker is the docker client for the dfs
type Docker interface {
	FindImage(image string) (*dockerclient.Image, error)
	SaveImages(images []string, writer io.Writer) error
	LoadImage(reader io.Reader) error
	PushImage(image string) error
	PullImage(image string) error
	TagImage(oldImage, newImage string)
	RemoveImage(image string) error
	ImageHistory(image string) ([]dockerclient.ImageHistory, error)
	FindContainer(ctr string) (*dockerclient.Container, error)
	CommitContainer(ctr, image string) (*dockerclient.Image, error)
}

// Registry is the docker registry client for the dfs
type Registry interface {
	GetImage(image string) *registry.Image
	PushImage(image, uuid string) (string, error)
	PullImage(image string) (string, error)
	DeleteImage(image string) error
	SearchLibraryByTag(library, tag string) ([]registry.Image, error)
}
