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
)

/*
   When sending an RPC request, if authentication is required, the client sends an authentication token as well
   as its signature. The token determines if the sender is authorized to make the RPC request

   ----------------------------------------------------------------------------------------
   | Auth Token length (4 bytes)  |     Auth Token (N bytes)  | Signature (256 bytes) |
   ----------------------------------------------------------------------------------------
*/

var (
	ErrBadRPCHeader = errors.New("Bad rpc header")
)

type RPCHeaderParser interface {
	ParseHeader([]byte) (Identity, error)
}
type RPCHeaderBuilder interface {
	BuildHeader() ([]byte, error)
}

type RPCHeaderHandler struct{}

func (r *RPCHeaderHandler) BuildHeader() ([]byte, error) {
	var (
		token        string
		err          error
		err2         error
		signAsMaster bool
	)

	// get current host token
	signAsMaster = false
	token, err = AuthTokenNonBlocking()
	if err != nil {
		// We may be an un-added master
		token, err2 = MasterToken()
		if err2 != nil {
			log.WithError(err2).Debug("Unable to retrieve master token")
			// Return the original error message
			return nil, err
		}
		signAsMaster = true
	}

	return r.BuildAuthRPCHeader(token, signAsMaster)
}

func (r *RPCHeaderHandler) BuildAuthRPCHeader(token string, signAsMaster bool) ([]byte, error) {
	headerBuf := new(bytes.Buffer)

	// add token length
	var tokenLen uint32 = uint32(len(token))
	tokenLenBuf := make([]byte, TOKEN_LEN_BYTES)
	endian.PutUint32(tokenLenBuf, tokenLen)
	headerBuf.Write(tokenLenBuf)

	// add token
	headerBuf.Write([]byte(token))

	// Sign what we have so far
	signer := SignAsDelegate
	if signAsMaster {
		signer = SignAsMaster
	}

	signature, err := signer(headerBuf.Bytes())
	if err != nil {
		return nil, err
	}

	// add signature to header
	headerBuf.Write(signature)

	return headerBuf.Bytes(), nil
}

func (r *RPCHeaderHandler) ParseHeader(rawHeader []byte) (Identity, error) {

	if len(rawHeader) <= TOKEN_LEN_BYTES {
		return nil, ErrBadRPCHeader
	}

	var offset uint32 = 0

	// First four bytes represents the token length
	tokenLen := endian.Uint32(rawHeader[offset : offset+TOKEN_LEN_BYTES])
	offset += TOKEN_LEN_BYTES
	if len(rawHeader) <= TOKEN_LEN_BYTES+int(tokenLen) {
		return nil, ErrBadRPCHeader
	}

	// Next tokeLen bytes contain the token
	token := string(rawHeader[offset : offset+tokenLen])
	offset += tokenLen

	// Validate the token can be parsed
	senderIdentity, err := ParseJWTIdentity(token)
	if err != nil || senderIdentity == nil {
		if err == nil || senderIdentity == nil {
			err = ErrBadToken
		}
		return nil, err
	}

	// get the part of the header that has been signed
	signed_message := rawHeader[:offset]

	// Whatever is left is the signature
	signature := rawHeader[offset:]

	// Verify the identity of the signed message
	senderVerifier, err := senderIdentity.Verifier()
	if err != nil {
		return nil, err
	}
	err = senderVerifier.Verify(signed_message, signature)
	if err != nil {
		return nil, err
	}

	return senderIdentity, nil
}
