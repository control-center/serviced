// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
)

type JsonMessage interface {
	Bytes() []byte
}

func NewJsonMessage(data []byte) JsonMessage {
	return &jsonMessage{data}
}

type jsonMessage struct {
	data json.RawMessage
}

// MarshalJSON returns *m as the JSON encoding of m.
func (m *jsonMessage) MarshalJSON() ([]byte, error) {
	return m.data.MarshalJSON()
}

// UnmarshalJSON sets *m to a copy of data.
func (m *jsonMessage) UnmarshalJSON(data []byte) error {
	return m.data.UnmarshalJSON(data)
}
func (m *jsonMessage) Bytes() []byte {
	return m.data
}
