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

package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

var defaultEncoder = base64.URLEncoding

var ErrInvalidPacket = errors.New("invalid packet")

// PacketData is the encoded message type that is provided by the RemotePool
// and received by the GovernedPool.
type PacketData struct {
	RemotePoolID  string
	RemoteAddress string
	Secret        string
}

// IsValid verifies that the packet data is valid
func (v PacketData) IsValid() bool {
	if v.RemotePoolID == "" {
		return false
	} else if v.RemoteAddress == "" {
		return false
	} else if v.Secret == "" {
		return false
	}
	return true
}

// EncodePacket transforms the PacketData object into a base64 standard encoded
// message.
func EncodePacket(packet PacketData) (string, error) {
	if raw, err := json.Marshal(packet); err != nil {
		return "", err
	} else {
		return defaultEncoder.EncodeToString(raw), nil
	}
}

// DecodePacket decodes a base64 standard encoded message into a packet data
// object.
func DecodePacket(msg string, packet *PacketData) error {
	if raw, err := defaultEncoder.DecodeString(msg); err != nil {
		return err
	} else if err := json.Unmarshal(raw, packet); err != nil {
		return err
	} else if !packet.IsValid() {
		return ErrInvalidPacket
	}

	return nil
}