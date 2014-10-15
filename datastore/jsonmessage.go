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

package datastore

import (
	"encoding/json"
)

// JSONMessage Represents a enity as JSON
type JSONMessage interface {
	// Bytes return the JSON bytes of an entity
	Bytes() json.RawMessage
	Version() int
}

// NewJSONMessage creates a JSONMessage using the provided bytes. The bytes should represent valid JSON
func NewJSONMessage(data []byte, version int) JSONMessage {
	return &jsonMessage{data, version}
}

type jsonMessage struct {
	data    json.RawMessage
	version int
}

func (m *jsonMessage) Bytes() json.RawMessage {
	return m.data
}

func (m *jsonMessage) Version() int {
	return m.version
}

// MarshalJSON returns *m as the JSON encoding of m.
func (m *jsonMessage) MarshalJSON() ([]byte, error) {
	return m.data.MarshalJSON()
}

// UnmarshalJSON sets *m to a copy of data.
func (m *jsonMessage) UnmarshalJSON(data []byte) error {
	return m.data.UnmarshalJSON(data)
}
