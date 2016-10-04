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

	jwt "github.com/dgrijalva/jwt-go"
	"time"
)

/*
   When sending an RPC request, if authentication is required, the client sends an authentication token as well
   as its signature. The token determines if the sender is authorized to make the RPC request

   -------------------------------------------------------------------------------------------------------------------------------
   |Auth Token length (4 bytes)|Auth Token (N bytes)|Timestamp(8 bytes)|Signature of Auth Token + Timestamp + Request(256 bytes) |
   -------------------------------------------------------------------------------------------------------------------------------
*/

const (
	TIMESTAMP_BYTES = 8
)

var (
	ErrBadRPCHeader   = errors.New("Bad rpc header")
	ErrRequestExpired = errors.New("Expired rpc request")
)

type RPCHeaderParser interface {
	ParseHeader(header []byte, request []byte) (Identity, error)
}
type RPCHeaderBuilder interface {
	BuildHeader([]byte) ([]byte, error)
}

type RPCHeaderHandler struct{}

func (r *RPCHeaderHandler) BuildHeader(req []byte) ([]byte, error) {
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
		log.WithError(err).Debug("Unable to retrieve delegate token")
		// We may be an un-added master
		token, err2 = MasterToken()
		if err2 != nil {
			log.WithError(err2).Debug("Unable to retrieve master token")
			// Return the original error message
			return nil, err
		}
		signAsMaster = true
	}

	return r.BuildAuthRPCHeader(token, req, signAsMaster)
}

func (r *RPCHeaderHandler) BuildAuthRPCHeader(token string, req []byte, signAsMaster bool) ([]byte, error) {
	headerBuf := new(bytes.Buffer)

	// add token length
	var tokenLen uint32 = uint32(len(token))
	tokenLenBuf := make([]byte, TOKEN_LEN_BYTES)
	endian.PutUint32(tokenLenBuf, tokenLen)
	headerBuf.Write(tokenLenBuf)

	// add token
	headerBuf.Write([]byte(token))

	// add timestamp
	var timestamp uint64 = uint64(time.Now().UTC().Unix())
	timestampBuf := make([]byte, TIMESTAMP_BYTES)
	endian.PutUint64(timestampBuf, timestamp)
	headerBuf.Write(timestampBuf)

	// Build the data we want to sign (token + timestamp + request)
	// copy headerBuf into a new buffer
	sigBuffer := bytes.NewBuffer([]byte(token))

	// Add the timestamp
	sigBuffer.Write(timestampBuf)

	// add the request
	sigBuffer.Write(req)

	// Sign sigBuffer
	signer := SignAsDelegate
	if signAsMaster {
		signer = SignAsMaster
	}

	signature, err := signer(sigBuffer.Bytes())
	if err != nil {
		return nil, err
	}

	// add signature to header
	headerBuf.Write(signature)

	return headerBuf.Bytes(), nil
}

// Extracts the token and signature from the header, and validates the signature against the token and request
func (r *RPCHeaderHandler) ParseHeader(rawHeader []byte, req []byte) (Identity, error) {

	if len(rawHeader) <= TOKEN_LEN_BYTES+TIMESTAMP_BYTES {
		return nil, ErrBadRPCHeader
	}

	var offset uint32 = 0

	// First four bytes represents the token length
	tokenLen := endian.Uint32(rawHeader[offset : offset+TOKEN_LEN_BYTES])
	offset += TOKEN_LEN_BYTES
	if len(rawHeader) <= TOKEN_LEN_BYTES+int(tokenLen) {
		return nil, ErrBadRPCHeader
	}

	// Next tokenLen bytes contain the token
	token := string(rawHeader[offset : offset+tokenLen])
	offset += tokenLen

	// Validate the token can be parsed
	senderIdentity, err := ParseJWTIdentity(token)
	if err != nil {
		return nil, err
	}
	if senderIdentity == nil {
		return nil, ErrBadToken
	}

	// Next 8 bytes contains the timestamp
	timestampBuf := rawHeader[offset : offset+TIMESTAMP_BYTES]
	timestamp := endian.Uint64(timestampBuf)
	offset += TIMESTAMP_BYTES

	// Validate timestamp (should be no earlier than current time - expirationDelta)
	requestTime := time.Unix(int64(timestamp), 0)
	cutoffTime := jwt.TimeFunc().UTC().Add(-expirationDelta)
	if requestTime.Before(cutoffTime) {
		return nil, ErrRequestExpired
	}

	// Grab the signature off the header
	signature := rawHeader[offset:]

	// Build the message that was signed (token + timestamp + req)
	signed_message := bytes.NewBuffer([]byte(token))

	// Add the timestamp
	signed_message.Write(timestampBuf)

	// add the request
	signed_message.Write(req)

	// Verify the identity of the signed message
	senderVerifier, err := senderIdentity.Verifier()
	if err != nil {
		return nil, err
	}
	err = senderVerifier.Verify(signed_message.Bytes(), signature)
	if err != nil {
		return nil, err
	}

	return senderIdentity, nil
}
