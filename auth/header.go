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
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

/*
The authentication header protocol comprises information necessary to decide
whether a connection or request should be permitted or rejected, including the
signed identity of the sender, the timestamp of the connection's initiation,
and a signature of the token and payload.

The following diagram shows the components of the header in byte order:

 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-----+-+---------------+---+---+-------------------------------+
|  M  |V|   TIMESTAMP   |T L|P L|            TOKEN              |
|  A  |E|      (8)      |O E|A E|        (max len 32k)          |
|  G  |R|               |K N|Y N|                               |
|  I  |S|               |E  |L  |                               |
|  C  |N|               |N  |D  |                               |
+-+-+-+-+---------------+---+---+ - - - - - - - - - - - - - - - +
:                    TOKEN continued ...                        :
+---------------------------------------------------------------+
:                    PAYLOAD (max len 32k)                      |
+ - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
:                    PAYLOAD continued ...                      :
+---------------------------------------------------------------+
|                    SIGNATURE (256)                            |
+ - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
:                    SIGNATURE continued ...                    :
+---------------------------------------------------------------+

MAGIC:     A magic number to identify this as an auth header connection

VERSN:     The version of the protocol being used (currently 1)

TIMESTAMP: A Unix timestamp (UTC)

TOKEN LEN: The length of the authentication token, limited to 32k

PAYLD LEN: The length of the header payload, limited to 32k

TOKEN:     A JSON Web Token, signed by a trusted third party (the master),
		   comprising the sender's identity, including a public key that can be
		   used to verify the header itself.

PAYLOAD:   The payload, if any. For the mux, this will be the target address
		   proxied by this connection.

SIGNATURE: An RSA signature of the entire header minus the magic number and
		   protocol version, using the private key whose public key is
		   contained in the identity token.
*/

type magicNumber [3]byte
type protocolVersion uint8
type timestamp uint64
type tokenLength uint16
type payloadLength uint16
type signature [256]byte

var (
	// ProtocolVersion is the current protocol version
	ProtocolVersion protocolVersion = 1
	// MagicNumber is a magic number to identify headers as auth
	MagicNumber = magicNumber([3]byte{139, 143, 165})
	byteOrder   = binary.BigEndian

	tl tokenLength
	pl payloadLength

	// ErrInvalidAuthHeader is thrown when the auth header is invalid
	ErrInvalidAuthHeader = errors.New("Invalid authentication header")
	// ErrAuthProtocolVersion is thrown when the auth protocol version passed is unknown
	ErrAuthProtocolVersion = errors.New("Unknown authentication protocol version")
	// ErrPayloadTooLarge is thrown when the payload length for an auth header overflows a uint16
	ErrPayloadTooLarge = errors.New("Authentication header payload is too large")
	// ErrUnknownAuthProtocol is thrown when the authentication protocol version is not supported
	ErrUnknownAuthProtocol = errors.New("Unknown authentication protocol version")
	// ErrHeaderExpired is thrown when an authentication header is more than 10s old
	ErrHeaderExpired = errors.New("Expired authentication header")
)

// authHeaderWriterTo is a simple struct that has references to all the
// user-supplied auth header data, and thus can build the header
type authHeaderWriterTo struct {
	token   []byte
	payload []byte
	signer  Signer
}

