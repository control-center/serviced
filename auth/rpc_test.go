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
	"crypto/rsa"
	"encoding/binary"
	"time"

	"github.com/control-center/serviced/auth"
	. "gopkg.in/check.v1"
)

var (
	endian = binary.BigEndian
)

func (s *TestAuthSuite) TestExtractBadRPCHeader(c *C) {
	mockHeader := []byte{0, 0, 0, 19, 109, 121, 32, 115, 117, 112, 101, 114, 32, 102}
	_, err := auth.ExtractRPCHeader(mockHeader)
	c.Assert(err, Equals, auth.ErrBadRPCHeader)
}

func (s *TestAuthSuite) TestExtractBadToken(c *C) {
	badToken := []byte("This is not a token")
	tokenLength := len(badToken)
	mockHeaderLength := auth.TOKEN_LEN_BYTES + tokenLength + auth.SIGNATURE_BYTES
	mockHeader := make([]byte, mockHeaderLength)
	endian.PutUint32(mockHeader[:auth.TOKEN_LEN_BYTES], uint32(tokenLength))
	_ = copy(mockHeader[auth.TOKEN_LEN_BYTES:auth.TOKEN_LEN_BYTES+tokenLength], badToken)
	_, err := auth.ExtractRPCHeader(mockHeader)
	c.Assert(err, Equals, auth.ErrBadToken)
}

func (s *TestAuthSuite) TestExtractBadSignature(c *C) {
	fakeToken, _, err := auth.CreateJWTIdentity(s.hostId, s.poolId, s.admin, s.dfs, s.delegatePubPEM, time.Hour)
	c.Assert(err, IsNil)
	c.Assert(fakeToken, NotNil)
	tokenLength := len(fakeToken)
	mockHeaderLength := auth.TOKEN_LEN_BYTES + tokenLength + auth.SIGNATURE_BYTES
	mockHeader := make([]byte, mockHeaderLength)
	endian.PutUint32(mockHeader[:auth.TOKEN_LEN_BYTES], uint32(tokenLength))
	_ = copy(mockHeader[auth.TOKEN_LEN_BYTES:auth.TOKEN_LEN_BYTES+tokenLength], fakeToken)
	_, err = auth.ExtractRPCHeader(mockHeader)
	c.Assert(err, Equals, rsa.ErrVerification)
}

func (s *TestAuthSuite) TestBuildAndExtractRPCHeader(c *C) {
	fakeToken, _, err := auth.CreateJWTIdentity(s.hostId, s.poolId, s.admin, s.dfs, s.delegatePubPEM, time.Hour)
	c.Assert(err, IsNil)
	c.Assert(fakeToken, NotNil)
	// build header
	header, err := auth.BuildAuthRPCHeader(fakeToken)
	c.Assert(err, Equals, nil)
	c.Assert(header, NotNil)
	// extract header
	ident, err := auth.ExtractRPCHeader(header)
	c.Assert(err, IsNil)
	// check the identity has been correctly extracted
	c.Assert(s.hostId, DeepEquals, ident.HostID())
	c.Assert(s.poolId, DeepEquals, ident.PoolID())
	c.Assert(s.admin, Equals, ident.HasAdminAccess())
	c.Assert(s.dfs, Equals, ident.HasDFSAccess())
}
