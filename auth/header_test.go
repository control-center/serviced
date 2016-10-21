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

// +build unit

package auth_test

import (
	"bytes"
	"io"
	"time"

	"github.com/control-center/serviced/auth"
	. "gopkg.in/check.v1"
)

var authHeaderError *auth.AuthHeaderError

func (s *TestAuthSuite) getHeader(payload []byte) io.WriterTo {
	tokenString, _, _ := auth.CreateJWTIdentity(s.hostId, s.poolId, s.admin, s.dfs, s.delegatePubPEM, time.Hour)
	signer, _ := auth.RSASignerFromPEM(s.delegatePrivPEM)
	header := auth.NewAuthHeaderWriterTo([]byte(tokenString), payload, signer)
	return header
}

func (s *TestAuthSuite) TestHappyPath(c *C) {
	var b bytes.Buffer

	pl := []byte("payload")
	header := s.getHeader(pl)
	_, err := header.WriteTo(&b)
	c.Assert(err, IsNil)

	sender, timestamp, payload, err := auth.ReadAuthHeader(&b)
	c.Assert(err, IsNil)

	c.Assert(sender.HostID(), Equals, s.hostId)
	c.Assert(sender.PoolID(), Equals, s.poolId)

	c.Assert(timestamp.Before(time.Now().UTC()), Equals, true)
	c.Assert(string(payload), Equals, string(pl))

}

func (s *TestAuthSuite) TestBadMagicNumber(c *C) {
	var b bytes.Buffer

	pl := []byte("payload")
	header := s.getHeader(pl)

	b.Write([]byte{13})
	_, err := header.WriteTo(&b)
	c.Assert(err, IsNil)

	sender, timestamp, payload, err := auth.ReadAuthHeader(&b)
	c.Assert(sender, IsNil)
	c.Assert(timestamp.IsZero(), Equals, true)
	c.Assert(payload, IsNil)
	c.Assert(err, Equals, auth.ErrInvalidAuthHeader)
}

func (s *TestAuthSuite) TestBadProtocolVersion(c *C) {
	var b bytes.Buffer

	pl := []byte("payload")
	header := s.getHeader(pl)

	_, err := header.WriteTo(&b)
	c.Assert(err, IsNil)

	bs := b.Bytes()
	bs[3] = 0 // We started with 1, so this will never be a valid proto version
	b.Reset()
	b.Write(bs)

	sender, timestamp, payload, err := auth.ReadAuthHeader(&b)
	c.Assert(err, Equals, auth.ErrUnknownAuthProtocol)
	c.Assert(sender, IsNil)
	c.Assert(timestamp.IsZero(), Equals, true)
	c.Assert(payload, IsNil)
}

func (s *TestAuthSuite) TestInvalidToken(c *C) {
	var b bytes.Buffer
	tokenString, _, _ := auth.CreateJWTIdentity(s.hostId, s.poolId, s.admin, s.dfs, s.delegatePubPEM, time.Hour)
	token := []byte(tokenString)
	// Break that there token
	token[10] += 1

	payload := []byte("payload")
	signer, _ := auth.RSASignerFromPEM(s.delegatePrivPEM)
	header := auth.NewAuthHeaderWriterTo(token, payload, signer)

	_, err := header.WriteTo(&b)
	c.Assert(err, IsNil)
	sender, _, _, err := auth.ReadAuthHeader(&b)
	c.Assert(sender, IsNil)

	c.Assert(err, FitsTypeOf, authHeaderError)
	e := err.(*auth.AuthHeaderError)
	c.Assert(e.Err, Equals, auth.ErrIdentityTokenBadSig)
	c.Assert(e.Payload, DeepEquals, payload)
}

func (s *TestAuthSuite) TestHugeToken(c *C) {
	var b bytes.Buffer

	token := make([]byte, 1<<16) // One byte too large
	signer, _ := auth.RSASignerFromPEM(s.delegatePrivPEM)
	header := auth.NewAuthHeaderWriterTo(token, []byte("payload"), signer)
	_, err := header.WriteTo(&b)
	c.Assert(err, Equals, auth.ErrBadToken)
}

func (s *TestAuthSuite) TestZeroToken(c *C) {
	var b bytes.Buffer

	payload := []byte("payload")
	token := []byte{}
	signer, _ := auth.RSASignerFromPEM(s.delegatePrivPEM)
	header := auth.NewAuthHeaderWriterTo(token, payload, signer)
	_, err := header.WriteTo(&b)
	c.Assert(err, IsNil)
	sender, _, _, err := auth.ReadAuthHeader(&b)
	c.Assert(sender, IsNil)
	c.Assert(err, FitsTypeOf, authHeaderError)
	e := err.(*auth.AuthHeaderError)
	c.Assert(e.Err, Equals, auth.ErrBadToken)
	c.Assert(e.Payload, DeepEquals, payload)
}

func (s *TestAuthSuite) TestZeroPayload(c *C) {
	var b bytes.Buffer

	pl := []byte{}
	header := s.getHeader(pl)
	_, err := header.WriteTo(&b)
	c.Assert(err, IsNil)

	sender, timestamp, payload, err := auth.ReadAuthHeader(&b)
	c.Assert(err, IsNil)

	c.Assert(sender.HostID(), Equals, s.hostId)
	c.Assert(sender.PoolID(), Equals, s.poolId)

	c.Assert(timestamp.Before(time.Now().UTC()), Equals, true)
	c.Assert(string(payload), Equals, string(pl))
}

func (s *TestAuthSuite) TestHugePayload(c *C) {
	var b bytes.Buffer

	payload := make([]byte, 1<<32) // One byte too large
	tokenString, _, _ := auth.CreateJWTIdentity(s.hostId, s.poolId, s.admin, s.dfs, s.delegatePubPEM, time.Hour)
	token := []byte(tokenString)
	signer, _ := auth.RSASignerFromPEM(s.delegatePrivPEM)
	header := auth.NewAuthHeaderWriterTo(token, payload, signer)
	_, err := header.WriteTo(&b)
	c.Assert(err, Equals, auth.ErrPayloadTooLarge)
}

func (s *TestAuthSuite) TestExpiredRequest(c *C) {
	var b bytes.Buffer

	payload := []byte("payload")
	header := s.getHeader(payload)
	_, err := header.WriteTo(&b)
	c.Assert(err, IsNil)

	auth.At(time.Now().UTC().Add(time.Hour), func() {
		_, _, _, err := auth.ReadAuthHeader(&b)
		c.Assert(err, FitsTypeOf, authHeaderError)
		e := err.(*auth.AuthHeaderError)
		c.Assert(e.Err, Equals, auth.ErrHeaderExpired)
		c.Assert(e.Payload, DeepEquals, payload)
	})

}
