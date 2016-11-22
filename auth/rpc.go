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
	"encoding/binary"
	"errors"
	"io"
)

var (
	// RPCMagicNumber is a magic number for RPC
	RPCMagicNumber = []byte{224, 227, 155}
	// BodyLenLen is the length of the length of the payload
	BodyLenLen = 4

	// ErrBadRPCHeader is thrown when the RPC header is bad
	ErrBadRPCHeader = errors.New("Bad rpc header")
	// ErrWritingLength is thrown when the length writing has an error
	ErrWritingLength = errors.New("Wrote too few bytes for message length")
	// ErrWritingBody is thrown when the body writing has an error
	ErrWritingBody = errors.New("Wrote too few bytes for message body")
	// ErrReadingLength is thrown when we read too few bytes for message length
	ErrReadingLength = errors.New("Read too few bytes for message length")
	// ErrReadingBody is thrown when we read too few bytes for message body
	ErrReadingBody = errors.New("Read too few bytes for message body")
)

// RPCHeaderParser reads headers from readers and parses them
type RPCHeaderParser interface {
	// ReadHeader reads an RPC header from a reader, parsing the authentication
	// header, if any.
	ReadHeader(io.Reader) (Identity, []byte, error)
}

// RPCHeaderBuilder builds headers and writes them to writers
type RPCHeaderBuilder interface {
	// WriteHeader writes an RPC header to the provided writer. Optionally, it
	// writes an authentication header as part of the RPC header.
	WriteHeader(io.Writer, []byte, bool) error
}

// RPCHeaderHandler is an implementation of RPCHeaderParser and RPCHeaderBuilder
type RPCHeaderHandler struct{}

// WriteLengthAndBytes writes the length of a byte array and then the bytes
// themselves. It is the inverse of ReadLengthAndBytes.
func WriteLengthAndBytes(b []byte, writer io.Writer) error {
	// write length
	var pl payloadLength = payloadLength(len(b))
	if err := binary.Write(writer, byteOrder, pl); err != nil {
		return err
	}
	if err := binary.Write(writer, byteOrder, b); err != nil {
		return err
	}
	return nil
}

// ReadLengthAndBytes reads the length of a byte array and then the bytes
// themselves. It is the inverse of WriteLengthAndBytes.
func ReadLengthAndBytes(reader io.Reader) ([]byte, error) {
	// Read the length of the data
	var payloadLen payloadLength
	if err := binary.Read(reader, byteOrder, &payloadLen); err != nil {
		return nil, err
	}

	// Now read the data
	b := make([]byte, payloadLen)
	if err := binary.Read(reader, byteOrder, &b); err != nil {
		return nil, err
	}
	return b, nil
}

// WriteHeader writes an RPC header to the provided writer. Optionally, it
// writes an authentication header as part of the RPC header.
func (r *RPCHeaderHandler) WriteHeader(w io.Writer, req []byte, writeAuth bool) error {
	var (
		token string
		err   error
		err2  error
	)
	binary.Write(w, byteOrder, RPCMagicNumber)
	if writeAuth {
		binary.Write(w, byteOrder, uint8(1))
		// get current host token
		var signer Signer = &delegateKeys
		token, err = AuthTokenNonBlocking()
		if err != nil {
			log.WithError(err).Debug("Unable to retrieve delegate token")
			// We may be an un-added master
			token, err2 = MasterToken()
			if err2 != nil {
				log.WithError(err2).Debug("Unable to retrieve master token")
				// Return the original error message
				return err
			}
			signer = &masterKeys
		}
		h := NewAuthHeaderWriterTo([]byte(token), req, signer)
		_, err = h.WriteTo(w)
	} else {
		binary.Write(w, byteOrder, uint8(0))
		WriteLengthAndBytes(req, w)
	}
	return err
}

// ReadHeader reads an RPC header from a reader, parsing the authentication
// header, if any.
func (r *RPCHeaderHandler) ReadHeader(reader io.Reader) (Identity, []byte, error) {
	// Read and verify the first three bytes are the magic number
	var (
		m       = make([]byte, 3)
		sender  Identity
		payload []byte
	)
	if err := binary.Read(reader, byteOrder, &m); err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(m, RPCMagicNumber) {
		return nil, nil, ErrBadRPCHeader
	}
	var hasAuth [1]byte
	err := binary.Read(reader, byteOrder, &hasAuth)
	if err != nil {
		return nil, nil, err
	}
	if hasAuth[0] == 1 {
		sender, _, payload, err = ReadAuthHeader(reader)
		if err != nil {
			return nil, nil, err
		}
	} else {
		payload, err = ReadLengthAndBytes(reader)
		if err != nil {
			return nil, nil, err
		}
	}
	return sender, payload, nil
}
