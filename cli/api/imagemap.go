// Copyright 2016 The Serviced Authors.
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

package api

import (
	"fmt"
	"strings"
)

// ImageMap parses docker image data
type ImageMap map[string]string

// Set converts a docker image mapping into an ImageMap
func (m *ImageMap) Set(value string) error {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return fmt.Errorf("bad format")
	}

	(*m)[parts[0]] = parts[1]
	return nil
}

func (m *ImageMap) String() string {
	var mapping []string
	for k, v := range *m {
		mapping = append(mapping, k+","+v)
	}

	return strings.Join(mapping, " ")
}
