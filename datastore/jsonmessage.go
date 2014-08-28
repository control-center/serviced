// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
)

// JSONMessage Represents a enity as JSON
type JSONMessage interface {
	// Bytes return the JSON bytes of an entity
	Bytes() json.RawMessage
}

// NewJSONMessage creates a JSONMessage using the provided bytes. The bytes should represent valid JSON
func NewJSONMessage(data []byte) JSONMessage {
	return &jsonMessage{data}
}

type jsonMessage struct {
	data json.RawMessage
}

func (m *jsonMessage) Bytes() json.RawMessage {
	return m.data
}

// MarshalJSON returns *m as the JSON encoding of m.
func (m *jsonMessage) MarshalJSON() ([]byte, error) {
	return m.data.MarshalJSON()
}

// UnmarshalJSON sets *m to a copy of data.
func (m *jsonMessage) UnmarshalJSON(data []byte) error {
	return m.data.UnmarshalJSON(data)
}
