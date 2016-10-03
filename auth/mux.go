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
	"bytes"
	"errors"
	"io"
	"net"
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
	ErrBadMuxHeader  = errors.New("Bad mux header")
)

func BuildAuthMuxHeader(address []byte, token string) ([]byte, error) {
	if len(address) != ADDRESS_BYTES {
		return nil, ErrBadMuxAddress
	}
	headerBuf := new(bytes.Buffer)

	// add token length
	var tokenLen uint32 = uint32(len(token))
	tokenLenBuf := make([]byte, 4)
	endian.PutUint32(tokenLenBuf, tokenLen)
	headerBuf.Write(tokenLenBuf)

	// add token
	headerBuf.Write([]byte(token))

	// add address
	headerBuf.Write([]byte(address))

	// Sign what we have so far
	signature, err := SignAsDelegate(headerBuf.Bytes())
	if err != nil {
		return nil, err
	}
	// add signature to header
	headerBuf.Write(signature)

	return headerBuf.Bytes(), nil
}

func errorExtractingHeader(err error) ([]byte, Identity, error) {
	return nil, nil, err
}

func ExtractMuxHeader(rawHeader []byte) ([]byte, Identity, error) {
	if len(rawHeader) <= TOKEN_LEN_BYTES+ADDRESS_BYTES {
		return errorExtractingHeader(ErrBadMuxHeader)
	}

	var offset uint32 = 0

	// First four bytes represents the token length
	tokenLen := endian.Uint32(rawHeader[offset : offset+TOKEN_LEN_BYTES])
	offset += TOKEN_LEN_BYTES
	if len(rawHeader) <= TOKEN_LEN_BYTES+int(tokenLen)+ADDRESS_BYTES {
		return errorExtractingHeader(ErrBadMuxHeader)
	}

	// Next tokeLen bytes contain the token
	token := string(rawHeader[offset : offset+tokenLen])
	offset += tokenLen

	// Validate the token can be parsed
	senderIdentity, err := ParseJWTIdentity(token)
	if err != nil {
		return errorExtractingHeader(err)
	}
	if senderIdentity == nil {
		return errorExtractingHeader(ErrBadToken)
	}

	// Next six bytes is going to be the address
	address := rawHeader[offset : offset+ADDRESS_BYTES]
	offset += ADDRESS_BYTES

	// get the part of the header that has been signed
	signed_message := rawHeader[:offset]

	// Whatever is left is the signature
	signature := rawHeader[offset:]

	// Verify the identity of the signed message
	senderVerifier, err := senderIdentity.Verifier()
	if err != nil {
		return errorExtractingHeader(err)
	}
	err = senderVerifier.Verify(signed_message, signature)
	if err != nil {
		return errorExtractingHeader(err)
	}

	return address, senderIdentity, nil
}

func ReadMuxHeader(conn net.Conn) ([]byte, error) {
	// Read token Length
	tokenLenBuff := make([]byte, TOKEN_LEN_BYTES)
	_, err := io.ReadFull(conn, tokenLenBuff)
	if err != nil {
		return nil, err
	}
	tokenLen := endian.Uint32(tokenLenBuff)
	// Read rest of the header
	remainderBuff := make([]byte, tokenLen+ADDRESS_BYTES+SIGNATURE_BYTES)
	_, err = io.ReadFull(conn, remainderBuff)
	if err != nil {
		return nil, err
	}
	return append(tokenLenBuff, remainderBuff...), nil
}