// WriteTo satisfies the io.WriterTo interface. It builds an auth header
// according to the spec, signs it, and writes it to the specified writer.
func (h *authHeaderWriterTo) WriteTo(w io.Writer) (n int64, err error) {

	var (
		signedContent bytes.Buffer
		bytesWritten  int64
	)

	// Timestamp
	ts := jwt.TimeFunc().UTC().Unix()
	if err = binary.Write(&signedContent, byteOrder, timestamp(ts)); err != nil {
		return 0, err
	}

	// Token len
	tokenLen := len(h.token)
	// Check to see if it overflows uint16
	if int(tokenLength(tokenLen)) != tokenLen {
		return 0, ErrBadToken
	}
	if err = binary.Write(&signedContent, byteOrder, tokenLength(tokenLen)); err != nil {
		return 0, err
	}

	// Payload len
	payloadLen := len(h.payload)
	// Check to see if it overflows uint16
	if int(payloadLength(payloadLen)) != payloadLen {
		return 0, ErrPayloadTooLarge
	}
	if err = binary.Write(&signedContent, byteOrder, payloadLength(payloadLen)); err != nil {
		return 0, err
	}

	// Token
	if err = binary.Write(&signedContent, byteOrder, h.token); err != nil {
		return 0, err
	}

	// Payload
	if err = binary.Write(&signedContent, byteOrder, h.payload); err != nil {
		return 0, err
	}

	// Sign the contents of the buffer
	sig, err := h.signer.Sign(signedContent.Bytes())
	if err != nil {
		return 0, err
	}

	// Write the magic number
	err = binary.Write(w, byteOrder, MagicNumber)
	if err != nil {
		return n, err
	}
	n += int64(binary.Size(MagicNumber))

	// Write the protocol version
	err = binary.Write(w, byteOrder, ProtocolVersion)
	if err != nil {
		return n, err
	}
	n += int64(binary.Size(ProtocolVersion))

	// Write the buffer contents
	bytesWritten, err = signedContent.WriteTo(w)
	if err != nil {
		return n, err
	}
	n += bytesWritten

	// Write the signature
	err = binary.Write(w, byteOrder, sig)
	if err != nil {
		return n, err
	}
	n += int64(binary.Size(sig))

	return n, nil
}

func NewAuthHeader(token, payload []byte, signer Signer) io.WriterTo {
	return &authHeaderWriterTo{token, payload, signer}
}

func ReadAuthHeader(r io.Reader) (sender Identity, timestamp time.Time, payload []byte, err error) {

	// Read and verify the first three bytes are the magic number
	var m magicNumber
	err = binary.Read(r, byteOrder, &m)
	if err != nil {
		return
	}
	if m != MagicNumber {
		err = ErrInvalidAuthHeader
		return
	}

	// Read the protocol version
	var pv protocolVersion
	err = binary.Read(r, byteOrder, &pv)
	if err != nil {
		return
	}

	// Pass to the appropriate protocol handler, or error if it's a protocol
	// version we don't support
	switch pv {
	case ProtocolVersion:
		return readAuthHeaderV1(r)
	}
	err = ErrUnknownAuthProtocol
	return
}

func readAuthHeaderV1(r io.Reader) (sender Identity, tstamp time.Time, payload []byte, err error) {

	var signable bytes.Buffer
	teed := io.TeeReader(r, &signable)

	// Read and validate the timestamp
	var ts timestamp
	if err = binary.Read(teed, byteOrder, &ts); err != nil {
		return
	}
	tstamp = time.Unix(int64(ts), 0).UTC()
	cutoff := jwt.TimeFunc().UTC().Add(-expirationDelta)
	if tstamp.Before(cutoff) {
		err = ErrHeaderExpired
		return
	}

	// Read the token length
	var tokenLen tokenLength
	if err = binary.Read(teed, byteOrder, &tokenLen); err != nil {
		return
	}

	// Read the payload length
	var payloadLen payloadLength
	if err = binary.Read(teed, byteOrder, &payloadLen); err != nil {
		return
	}

	// Read the token and validate it
	token := make([]byte, int(tokenLen))
	if err = binary.Read(teed, byteOrder, &token); err != nil {
		return
	}
	sender, err = ParseJWTIdentity(string(token))
	if err != nil {
		return
	}
	if sender == nil {
		err = ErrBadToken
		return
	}

	// Read the payload
	payload = make([]byte, int(payloadLen))
	if err = binary.Read(teed, byteOrder, &payload); err != nil {
		return
	}

	// Freeze the signable content for verification. Even though we don't plan
	// to read from the tee reader anymore, might as well be clear as to our
	// intentions.
	signableContent := signable.Bytes()

	// Read the signature from the original reader
	var sig = make([]byte, 256)
	if err = binary.Read(r, byteOrder, sig); err != nil {
		return
	}

	// Verify the signature of the signable content
	var verifier Verifier
	verifier, err = sender.Verifier()
	if err != nil {
		return
	}
	err = verifier.Verify(signableContent, sig)
	return
}
