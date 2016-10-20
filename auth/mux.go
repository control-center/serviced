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

package auth

import (
	"encoding/binary"
	"errors"
	"io"
)

/*
   When establishing a connection to the mux, in addition to the address of the receiver,
   the sender sends an authentication token and signs the whole message. The token determines
   if the sender is authorized to send data to the receiver or not

   ---------------------------------------------------------------------------------------------------------
   | Auth Token length (4 bytes)  |     Auth Token (N bytes)  | Address (6 bytes) |  Signature (256 bytes) |
   ---------------------------------------------------------------------------------------------------------
*/

const (
	ADDRESS_BYTES = 6
)

var (
	ErrBadMuxAddress = errors.New("Bad mux address")

	endian = binary.BigEndian
)

func AddSignedMuxHeader(w io.Writer, address []byte, token string) error {
	if len(address) != ADDRESS_BYTES {
		return ErrBadMuxAddress
	}
	header := NewAuthHeader([]byte(token), address, &delegateKeys)
	_, err := header.WriteTo(w)
	return err
}

func ReadMuxHeader(r io.Reader) ([]byte, Identity, error) {
	sender, _, address, err := ReadAuthHeader(r)
	return address, sender, err
}
