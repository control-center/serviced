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

package layer

import (
	"archive/tar"
)

const (
	// Maximum layer count is set and enforced by docker:
	//  https://github.com/docker/docker/blob/101c749b6533ab309eccea6b6c6173a0c25f787d/image/image.go#L308
	MAX_LAYER_COUNT int = 127 - 2
	WARN_LAYER_COUNT int = MAX_LAYER_COUNT - 16
)


type byName []*tar.Header

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type layersT []string

func (layers layersT) contains(layer string) bool {
	for i := range layers {
		if layers[i] == layer {
			return true
		}
	}
	return false
}

// getLayers returns a list of layer IDs starting from name down to the base layer
func getLayers(client DockerClient, name string) (layers layersT, err error) {

	layers = make(layersT, 0)
	for {
		image, err := client.InspectImage(name)
		if err != nil {
			return layers, err
		}
		layers = append(layers, image.ID)
		if image.Parent == "" {
			break
		}
		name = image.Parent
	}
	return layers, err
}
