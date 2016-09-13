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
	"time"

	"github.com/control-center/serviced/auth"
	. "gopkg.in/check.v1"
)

var (
	hostId = "MyHost"
	poolId = "MyPool"
	admin  = false
	dfs    = true
)

func (s *TestAuthSuite) TestBuildHeaderBadAddr(c *C) {
	fakeToken, err := auth.CreateJWTIdentity(hostId, poolId, admin, dfs, s.delegatePubPEM, time.Hour)
	c.Assert(err, IsNil)
	c.Assert(fakeToken, NotNil)
	addr := "this is more than 6 bytes"
	fakeSigner, err := auth.RSASignerFromPEM(s.delegatePrivPEM)
	c.Assert(err, IsNil)
	_, err = auth.BuildAuthMuxHeader([]byte(addr), fakeToken, fakeSigner)
	c.Assert(err, Equals, auth.ErrBadMuxAddress)
}

func (s *TestAuthSuite) TestExtractBadHeader(c *C) {
	mockHeader := []byte{0, 0, 0, 19, 109, 121, 32, 115, 117, 112, 101, 114, 32, 102}
	_, _, err := auth.ExtractMuxHeader(mockHeader)
	c.Assert(err, Equals, auth.ErrBadMuxHeader)
}

func (s *TestAuthSuite) TestBuildAndExtractHeader(c *C) {
	fakeToken, err := auth.CreateJWTIdentity(hostId, poolId, admin, dfs, s.delegatePubPEM, time.Hour)
	c.Assert(err, IsNil)
	fakeSigner, err := auth.RSASignerFromPEM(s.delegatePrivPEM)
	c.Assert(err, IsNil)
	c.Assert(fakeToken, NotNil)
	c.Assert(fakeSigner, NotNil)
	addr := "zenoss"
	// build header
	header, err := auth.BuildAuthMuxHeader([]byte(addr), fakeToken, fakeSigner)
	c.Assert(err, Equals, nil)
	c.Assert(header, NotNil)
	// extract header
	extractedAddr, ident, err := auth.ExtractMuxHeader(header)
	// check the address is correctly decoded
	c.Assert(err, IsNil)
	c.Assert(string(extractedAddr), DeepEquals, addr)
	// check the identity has been correctly extracted
	c.Assert(hostId, DeepEquals, ident.HostID())
	c.Assert(poolId, DeepEquals, ident.PoolID())
	c.Assert(admin, Equals, ident.HasAdminAccess())
	c.Assert(dfs, Equals, ident.HasDFSAccess())
}
