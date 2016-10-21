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
	"io/ioutil"
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
+-----+-+-------+--------+---+---+------------------------------+
|  M  |V|   L   |                                               |
|  A  |E|   E   |                                               |
|  G  |R|   N   |              SIGNATURE (256)                  |
|  I  |S|       |                                               |
|  C  |N|       |                                               |
+-+-+-+-+-------+- - - - - - - - - - - - - - - - - - - - - - - -+
:                    SIGNATURE continued ...                    :
+-----------+---------------------------------------------------+
| TIMESTAMP |                   AUTH TOKEN                      |
|    (8)    |                                                   |
+-----------+ - - - - - - - - - - - - - - - - - - - - - - - - - +
:                    AUTH TOKEN continued ...                   :
+---+-----------------------------------------------------------+
|P L|                                                           |
|A E|                                                           |
|Y N|               PAYLOAD (max len 32k)                       |
|L  |                                                           |
|D  |                                                           |
+---+ - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
:                    PAYLOAD continued ...                      :
+---------------------------------------------------------------+


MAGIC:     A magic number to identify this as an auth header connection

VERSN:     The version of the protocol being used (currently 1)

LEN:       The length of the timestamp and token, limited to 32k

SIGNATURE: An RSA signature of the entire header minus the magic number and
           protocol version, using the private key whose public key is
           contained in the identity token.

TIMESTAMP: A Unix timestamp (UTC)

TOKEN:     A JSON Web Token, signed by a trusted third party (the master),
           comprising the sender's identity, including a public key that can be
           used to verify the header itself.

PAYLD LEN: The length of the header payload, limited to 32k

PAYLOAD:   The payload, if any. For the mux, this will be the target address
           proxied by this connection.

*/

type magicNumber [3]byte
type protocolVersion uint8
type timestamp uint64
type tokenLength uint16
type payloadLength uint32
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

// AuthHeaderError wraps another error with an accompanying payload, for the
// sake of those receiving the error who need access to the payload.
type AuthHeaderError struct {
	Err     error
	Payload []byte
}

// Error implements the error interface.
func (e *AuthHeaderError) Error() string {
	return e.Err.Error()
}

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

	// MAGIC VERSION | ALLLEN SIGNATURE TIMESTAMP TOKEN PAYLOADLEN PAYLOAD

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

	// Token
	if err = binary.Write(&signedContent, byteOrder, h.token); err != nil {
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

	// Write the length of everything left except the payload len and payload
	allLen := uint32(tokenLen + 8)
	err = binary.Write(w, byteOrder, allLen)
	if err != nil {
		return n, err
	}

	// Write the signature
	err = binary.Write(w, byteOrder, sig)
	if err != nil {
		return n, err
	}
	n += int64(binary.Size(sig))

	// Write the buffer contents (timestamp, token, payloadlen, payload)
	bytesWritten, err = signedContent.WriteTo(w)
	if err != nil {
		return n, err
	}
	n += bytesWritten

	return n, nil
}

// NewAuthHeaderWriterTo returns an io.WriterTo that can write an
// authentication header with the given parameters to a Writer.
func NewAuthHeaderWriterTo(token, payload []byte, signer Signer) io.WriterTo {
	return &authHeaderWriterTo{token, payload, signer}
}

// ReadAuthHeader reads an authentication header from the reader given. If
// there's a non-EOF error, it will read to the payload and return an
// AuthHeaderError with the contents of the payload and the original error.
// This allows us to support protocols that require the payload for bookkeeping
// purposes, like RPC.
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

// readAuthHeaderV1 implements version 1 of the authentication header protocol.
func readAuthHeaderV1(r io.Reader) (sender Identity, tstamp time.Time, payload []byte, err error) {

	// Read in the length of everything up to the payload length
	var allLen uint32
	if err = binary.Read(r, byteOrder, &allLen); err != nil {
		return
	}

	// Read the signature
	var sig = make([]byte, 256)
	if err = binary.Read(r, byteOrder, sig); err != nil {
		return
	}

	var (
		signable  bytes.Buffer
		remaining = allLen
	)

	// Read into a tee reader so we can verify the signature on the original
	// data later
	teed := io.TeeReader(r, &signable)

	// Read and validate the timestamp
	var ts timestamp
	remaining -= uint32(binary.Size(ts))
	if err = binary.Read(teed, byteOrder, &ts); err != nil {
		err = eatBytesAndGetPayloadError(teed, remaining, err)
		return
	}
	tstamp = time.Unix(int64(ts), 0).UTC()
	cutoff := jwt.TimeFunc().UTC().Add(-expirationDelta)
	if tstamp.Before(cutoff) {
		err = eatBytesAndGetPayloadError(teed, remaining, ErrHeaderExpired)
		return
	}

	// Read the token and validate it
	token := make([]byte, int(remaining))
	if err = binary.Read(teed, byteOrder, &token); err != nil {
		err = eatBytesAndGetPayloadError(teed, 0, err)
		return
	}
	sender, err = ParseJWTIdentity(string(token))
	if err != nil {
		err = eatBytesAndGetPayloadError(teed, 0, err)
		return
	}
	if sender == nil {
		err = eatBytesAndGetPayloadError(teed, 0, ErrBadToken)
		return
	}

	payload, err = readPayload(teed)
	if err != nil {
		return
	}

	// Freeze the signable content for verification. Even though we don't plan
	// to read from the tee reader anymore, might as well be clear as to our
	// intentions.
	signableContent := signable.Bytes()

	// Verify the signature of the signable content
	var verifier Verifier
	verifier, err = sender.Verifier()
	if err != nil {
		err = &AuthHeaderError{err, payload}
		return
	}
	err = verifier.Verify(signableContent, sig)
	if err != nil {
		err = &AuthHeaderError{err, payload}
	}
	return
}

// eatBytesAndGetPayloadError fast-forwards the reader to the payload, reads
// off the payload, and wraps the provided error with the payload in an
// AuthHeaderError.
func eatBytesAndGetPayloadError(r io.Reader, n uint32, e error) error {
	written, err := io.CopyN(ioutil.Discard, r, int64(n))
	if err != nil {
		return ErrReadingBody
	}
	if written != int64(n) {
		return ErrReadingBody
	}
	payload, err := readPayload(r)
	if err != nil {
		return err
	}
	return &AuthHeaderError{e, payload}
}

// readPayload reads a payload length and the accompanying payload from
// a reader.
func readPayload(r io.Reader) ([]byte, error) {
	// Read the payload length
	var payloadLen payloadLength
	if err := binary.Read(r, byteOrder, &payloadLen); err != nil {
		return nil, err
	}

	// Read the payload
	payload := make([]byte, int(payloadLen))
	if err := binary.Read(r, byteOrder, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}
