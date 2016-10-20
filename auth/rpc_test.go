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
	"encoding/binary"
	"github.com/control-center/serviced/auth"
	. "gopkg.in/check.v1"
)

var (
	endian           = binary.BigEndian
	rpcHeaderHandler = &auth.RPCHeaderHandler{}
)

func (s *TestAuthSuite) TestAuthenticated(c *C) {
	var conn bytes.Buffer
	request := []byte("request body")

	err := rpcHeaderHandler.WriteHeader(&conn, request, true)
	c.Assert(err, IsNil)

	// extract header
	ident, body, err := rpcHeaderHandler.ReadHeader(&conn)
	c.Assert(err, IsNil)
	// check the identity has been correctly extracted
	c.Assert(body, DeepEquals, request)
	c.Assert(s.hostId, DeepEquals, ident.HostID())
	c.Assert(s.poolId, DeepEquals, ident.PoolID())
	c.Assert(s.admin, Equals, ident.HasAdminAccess())
	c.Assert(s.dfs, Equals, ident.HasDFSAccess())
}

func (s *TestAuthSuite) TestNotAuthenticated(c *C) {
	var conn bytes.Buffer
	request := []byte("request body")

	err := rpcHeaderHandler.WriteHeader(&conn, request, false)
	c.Assert(err, IsNil)

	// extract header
	ident, body, err := rpcHeaderHandler.ReadHeader(&conn)
	c.Assert(err, IsNil)
	c.Assert(body, DeepEquals, request)
	// Check that we dont have an ident
	c.Assert(ident, IsNil)
}

func (s *TestAuthSuite) TestBadRPCMagicNumber(c *C) {
	var b bytes.Buffer

	pl := []byte("payload")
	b.Write([]byte{13})
	err := rpcHeaderHandler.WriteHeader(&b, pl, false)
	c.Assert(err, IsNil)

	sender, payload, err := rpcHeaderHandler.ReadHeader(&b)
	c.Assert(err, Equals, auth.ErrBadRPCHeader)
	c.Assert(sender, IsNil)
	c.Assert(payload, IsNil)
}
