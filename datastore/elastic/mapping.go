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

package elastic

import (
	"github.com/zenoss/glog"

	"encoding/json"
	"fmt"
)

type Mapping struct {
	Name    string
	Entries map[string]interface{}
}

// MarshalJSON returns *m as the JSON encoding of m.
func (m Mapping) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{m.Name: m.Entries}
	return json.Marshal(data)
}

// UnmarshalJSON sets *m to a copy of data.
func (m *Mapping) UnmarshalJSON(data []byte) error {
	var rawmapping map[string]map[string]interface{}
	if err := json.Unmarshal(data, &rawmapping); err != nil {
		return err
	}
	if mapping, err := newMapping(rawmapping); err != nil {
		return err
	} else {
		m.Name = mapping.Name
		m.Entries = mapping.Entries
	}
	return nil
}

func newMapping(rawmapping map[string]map[string]interface{}) (Mapping, error) {
	if len(rawmapping) > 1 {
		return Mapping{}, fmt.Errorf("unexpected number of top level entries: %v", len(rawmapping))
	}
	mapping := Mapping{}
	for key, val := range rawmapping {
		mapping.Name = key
		mapping.Entries = val
	}
	return mapping, nil
}

func NewMapping(mapping string) (Mapping, error) {
	bytes := []byte(mapping)
	var result Mapping
	if err := json.Unmarshal(bytes, &result); err != nil {
		glog.Errorf("error creating mapping: %v", err)
		return result, err
	}
	return result, nil
}
